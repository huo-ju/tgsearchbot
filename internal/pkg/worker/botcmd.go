package worker

import (
    "fmt"
    "strings"
    "strconv"
    "regexp"
	"net/url"
    "time"
    "math"
	"github.com/golang/glog"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/virushuo/tgsearchbot/internal/pkg/service"
	"github.com/virushuo/tgsearchbot/pkg/cypress"
)


// TimerDeleteMessage will call the function f after n seconds
func TimerDeleteMessage(n int, f func()) (*time.Timer){
    timer := time.AfterFunc(time.Duration(n ) * time.Second, f)
    return timer
}

// deleteMessage will delete the telegram bot message  
func deleteMessage(tgservice *service.Telegram, chatID int64, messageID int) {
	deleteMessageConfig := tgbotapi.DeleteMessageConfig{
		ChatID:    chatID,
		MessageID: messageID,
	}
	_, err := tgservice.Bot.DeleteMessage(deleteMessageConfig)
    if err != nil {
	    glog.Errorf("Telegram DeleteMessage error: %v\n", err)
    }
}


type TGBotCommandConf struct {
    DeleteAfterSeconds int
    ResultPerPage int
}


// TGBotCommand telegram bot command processor
func TGBotButtonQuery(tgservice *service.Telegram, conf *TGBotCommandConf, cypressapi *cypress.API, query *tgbotapi.CallbackQuery) {
    fmt.Println("====query button")
    fmt.Println(query.Data)
    fmt.Println(query.Message.MessageID)

    callbackcmd := strings.Split(query.Data, "_")
    if len(callbackcmd) >=3 {
        fmt.Println("callback type:")
        fmt.Println(callbackcmd[0])
        switch callbackcmd[0] {
            case "P":
                if len(callbackcmd) ==4 {
                    pageCount,_ := strconv.Atoi(callbackcmd[3])
                    currentPage,_ := strconv.Atoi(callbackcmd[2])
                    if callbackcmd[1] == "N" && currentPage < pageCount-1 {
                        fmt.Println("pagineation...Next page", currentPage+1)
                        //replymsg := runSearch(message.Text[3:], currentPage, message.Chat.ID, message.MessageID, conf, cypressapi)
                    }
                }
            default:
				glog.V(2).Infof("Unknown Query: %v", query)
        }
    }
}

func runSearch(querystring string, page int, chatID int64, messageID int, conf *TGBotCommandConf, cypressapi *cypress.API) (message *tgbotapi.MessageConfig) {
    num := conf.ResultPerPage
    start := page * conf.ResultPerPage
    queryword := querystring
    restrict := make(map[string]string)

    querycmds := strings.Split(querystring, " ")

    //queryparams[0]= "q=" + url.QueryEscape(strings.Trim(querystring, " "))
    if len(querycmds)>0 {
        match, _ := regexp.MatchString("uid:[0-9]+\\s*", querycmds[0])
        if match == true {
            queryword =  url.QueryEscape(strings.Trim(querystring[len(querycmds[0]):]," "))
            restrict["userid"] = url.QueryEscape(strings.Trim(querycmds[0][4:], " "))
        } else {
            match, _ := regexp.MatchString("name:@{0,1}[0-9a-zA-Z]+\\s*", querycmds[0])
            if match == true {
                queryword =  url.QueryEscape(strings.Trim(querystring[len(querycmds[0]):]," "))
                username := strings.Trim(querycmds[0][5:], " ")
                if username[0] == '@' {
                    restrict["username"] = url.QueryEscape(username[1:])
                }else {
                    restrict["username"] = url.QueryEscape(username)
                }
            }
        }
    }

    //SearchWithClause
    params := make(map[string]string)
    params["cy_termmust"] = "true"
    tenantid := cypress.TGChatID2TanantID(chatID)
    restrict["cy_tenantid"]=tenantid
    clause := &cypress.SearchClause{Queryword: queryword, Conf: "default", Start:start, Num:num, Restrict:&restrict, Params:&params}

    result,err := cypressapi.SearchWithClause(clause, chatID)
    fmt.Println(result)
//type SearchClause struct {
//    Queryword string
//    Conf string
//    Start int
//    Num int
//    Restrict map[string]string
//    Params map[string]string
//}

    //result, err := cypressapi.Search(queryword , start, num , chatID)
    if err != nil {
        glog.Errorf("cypressapi Search error: %v\n", err)
    }
    outputresult := FormatSearchResult(result, conf.ResultPerPage)
    replymsg := tgbotapi.NewMessage(chatID, outputresult)
    replymsg.ReplyToMessageID = messageID
    replymsg.ParseMode = "HTML"
    //pagination counting
    pageCount := 1
    fmt.Printf(" currentpage : %d , all page count: %d \n", page, pageCount)
    if result.Count > conf.ResultPerPage { //show pagination
        pageCount = int(math.Ceil(float64(result.Count) / float64(conf.ResultPerPage)))
        inlinekeyboard := makePaginationKeyboard(pageCount, page)
        replymsg.ReplyMarkup = inlinekeyboard
    }
    return &replymsg
}


