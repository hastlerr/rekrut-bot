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
	// Pagination
	nextPage     = "Следующая страница"
	previousPage = "Предыдущая страница"
	// Parameters
	category	 = "Категория"
	workTime	 = "Рабочий график"
	vilka	 = "Вилка"
	//Categories
	it = "Айти"
	gos = "Гос"
	//Work time
	fullTime = "Полный рабочий день"
	partTime = "Не полный рабочий день"
)

var paginationKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton(previousPage),
		tgbotapi.NewKeyboardButton(nextPage),
	),

)

var filterParametersKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton(category),
		tgbotapi.NewKeyboardButton(workTime),
		tgbotapi.NewKeyboardButton(vilka),
	),
)

var categoriesKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton(it),
		tgbotapi.NewKeyboardButton(gos),
	),
)

var workTimeKeyboard= tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton(fullTime),
		tgbotapi.NewKeyboardButton(partTime),
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
		var user = cache[userID]

		log.Printf("[%d] %s", userID, update.Message.Text)
		log.Printf("Command: %s", update.Message.Command())

		switch update.Message.Command() {
		case "vacancies":
			if update.Message.CommandArguments() == ""{
				reply, replyMarkup = sendVacancies(user, update, bot)
			} else {
				user.page = 1
				user.searchText = update.Message.CommandArguments()
				log.Printf(cache[userID].searchText)
				reply, replyMarkup = sendVacancies(user, update, bot)
			}
		case "setFilter":
			reply = "Выберите параметр фильтрации"
			replyMarkup = filterParametersKeyboard
		case "resetFilter":
			reply = "Вы сбросили филтры"
			replyMarkup = tgbotapi.NewHideKeyboard(true)
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
			// Обработка кнопок
			switch update.Message.Text {
			case nextPage:
				user.page += 1
				reply, replyMarkup = sendVacancies(user, update, bot)
			case previousPage:
				if user.page > 1 {
					user.page -= 1
					reply, replyMarkup = sendVacancies(user, update, bot)
				} else {
					reply = "Невозможно показать предыдущую страницу"
				}
			case category:
				reply = "Выберите категорию"
				replyMarkup = categoriesKeyboard
			case workTime:
				reply = "Выберите график работы"
				replyMarkup = workTimeKeyboard
			case vilka:

			default:
				reply = "Данной команды нет"
			}

		}
		cache[userID] = user
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, reply)
		msg.ReplyMarkup = replyMarkup
		bot.Send(msg)
	}
}

// Helper

func sendVacancies(user UserConfigurations, update tgbotapi.Update, bot *tgbotapi.BotAPI) (string, interface{}) {

	var url = fmt.Sprintf("%s/api/v1/vacancies.json?page=%d", baseUrl, user.page)
	if user.searchText != ""{
		url = fmt.Sprintf("%s&search=%s", url, user.searchText)
	}
	fmt.Println(url)
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
	return fmt.Sprintf("Список вакансий по вашему запросу выведен \nСтраница %d", user.page), paginationKeyboard
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


// Model

type UserConfigurations struct {
	page       int
	isFilterSearch bool
	searchText string
	category   string
	salaryFrom int
	salaryTo   int
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


// Extension

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
