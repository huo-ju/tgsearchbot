package cypress
import (
    "fmt"
	"strconv"
	"encoding/xml"
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// Title is the Cypress Document Title Node
type Title struct {
    XMLName xml.Name `xml:"title"`
	Text    string   `xml:",cdata"`
}

// Content is the Cypress Document Content Node
type Content struct {
    XMLName xml.Name `xml:"content"`
	Text    string   `xml:",cdata"`
}

// ChatDocument is the Cypress Document Node
type ChatDocument struct{
	XMLName xml.Name `xml:"node"`
	URI string  `xml:"uri"`
	XMLURI string  `xml:"xmluri"`
	Title *Title `xml:"title"`
	Content *Content `xml:"content"`
    IsReply bool `xml:"IsReply"`
	ChatID int64 `xml:"chatid"`
	MessageID int `xml:"messsageid"`
	UserID int `xml:"userid"`
	Date int `xml:"date"`
}

// TelegramMessageToDocument make ChatDocument Instances from Telegram Messages
func TelegramMessageToDocument(msg *tgbotapi.Message) *ChatDocument{
    doc := &ChatDocument{}
	doc.XMLURI = fmt.Sprintf("telegram://%d/%d", msg.Chat.ID, msg.MessageID)
    doc.MessageID = msg.MessageID
    doc.UserID = msg.From.ID
    doc.Title = &Title{Text:msg.Text}
    doc.Content = &Content{}
    doc.ChatID = msg.Chat.ID
    doc.Date = msg.Date
    if msg.ReplyToMessage != nil {
        doc.IsReply = true
    }
	s := strconv.FormatInt(doc.ChatID, 10)
	if len(s)>4 && s[0:4] == "-100" { //public group, we can build the message the uri
		doc.URI = fmt.Sprintf("https://t.me/c/%s/%d",s[4:], doc.MessageID)
	}
    return doc
}
