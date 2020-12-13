package main

import (
	"flag"
    "strings"
	"path/filepath"
	"github.com/spf13/viper"
	"github.com/golang/glog"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"

	"github.com/virushuo/tgsearchbot/internal/pkg/service"
	"github.com/virushuo/tgsearchbot/internal/pkg/worker"
	"github.com/virushuo/tgsearchbot/pkg/cypress"
)

var (
	botToken string
    searchAPIEndPoint string
    termMustMode bool
    deleteAfterSecond int
    resultPerPage int
    cypressapi *cypress.API
    tgservice *service.Telegram
    botcmdworker *worker.BotCmdWorker
)

func loadconf() {
	viper.AddConfigPath(filepath.Dir("./config/"))
	viper.AddConfigPath(filepath.Dir("."))
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.ReadInConfig()
	botToken = viper.GetString("BOT_TOKEN")
	searchAPIEndPoint = viper.GetString("SEARCHAPI_ENDPOINT")
	termMustMode = viper.GetBool("TERM_MUST_MODE")
	deleteAfterSecond = viper.GetInt("DELETE_AFTER_SECOND")
	resultPerPage = viper.GetInt("RESULT_PERPAGE")
}

func readInputMessageChannel(ch chan interface{}) {
	for {
        p := <-ch
        switch p := p.(type) {
            case tgbotapi.Message:
				if strings.HasPrefix(p.Text, "/") == true { //It's a bot command
                    go botcmdworker.TGBotCommand(tgservice, &worker.TGBotCommandConf{DeleteAfterSeconds: deleteAfterSecond, ResultPerPage:resultPerPage }, cypressapi, &p)
                } else {
                    doc := cypress.TelegramMessageToDocument(&p)
                    go cypressapi.Update(doc)
                }
            case tgbotapi.CallbackQuery:
                go botcmdworker.TGBotButtonQuery(tgservice, &worker.TGBotCommandConf{DeleteAfterSeconds: deleteAfterSecond, ResultPerPage:resultPerPage }, cypressapi, &p)
            default:
				glog.V(2).Infof("received: %v", p)
        }
    }
}

func main() {
	flag.Parse()
	glog.V(2).Infof("Service Start...")
	loadconf()
    cypressapi = &cypress.API{Endpoint: searchAPIEndPoint,TermMustMode: termMustMode }
    botcmdworker = worker.NewBotCmdWorker()
    var err error
    tgservice,err = service.NewTelegramService(botToken)
    if err != nil {
	    glog.Errorf("Telegram service error: %v\n", err)
    }

    var chtgmsg chan interface{} = make(chan interface{})
	go readInputMessageChannel(chtgmsg)
    tgservice.Startpolling(chtgmsg)
}
