package worker

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	//"encoding/json"
	//"encoding/base64"
	"math"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/golang/glog"
	"github.com/virushuo/tgsearchbot/internal/pkg/service"
	"github.com/virushuo/tgsearchbot/pkg/cypress"
)

// BotCmd worker struct
type BotCmdWorker struct {
	//BotToken string
	QueryCache map[string]*cypress.SearchClause
	TimerCache map[string]*time.Timer
}

func NewBotCmdWorker() *BotCmdWorker {
	cache := make(map[string]*cypress.SearchClause)
	timercache := make(map[string]*time.Timer)
	return &BotCmdWorker{QueryCache: cache, TimerCache: timercache}
}

// TimerDeleteMessage will call the function f after n seconds
func TimerDeleteMessage(n int, f func()) *time.Timer {
	timer := time.AfterFunc(time.Duration(n)*time.Second, f)
	return timer
}

// deleteMessage will delete the telegram bot message
func deleteMessage(tgservice *service.Telegram, botcmdworker *BotCmdWorker, chatID int64, messageID int) {
	deleteMessageConfig := tgbotapi.DeleteMessageConfig{
		ChatID:    chatID,
		MessageID: messageID,
	}
	_, err := tgservice.Bot.DeleteMessage(deleteMessageConfig)
	botcmdworker.UnregTimer(messageID, chatID)
	if err != nil {
		glog.Errorf("Telegram DeleteMessage error: %v\n", err)
	}
}

type TGBotCommandConf struct {
	DeleteAfterSeconds int
	ResultPerPage      int
}

func (botcmdworker *BotCmdWorker) AddCache(messageID int, chatID int64, clause *cypress.SearchClause) {
	key := fmt.Sprintf("%d:%d", chatID, messageID)
	botcmdworker.QueryCache[key] = clause
}

func (botcmdworker *BotCmdWorker) GetFromCache(messageID int, chatID int64) *cypress.SearchClause {
	key := fmt.Sprintf("%d:%d", chatID, messageID)
	return botcmdworker.QueryCache[key]
}

func (botcmdworker *BotCmdWorker) DelCache(messageID int, chatID int64) {
	key := fmt.Sprintf("%d:%d", chatID, messageID)
	delete(botcmdworker.QueryCache, key)
}

func (botcmdworker *BotCmdWorker) RegTimer(messageID int, chatID int64, timer *time.Timer) {
	key := fmt.Sprintf("%d:%d", chatID, messageID)
	botcmdworker.TimerCache[key] = timer
}

func (botcmdworker *BotCmdWorker) UnregTimer(messageID int, chatID int64) {
	key := fmt.Sprintf("%d:%d", chatID, messageID)
	delete(botcmdworker.TimerCache, key)
}

func (botcmdworker *BotCmdWorker) ResetTimer(messageID int, chatID int64, n int) {
	key := fmt.Sprintf("%d:%d", chatID, messageID)
	timer := botcmdworker.TimerCache[key]
	if timer != nil {
		timer.Reset(time.Duration(n) * time.Second)
	}
}

// TGBotCommand telegram bot command processor
func (botcmdworker *BotCmdWorker) TGBotButtonQuery(tgservice *service.Telegram, conf *TGBotCommandConf, cypressapi *cypress.API, query *tgbotapi.CallbackQuery) {
	queryFromID := query.From.ID
	callbackcmd := strings.Split(query.Data, "_")
	if len(callbackcmd) >= 3 {
		switch callbackcmd[0] {
		case "P":
			start, _ := strconv.Atoi(callbackcmd[1])
			fromID, _ := strconv.Atoi(callbackcmd[2])

			go func() {
				if queryFromID == fromID {
					botcmdworker.ResetTimer(query.Message.MessageID, query.Message.Chat.ID, conf.DeleteAfterSeconds)
					editmsg := botcmdworker.runSearchWithPaging(query.Message.Chat.ID, query.Message.MessageID, fromID, start, cypressapi)
					if editmsg != nil {
						tgservice.Bot.Send(editmsg)
					}
				} else {
					alert := tgbotapi.NewCallbackWithAlert(query.ID, "It's not your search result.")
					tgservice.Bot.AnswerCallbackQuery(alert)
					glog.V(2).Infof("Other users %d click the button, ignore.", queryFromID)
				}
			}()
		default:
			glog.V(2).Infof("Unknown Query: %v", query)
		}
	}
}

