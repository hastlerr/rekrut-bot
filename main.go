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
	page             = map[int]int{}
	baseUrl          = "https://rekrut-smarty.herokuapp.com/"
	telegramBotToken = "500044653:AAGOcDZBcSA_dMMhDz4KhguNTBKwNktHbmI"
	HelpMsg          = "Это бот для получения вакансий. Он стучится на rekrut.kg и высирает вакансии " +
		"Список доступных комманд:\n" +
		"/vacancies - выдаст список вакансий\n" +
		"/help - отобразить это сообщение\n" +
		"\n"
)

const (
	nextPage     = "Следующая страница"
	previousPage = "Предыдущая страница"
)

var numericKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton(previousPage),
		tgbotapi.NewKeyboardButton(nextPage),
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

		var userName = update.Message.From.ID
		log.Printf("[%d] %s", userName, update.Message.Text)

		log.Printf("Command: %s", update.Message.Command())

		switch update.Message.Command() {
		case "vacancies":
			reply, replyMarkup = sendVacancies(0, update, bot)
		case "vacancies_with_filter":
			reply = "vacancies_with_filter"

		case "help":
			reply = HelpMsg

		case "start":
			page[userName] = 1
			name := update.Message.From.UserName
			if update.Message.From.FirstName != "" {
				name = update.Message.From.FirstName
			}
			reply = "Добро пожаловать " + name
			replyMarkup = tgbotapi.NewHideKeyboard(true)

		default:
			switch update.Message.Text {
			case nextPage:
				currentPage := getUserPage(userName)
				page[userName] = currentPage + 1
				reply, replyMarkup = sendVacancies(page[userName], update, bot)
			case previousPage:
				currentPage := getUserPage(userName)
				if currentPage != 0 {
					page[userName] = currentPage - 1
					reply, replyMarkup = sendVacancies(page[userName], update, bot)
				} else {
					reply = "Невозможно показать предыдущую страницу"
				}
			default:
				reply = "Данной команды нет"
			}
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, reply)
		msg.ReplyMarkup = replyMarkup
		bot.Send(msg)
	}
}
func sendVacancies(currentPage int, update tgbotapi.Update, bot *tgbotapi.BotAPI) (string, interface{}) {
	vacancies, err := getVacancies(fmt.Sprintf("%s/api/v1/vacancies.json?page=%d", baseUrl, currentPage))
	if err != nil {
		return err.Error(), nil
	}
	log.Print(len(vacancies))
	for _, vacancy := range vacancies {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, vacancy.toString())
		bot.Send(msg)
	}
	return fmt.Sprintf("Список вакансий по вашему запросу выведен \n Страница %d", currentPage+1), numericKeyboard
}
func getUserPage(userID int) int {
	val, ok := page[userID]
	if ok {
		return val
	} else {
		return 0
	}
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
	return vacancy.Title + "\n" + vacancy.ShortDescription + "\n" + vacancy.Salary + "\n" + vacancy.PhoneNumbers + "\n"
}
