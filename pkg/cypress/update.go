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

// Date is the Cypress Document Date Node
type Date struct {
    XMLName xml.Name `xml:"date"`
    Type string `xml:"type,attr"`
    Text int `xml:",chardata"`
}

type TenantID struct {
    XMLName xml.Name `xml:"cy_tenantid"`
    Type string `xml:"type,attr"`
    Text string `xml:",chardata"`
}

// ChatDocument is the Cypress Document Node
type ChatDocument struct{
	XMLName xml.Name `xml:"node"`
	TenantID *TenantID `xml:"cy_tenantid"`
	URI string  `xml:"uri"`
	XMLURI string  `xml:"xmluri"`
	Title *Title `xml:"title"`
	Content *Content `xml:"content"`
    IsReply bool `xml:"isreply"`
	ChatID int64 `xml:"chatid"`
	MessageID int `xml:"messsageid"`
	UserID int `xml:"userid"`
    UserName string `xml:"username"`
	Date *Date `xml:"date"`
}

// TelegramMessageToDocument make ChatDocument Instances from Telegram Messages
func TelegramMessageToDocument(msg *tgbotapi.Message) *ChatDocument{
    doc := &ChatDocument{}
	doc.XMLURI = fmt.Sprintf("telegram://%d/%d", msg.Chat.ID, msg.MessageID)
    doc.MessageID = msg.MessageID
    doc.UserID = msg.From.ID
    doc.UserName = msg.From.UserName
    doc.Title = &Title{Text:msg.Text}
    doc.Content = &Content{}
    doc.ChatID = msg.Chat.ID
    doc.Date = &Date{Text:msg.Date, Type:"cypress.int"}
    if msg.ReplyToMessage != nil {
        doc.IsReply = true
    }
	s := strconv.FormatInt(doc.ChatID, 10)
	if len(s)>4 && s[0:4] == "-100" { //public group, we can build the message the uri
		doc.URI = fmt.Sprintf("https://t.me/c/%s/%d",s[4:], doc.MessageID)
        doc.TenantID = &TenantID{Text:fmt.Sprintf("%s.group.telegram", s[4:]), Type:"cypress.untoken"}
	}
	if len(s)>4 && s[0:4] == "-400" { //private group
        doc.TenantID = &TenantID{Text:fmt.Sprintf("%s.group.telegram", s[4:]), Type:"cypress.untoken"}
    }
    return doc
}

