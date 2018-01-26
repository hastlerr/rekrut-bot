package main

import (
	"log"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"net/http"
	"time"
	"encoding/json"
)

var (
	myClient = &http.Client{Timeout: 10 * time.Second}
	baseUrl = "https://rekrut-smarty.herokuapp.com/"
	telegramBotToken = "500044653:AAGOcDZBcSA_dMMhDz4KhguNTBKwNktHbmI"
	HelpMsg    = "Это бот для получения вакансий. Он стучится на rekrut.kg и высирает вакансии " +
		"Список доступных комманд:\n" +
		"/vacancies - выдаст список вакансий\n" +
		"/help - отобразить это сообщение\n" +
		"\n"
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

		if update.Message == nil {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		switch update.Message.Command() {
		case "vacancies":
			var vacancies []Vacancy
			getVacancies(baseUrl + "api/v1/vacancies.json?page=1", vacancies)
			log.Print(vacancies)

			for _, vacancy := range vacancies {
				reply += vacancy.toString() + "\n\n\n"
			}

		case "vacancies_with_filter":
			reply = "vacancies_with_filter"

		case "help":
			reply = HelpMsg

		default:
			reply = "Данной команды нет"
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, reply)
		bot.Send(msg)
	}
}

func getVacancies(url string, target interface{}) error {
	r, err := myClient.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.Unmarshal([]byte(r), target)
}

type Vacancy struct {
	title		 	string
	phone_numbers 	string
	salary 			string
	short_description string
}

func (this Vacancy) toString() string {
	return this.title + "\n" + this.short_description + "\n" + this.salary + "\n" + this.phone_numbers + "\n"
}
