package main

import (
	"log"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"net/http"
	"time"
	"encoding/json"
	"io/ioutil"
	"fmt"
)

var (
	myClient         = &http.Client{Timeout: 10 * time.Second}
	cache            = map[int]UserConfigurations{}
	baseUrl          = "https://rekrut-smarty.herokuapp.com"
	siteUrl          = "http://rekrut.smartylab.net"
	telegramBotToken = "500044653:AAGOcDZBcSA_dMMhDz4KhguNTBKwNktHbmI"
	HelpMsg          = "Это бот для получения вакансий. Он стучится на rekrut.kg и высирает вакансии " +
		"Список доступных комманд:\n" +
		"/vacancies - выдаст список вакансий\n" +
		"/vacancies_with_filter - позволит отфильтровать вывод вакансий\n" +
		"/help - отобразить это сообщение\n" +
		"\n"
)

const (
	nextPage     = "Следующая страница"
	previousPage = "Предыдущая страница"
	textSearch   = "Текстовый поиск"
	filterSearch = "Поиск по фильтру"

)

var paginationKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton(previousPage),
		tgbotapi.NewKeyboardButton(nextPage),
	),

)

var searchType = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton(textSearch),
		tgbotapi.NewKeyboardButton(filterSearch),
	),
)


func main() {
	bot, err := tgbotapi.NewBotAPI(telegramBotToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		reply := ""
		var replyMarkup interface{}

		if update.Message == nil {
			continue
		}

		var userID = update.Message.From.ID
		log.Printf("[%d] %s", userID, update.Message.Text)
		log.Printf("Command: %s", update.Message.Command())

		switch update.Message.Command() {
		case "vacancies":
			reply, replyMarkup = sendVacancies(1, update, bot, cache[userID])
		case "vacancies_with_filter":
			reply = "Выберите тип фильтрации"
			replyMarkup = searchType

		case "help":
			reply = HelpMsg

		case "start":

			cache[userID] = UserConfigurations{page:1}
			name := update.Message.From.UserName
			if update.Message.From.FirstName != "" {
				name = update.Message.From.FirstName
			}
			reply = "Добро пожаловать " + name
			replyMarkup = tgbotapi.NewHideKeyboard(true)

		default:
			user := cache[userID]

			switch update.Message.Text {

			case nextPage:
				user.page += 1
				reply, replyMarkup = sendVacancies(user.page, update, bot, user)
			case previousPage:
				if user.page > 1 {
					user.page -= 1
					reply, replyMarkup = sendVacancies(user.page, update, bot, user)
				} else {
					reply = "Невозможно показать предыдущую страницу"
				}
			case textSearch:
				reply = "Введите ключевое слово"
				user.searchType = textSearch
				replyMarkup = tgbotapi.NewHideKeyboard(true)

			case filterSearch:
				user.searchType = filterSearch
				reply = "Выберите категорию"

			default:
				if user.searchType == textSearch {
					user.searchType = ""
					user.page = 1
					user.searchText = update.Message.Text
					log.Printf(cache[userID].searchText)
					reply, replyMarkup = sendVacancies(user.page, update, bot, user)
				} else {
					reply = "Данной команды нет"
				}
			}
			cache[userID] = user

		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, reply)
		msg.ReplyMarkup = replyMarkup
		bot.Send(msg)
	}
}
func sendVacancies(currentPage int, update tgbotapi.Update, bot *tgbotapi.BotAPI, user UserConfigurations) (string, interface{}) {

	url := fmt.Sprintf("%s/api/v1/vacancies.json?page=%d", baseUrl, currentPage)
	if user.searchText != ""{
		url = fmt.Sprintf("%s&search=%s", url, user.searchText)
		fmt.Printf(url)
	}
	vacancies, err := getVacancies(url)
	fmt.Print(vacancies)
	if err != nil {
		return err.Error(), nil
	}
	log.Print(len(vacancies))
	for _, vacancy := range vacancies {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, vacancy.toString())
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
	}
	return fmt.Sprintf("Список вакансий по вашему запросу выведен \nСтраница %d", currentPage), paginationKeyboard
}


func getVacancies(url string) ([]Vacancy, error) {
	r, err := myClient.Get(url)
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	defer r.Body.Close()

	var result Result
	json.Unmarshal(b, &result)

	return result.Vacancies, nil
}

type UserConfigurations struct {
	page       int
	searchType string
	searchText string
	category   string
	priceStart int
	priceStop  int
	isDollar   bool

}

type Result struct {
	Vacancies []Vacancy `json:"vacancies"`
	Count     int       `json:"count"`
}

type Vacancy struct {
	Id               int    `json:"id"`
	Title            string `json:"title"`
	PhoneNumbers     string `json:"phone_numbers"`
	Salary           string `json:"salary"`
	ShortDescription string `json:"short_description"`
}

func (vacancy Vacancy) toString() string {
	return fmt.Sprintf("*%s*\n" +
		"\n"+
		"%s\n"+
		"%s\n"+
		"%s\n"+
		"%s/#/job/%d",
		vacancy.Title,
		vacancy.ShortDescription,
		vacancy.Salary,
		vacancy.PhoneNumbers,
		siteUrl, vacancy.Id)
}
