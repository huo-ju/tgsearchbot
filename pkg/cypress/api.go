package cypress
import (
    "fmt"
    "net/http"
	"net/url"
    "strconv"
    "regexp"
    "strings"
    "errors"
    "encoding/json"
    "io/ioutil"
	"bytes"
	"encoding/xml"
	"github.com/golang/glog"
)

const (
    xmlHeader = `<?xml version="1.0" encoding="UTF-8"?>`
)

// API :Cypress API struct
// Endpoint: cypress api endpoint
// TermMustMode = true: add cy_termmust=true to the cypress api request url url
type API struct {
    Endpoint string
    TermMustMode bool
}

// SearchResult is the Cypress Search result top node
type SearchResult struct {
    Result Result `json:"result"`
}

// Result is the child node of the top node
type Result struct {
    Spellcorrect string `json:"spellcorrect"`
    Segment string `json:"segment"`
    Count int `json:"count"`
    Querywords string `json:"querywords"`
    Time int `json:"time"`
    Items []Item `json:"items"`
}

// Item is search hit item node
type Item struct {
    Date string `json:"date"`
    Match float32 `json:"cypress.match"`
    IsReply string `json:"isreply"`
    MesssageID string `json:"messsageid"`
    ChatID string `json:"chatid"`
    Title string `json:"title"`
    URI string `json:"uri"`
    UserID string `json:"userid"`
    UserName string `json:"username"`
    Content string `json:"content"`
}


// Update send the ChatDocument to the cypress update API
func (api *API) Update(doc *ChatDocument) {
	buf := new(bytes.Buffer)
	enc := xml.NewEncoder(buf)
	enc.Indent("", "\t")
	err := enc.Encode(doc)
	if err != nil {
	    glog.Errorf("Update data xml encode err: %v\n", err)
	}
    result, err := httpPost(api.Endpoint+"/updatert",xmlHeader+buf.String())
	if err != nil {
	    glog.Errorf("http post err: %s %v\n", xmlHeader+buf.String(), err)
    }
	glog.V(2).Infof("post result : %v", result)
}

// Search : request the search result from cypress serach API
func (api *API) Search(querystring string, chatID int64) (*Result, error){
    queryparams := make([]string, 3)
    querycmds := strings.Split(querystring, " ")
    queryparams[0]= "q=" + url.QueryEscape(strings.Trim(querystring, " "))
    if len(querycmds)>0 {
        match, _ := regexp.MatchString("uid:[0-9]+\\s*", querycmds[0])
        if match == true {
            queryparams[0]= "q=" + url.QueryEscape(strings.Trim(querystring[len(querycmds[0]):]," "))
            queryparams[1]= "userid=" + url.QueryEscape(strings.Trim(querycmds[0][4:], " "))
        } else {
            match, _ := regexp.MatchString("name:[0-9a-zA-Z]+\\s*", querycmds[0])
            if match == true {
                queryparams[0]= "q=" + url.QueryEscape(strings.Trim(querystring[len(querycmds[0]):]," "))
                queryparams[1]= "username=" + url.QueryEscape(strings.Trim(querycmds[0][5:], " "))
            }
        }
    }
    //TOFIX: for debug
    //tenantid := TGChatID2TanantID(chatID)
    //queryparams[2]= "cy_tenantid=" + tenantid

    var finalqueryparams []string
    for _, str := range queryparams {
        if str != "" {
            finalqueryparams = append(finalqueryparams, str)
        }
    }
    if api.TermMustMode == true{
        finalqueryparams = append(finalqueryparams, "cy_termmust=true")
    }

    apiRequestURL := fmt.Sprintf("%s/search?%s",api.Endpoint, strings.Join(finalqueryparams[:], "&"))
	glog.V(1).Infof("Request cypress : %s", apiRequestURL)
	body, err := httpGet(apiRequestURL)
    if err != nil {
	    glog.Errorf("search error: %v\n", err)
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