func (botcmdworker *BotCmdWorker) runSearch(querystring string, page int, chatID int64, messageID int, fromID int, conf *TGBotCommandConf, cypressapi *cypress.API) (*tgbotapi.MessageConfig, *cypress.SearchClause) {
	num := conf.ResultPerPage
	start := page * conf.ResultPerPage
	queryword := querystring
	restrict := make(map[string]string)

	querycmds := strings.Split(querystring, " ")

	//queryparams[0]= "q=" + url.QueryEscape(strings.Trim(querystring, " "))
	if len(querycmds) > 0 {
		match, _ := regexp.MatchString("uid:[0-9]+\\s*", querycmds[0])
		if match == true {
			queryword = url.QueryEscape(strings.Trim(querystring[len(querycmds[0]):], " "))
			restrict["userid"] = url.QueryEscape(strings.Trim(querycmds[0][4:], " "))
		} else {
			match, _ := regexp.MatchString("name:@{0,1}[0-9a-zA-Z]+\\s*", querycmds[0])
			if match == true {
				queryword = url.QueryEscape(strings.Trim(querystring[len(querycmds[0]):], " "))
				username := strings.Trim(querycmds[0][5:], " ")
				if username[0] == '@' {
					restrict["username"] = url.QueryEscape(username[1:])
				} else {
					restrict["username"] = url.QueryEscape(username)
				}
			}
		}
	}

	//SearchWithClause
	params := make(map[string]string)
	params["cy_termmust"] = "true"
	tenantid := cypress.TGChatID2TanantID(chatID)
	restrict["cy_tenantid"] = tenantid
	clause := &cypress.SearchClause{Queryword: queryword, Conf: "default", Start: start, Num: num, Restrict: &restrict, Params: &params}

	result, err := cypressapi.SearchWithClause(clause, chatID)
	if err != nil {
		glog.Errorf("cypressapi Search error: %v\n", err)
	}
	outputresult := FormatSearchResult(clause.Start, result, conf.ResultPerPage)
	replymsg := tgbotapi.NewMessage(chatID, outputresult)
	replymsg.ReplyToMessageID = messageID
	replymsg.ParseMode = "HTML"
	replymsg.DisableWebPagePreview = true
	//pagination counting
	pageCount := 1
	fmt.Printf(" currentpage : %d , all page count: %d \n", page, pageCount)
	if result.Count > conf.ResultPerPage { //show pagination
		pageCount = int(math.Ceil(float64(result.Count) / float64(conf.ResultPerPage)))
		inlinekeyboard := makePaginationKeyboard(fromID, result.Count, clause)
		replymsg.ReplyMarkup = inlinekeyboard
	}
	return &replymsg, clause
}

// (*tgbotapi.MessageConfig)
func (botcmdworker *BotCmdWorker) runSearchWithPaging(chatID int64, messageID int, fromID int, start int, cypressapi *cypress.API) *tgbotapi.EditMessageTextConfig {
	clause := botcmdworker.GetFromCache(messageID, chatID)
	if clause == nil {
		return nil
	}
	clause.Start = start
	result, err := cypressapi.SearchWithClause(clause, chatID)
	if err != nil {
		glog.Errorf("cypressapi Search error: %v\n", err)
	}
	outputresult := FormatSearchResult(clause.Start, result, clause.Num)
	inlinekeyboard := makePaginationKeyboard(fromID, result.Count, clause)
	editmsg := tgbotapi.EditMessageTextConfig{
		BaseEdit: tgbotapi.BaseEdit{
			ChatID:      chatID,
			MessageID:   messageID,
			ReplyMarkup: &inlinekeyboard,
		},
		ParseMode: "HTML",
		Text:      outputresult,
	}
	return &editmsg
}

