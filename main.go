package main

import (
	"log"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"net/http"
	"time"
	"encoding/json"
	"io/ioutil"
	"fmt"
	"strings"
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
				user.searchText = fmt.Sprintf("&search=%s", update.Message.CommandArguments()[1:])
				log.Printf(cache[userID].searchText)
				reply, replyMarkup = sendVacancies(user, update, bot)
			}
		case "setFilter":
			reply = "Выберите параметр фильтрации"
			replyMarkup = filterParametersKeyboard
		case "resetFilter":
			reply = "Вы сбросили филтры"
			user.resetFilter(userID)
			replyMarkup = tgbotapi.NewHideKeyboard(true)
		case "help":
			reply = HelpMsg

		case "start":
			cache[userID] = UserConfigurations{page:1}
			cache[userID].resetFilter(userID)
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

			case it, gos:
				user.category = fmt.Sprintf("&category=%s", update.Message.Text)
				sendMessage(update, bot, "Вы выбрали категорию " + update.Message.Text, tgbotapi.NewHideKeyboard(true))

				reply, replyMarkup = sendVacancies(user, update, bot)

			case partTime, fullTime :
				user.workTime = fmt.Sprintf("&worktime=%s", update.Message.Text)
				sendMessage(update, bot, "Вы выбрали " + update.Message.Text + "график", tgbotapi.NewHideKeyboard(true))
				reply, replyMarkup = sendVacancies(user, update, bot)

			default:
				reply = "Данной команды нет"
			}

		}
		cache[userID] = user
		sendMessage(update, bot, reply, replyMarkup)
	}
}

// Helper

func sendMessage(update tgbotapi.Update, bot *tgbotapi.BotAPI, text string, keyboard interface{})  {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

func valueFromFilter(filter string) string {
	i := strings.Index(filter, "=")
	return filter[i+1:]
}

func getCurrentStatus(user UserConfigurations) string {
	return fmt.Sprintf("Список вакансий по вашему запросу выведен \nСтраница %d %s \nФильтры: %s %s",
		user.page,
		ternary(user.searchText != "", "\nКлючевое слово: " + valueFromFilter(user.searchText), ""),
		valueFromFilter(user.category),
		valueFromFilter(user.workTime))
}

func sendVacancies(user UserConfigurations, update tgbotapi.Update, bot *tgbotapi.BotAPI) (string, interface{}) {

	var url = fmt.Sprintf("%s/api/v1/vacancies.json?page=%d%s%s%s%s%s",
		baseUrl,
		user.page,
		user.searchText,
		user.category,
		user.workTime,
		user.salaryFrom,
		user.salaryTo)

	fmt.Println(url)
	vacancies, err := getVacancies(url)
	fmt.Print(vacancies)
	if err != nil {
		return err.Error(), nil
	}
	log.Print(len(vacancies))
	for _, vacancy := range vacancies {
		sendMessage(update, bot, vacancy.toString(), nil)
	}
	return getCurrentStatus(user), paginationKeyboard
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
	searchText string
	category   string
	salaryFrom string
	salaryTo   string
	workTime   string
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

func (user UserConfigurations) resetFilter(userID int) {
	user.workTime = ""
	user.category = ""
	user.page = 1
	user.salaryFrom = ""
	user.salaryTo = ""
	cache[userID] = user
}

func ternary(statement bool, a, b interface{}) interface{} {
	if statement {
		return a
	}
	return b
}