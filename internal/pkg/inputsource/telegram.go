package inputsource

import (
    "fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// Telegram API struct
type Telegram struct {
    BotToken string
}

// Startpolling polling Message from Telegram 
func (tg *Telegram) Startpolling(ch chan interface{}) {
    fmt.Println("startpolling from tgbot")
	bot, err := tgbotapi.NewBotAPI(tg.BotToken)
	if err != nil {
        fmt.Println(err)
	}
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := bot.GetUpdatesChan(u)
	for update := range updates {
		if update.Message != nil {
			if update.Message.Text != "" {
		        ch <- *update.Message
				//if strings.HasPrefix(update.Message.Text, "/") == true { //It's a bot command
                //    fmt.Println("income cmd:", update.Message.Text)
                //} else {
                //    fmt.Println("income msg:", update.Message.Text)
                //}
            }
        }
    }
}