// TGBotCommand telegram bot command processor
func TGBotCommand(tgservice *service.Telegram, conf *TGBotCommandConf, cypressapi *cypress.API, message *tgbotapi.Message) {
	if strings.HasPrefix(message.Text , "/s ") == true {
        currentPage := 0
        replymsg := runSearch(message.Text[3:], currentPage, message.Chat.ID, message.MessageID, conf, cypressapi)
        msg, err := tgservice.Bot.Send(replymsg)
        if err != nil{
	        glog.Errorf("Telegram Send message error: %v\n", err)
        } else {
            glog.V(3).Infof("Start the delete timer : %d on chatID %d MessageId %d", conf.DeleteAfterSeconds, msg.Chat.ID, msg.MessageID)
            TimerDeleteMessage(conf.DeleteAfterSeconds, func(){deleteMessage(tgservice, msg.Chat.ID, msg.MessageID)})
        }
    }
}

// FormatSearchResult format the search result and build the message text for telegram reply
func FormatSearchResult(result *cypress.Result, resultPerPage int) string {
    builder := strings.Builder{}
    if len(result.Items) ==0 {
        return "No results found."
    }
    for idx , item := range result.Items {
        if idx >= resultPerPage {
            break
        }
        humanTimestr := ""
        timestamp , err := strconv.ParseInt(item.Date, 10, 64)
        if err == nil {
            t := time.Unix(timestamp, 0)
            humanTimestr = t.Format("2006-01-02 15:04:05")
        }

        title := strings.Replace(item.Title, "<span class='yx_hl'>", "<b>", -1)
        title = strings.Replace(title, "</span>", "</b>", -1)

        tagline := ""
        if len(item.UserName) > 0 {
            tagline += "["+ item.UserName +"]"
        }
        if len(item.URI) > 0 {
            tagline += fmt.Sprintf(" - <a href=\"%s\">%s</a>", item.URI, humanTimestr)
        } else {
            tagline += " - " + humanTimestr
        }

        itemstr := fmt.Sprintf("%d. %s %s \n", idx, title, tagline)
        builder.WriteString(itemstr)
    }

    return builder.String()
}

// makePaginationKeyboard make a Pagination keyboard
func makePaginationKeyboard(pageCount int, currentPage int) tgbotapi.InlineKeyboardMarkup {
	var keyboard [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton

    labelPrev := "<<"
    if currentPage == 0 {
        labelPrev = ""
    }
	buttonPrev := tgbotapi.NewInlineKeyboardButtonData(labelPrev, fmt.Sprintf("P_P_%d_%d", currentPage, pageCount))
	row = append(row, buttonPrev)

    labelNext := ">>"
    if currentPage == pageCount-1 {
        labelNext = ""
    }
	buttonNext := tgbotapi.NewInlineKeyboardButtonData(labelNext, fmt.Sprintf("P_N_%d_%d", currentPage, pageCount))
	row = append(row, buttonNext)
	keyboard = append(keyboard, row)
	return tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: keyboard,
	}
}