// TGBotCommand telegram bot command processor
func (botcmdworker *BotCmdWorker) TGBotCommand(tgservice *service.Telegram, conf *TGBotCommandConf, cypressapi *cypress.API, message *tgbotapi.Message) {
	if strings.HasPrefix(message.Text, "/s ") == true {
		currentPage := 0
		replymsg, clause := botcmdworker.runSearch(message.Text[3:], currentPage, message.Chat.ID, message.MessageID, message.From.ID, conf, cypressapi)
		msg, err := tgservice.Bot.Send(replymsg)
		if err != nil {
			glog.Errorf("Telegram Send message error: %v\n", err)
		} else {
			botcmdworker.AddCache(msg.MessageID, msg.Chat.ID, clause)
			glog.V(3).Infof("Start the delete timer : %d on chatID %d MessageId %d", conf.DeleteAfterSeconds, msg.Chat.ID, msg.MessageID)
			timer := TimerDeleteMessage(conf.DeleteAfterSeconds, func() {
				deleteMessage(tgservice, botcmdworker, msg.Chat.ID, msg.MessageID)
				botcmdworker.DelCache(msg.MessageID, msg.Chat.ID)
			})
			botcmdworker.RegTimer(msg.MessageID, msg.Chat.ID, timer)
		}
	}
}

// FormatSearchResult format the search result and build the message text for telegram reply
func FormatSearchResult(start int, result *cypress.Result, resultPerPage int) string {
	builder := strings.Builder{}
	if len(result.Items) == 0 {
		return "No results found."
	}
	if result.Count == 1 {
		builder.WriteString("(1) result\n")
	} else {
		builder.WriteString(fmt.Sprintf("(%d) results\n", result.Count))
	}
	for idx, item := range result.Items {
		if idx >= resultPerPage {
			break
		}
		humanTimestr := ""
		timestamp, err := strconv.ParseInt(item.Date, 10, 64)
		if err == nil {
			t := time.Unix(timestamp, 0)
			humanTimestr = t.Format("2006-01-02 15:04:05")
		}
		if humanTimestr == "" { //patch for NewDate
			timestamp, err = strconv.ParseInt(item.NewDate, 10, 64)
			if err == nil {
				t := time.Unix(timestamp, 0)
				humanTimestr = t.Format("2006-01-02 15:04:05")
			}
		}

		title := strings.Replace(item.Title, "<span class='yx_hl'>", "<b>", -1)
		title = strings.Replace(title, "</span>", "</b>", -1)

		tagline := ""
		if len(item.UserName) > 0 {
			tagline += "[" + item.UserName + "]"
		}
		if len(item.URI) > 0 {
			tagline += fmt.Sprintf(" - <a href=\"%s\">%s</a>", item.URI, humanTimestr)
		} else {
			tagline += " - " + humanTimestr
		}

		itemstr := fmt.Sprintf("%d. %s %s\n", start+idx+1, title, tagline)
		builder.WriteString(itemstr)
	}

	return builder.String()
}

// makePaginationKeyboard make a Pagination keyboard
func makePaginationKeyboard(fromID int, resultCount int, clause *cypress.SearchClause) tgbotapi.InlineKeyboardMarkup {
	var keyboard [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton

	if clause.Start-clause.Num >= 0 { //prev button
		labelPrev := "<<"
		buttonPrev := tgbotapi.NewInlineKeyboardButtonData(labelPrev, fmt.Sprintf("P_%d_%d", clause.Start-clause.Num, fromID))
		row = append(row, buttonPrev)
	}
	if clause.Start+clause.Num < resultCount { //next button
		labelNext := ">>"
		buttonNext := tgbotapi.NewInlineKeyboardButtonData(labelNext, fmt.Sprintf("P_%d_%d", clause.Start+clause.Num, fromID))
		row = append(row, buttonNext)
	}

	keyboard = append(keyboard, row)
	return tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: keyboard,
	}
}
