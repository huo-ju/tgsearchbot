package cypress
import (
    "fmt"
    "net/http"
    "strconv"
    "regexp"
    "strings"
    "errors"
    "encoding/json"
    "io/ioutil"
	"bytes"
	"encoding/xml"
)

const (
    xmlHeader = `<?xml version="1.0" encoding="UTF-8"?>`
)

type API struct {
    Endpoint string
}

type SearchResult struct {
    Result Result `json:"result"`
}

type Result struct {
    Spellcorrect string `json:"spellcorrect"`
    Segment string `json:"segment"`
    Count int `json:"count"`
    Querywords string `json:"querywords"`
    Time int `json:"time"`
    Items []Item `json:"items"`
}

type Item struct {
    Date string `json:"date"`
    Match float32 `json:"cypress.match"`
    IsReply string `json:"isreply"`
    MesssageID string `json:"messsageid"`
    ChatID string `json:"chatid"`
    Title string `json:"title"`
    URI string `json:"uri"`
    UserID string `json:"userid"`
    Content string `json:"content"`
}


func (api *API) Update(doc *ChatDocument) {
	buf := new(bytes.Buffer)
	enc := xml.NewEncoder(buf)
	enc.Indent("", "\t")
	err := enc.Encode(doc)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println()
    result, err := httpPost(api.Endpoint+"/updatert",xmlHeader+buf.String())
    fmt.Println(result)
    fmt.Println(err)
}

func (api *API) Search(querystring string, chatID int64) (*Result, error){
    queryparams := make([]string, 3)
    querycmds := strings.Split(querystring, " ")
    queryparams[0]= "q=" + strings.Trim(querystring, " ")
    if len(querycmds)>0 {
        match, _ := regexp.MatchString("from:[0-9]+\\s*", querycmds[0])
        if match == true {
            queryparams[0]= "q=" + strings.Trim(querystring[len(querycmds[0]):]," ")
            queryparams[1]= "userid=" + strings.Trim(querycmds[0][5:], " ")
        }
    }
    tenantid := TGChatID2TanantID(chatID)
    queryparams[2]= "cy_tenantid=" + tenantid
    apiRequestURL := fmt.Sprintf("%s/search?%s",api.Endpoint, strings.Join(queryparams[:], "&"))
	body, err := httpGet(apiRequestURL)
    if err!=nil {
        fmt.Println("search error")
    }
    var searchresult SearchResult
    err = json.Unmarshal([]byte(body), &searchresult)
    if err != nil {
        return nil, err
    }
    return &searchresult.Result, nil
}

func httpGet(api string) (string,error) {
    res, err := http.Get(api)
    if err != nil {
		return "",err
    }

    if res.StatusCode != 200 {
        return "", errors.New("Status: "+ res.Status)
    }
    defer res.Body.Close()

    body, err := ioutil.ReadAll(res.Body)
    if err != nil {
		return "",err
    }
    return string(body), nil
}

func httpPost(api string, data string) (string,error) {
    client := &http.Client{}
    r, err := http.NewRequest("POST", api, strings.NewReader(data))
    if err != nil {
        return "",err
    }
    r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
    r.Header.Add("Content-Length", strconv.Itoa(len(data)))

    res, err := client.Do(r)
    if err != nil {
        return "",err
    }
    if res.StatusCode != 200 {
        return "", errors.New("Status: "+ res.Status)
    }
    defer res.Body.Close()
    body, err := ioutil.ReadAll(res.Body)
    if err != nil {
        return string(body), err
    }
    return string(body), nil
}
