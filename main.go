package main

import (
	"log"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"net/http"
	"time"
	"encoding/json"
	"io/ioutil"
)


var (
	myClient         = &http.Client{Timeout: 10 * time.Second}
	page = map[string]int { }
	baseUrl          = "https://rekrut-smarty.herokuapp.com/"
	telegramBotToken = "500044653:AAGOcDZBcSA_dMMhDz4KhguNTBKwNktHbmI"
	HelpMsg          = "Это бот для получения вакансий. Он стучится на rekrut.kg и высирает вакансии " +
		"Список доступных комманд:\n" +
		"/vacancies - выдаст список вакансий\n" +
		"/help - отобразить это сообщение\n" +
		"\n"
)

var numericKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Предыдущая страница"),
		tgbotapi.NewKeyboardButton("Следующая страница"),
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

		if update.Message == nil {
			continue
		}

		var userName = update.Message.From.UserName
		log.Printf("[%s] %s", userName , update.Message.Text)

		switch update.Message.Command() {
		case "vacancies":
			vacancies, err := getVacancies("https://rekrut-smarty.herokuapp.com/api/v1/vacancies.json?page=" + string(page[userName]))
			if err != nil {
				reply = err.Error()
				break
			}
			log.Print(len(vacancies))
			for _, vacancy := range vacancies {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, vacancy.toString())
				bot.Send(msg)
			}
			page[userName] += 1
			log.Printf("ssssss")
			log.Printf(string(len(page)))


			reply = "Список вакансий по вашему запросу выведен \n Страница " + string(page[userName])

		case "vacancies_with_filter":
			reply = "vacancies_with_filter"

		case "help":
			reply = HelpMsg


		case "start":
			page[userName] = 1
			reply = "Добро пожаловать " + update.Message.Chat.UserName

		default:
			reply = "Данной команды нет"
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, reply)
		msg.ReplyMarkup = numericKeyboard
		bot.Send(msg)
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

	log.Printf("%s", b)
	defer r.Body.Close()

	var result Result
	json.Unmarshal(b, &result)
	log.Printf("%v", result)
	log.Print(len(result.Vacancies))

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
