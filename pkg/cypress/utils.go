package cypress

import (
    "fmt"
	"strconv"
)

func TGChatID2TanantID(chatID int64) string{
    s := strconv.FormatInt(chatID, 10)
	if len(s)>4 && (s[0:4] == "-100" || len(s)>4 && s[0:4] == "-400"){ //public group, we can build the message the uri
        return fmt.Sprintf("%s.group.telegram", s[4:])
	}

    if len(s)>1 && s[0:1] == "-" {
        return fmt.Sprintf("%s.group.telegram", s[1:])
    }
    return fmt.Sprintf("%s.group.telegram", s)
}
