package service

import (
	"github.com/golang/glog"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// Telegram API struct
type Telegram struct {
    //BotToken string
    Bot *tgbotapi.BotAPI
}

// NewTelegramService create a service.Telegram instance
func NewTelegramService(botToken string) (*Telegram, error){
	bot, err := tgbotapi.NewBotAPI(botToken)
    if err !=nil {
        return nil, err
    }
    return &Telegram{Bot:bot}, nil
}

// Startpolling polling Message from Telegram 
func (tg *Telegram) Startpolling(ch chan interface{}) {
	glog.Infof("Startpolling from telegram...")
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := tg.Bot.GetUpdatesChan(u)
    if err!=nil {
	    glog.Errorf("Telegram GetUpdates error: %v\n", err)
    }
	for update := range updates {
		if update.Message != nil {
			if update.Message.Text != "" {
		        ch <- *update.Message
            }
        }
    }
}
