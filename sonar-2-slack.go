package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"

	"github.com/go-resty/resty"
)

// ServerInfor read from env
type ServerInfor struct {
	Username     string
	Password     string
	SonarURL     string
	SlackHookURL string
	ProjectName  string
	SlackChanel  string
}

// SonarStatus Response from json
type SonarStatus struct {
	Component struct {
		ID       string `json:"id"`
		Key      string `json:"key"`
		Name     string `json:"name"`
		Measures []struct {
			Metric string `json:"metric"`
			Value  string `json:"value"`
		} `json:"measures"`
	} `json:"component"`
}

// NotifContent to push to slack
type NotifContent struct {
	ID                     string
	Key                    string
	Name                   string
	Color                  string
	AlertStatus            string
	Status                 string
	SqaleIndex             string
	Bugs                   string
	DuplicatedLinesDensity string
	CodeSmells             string
}

// Field slack
type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// Attachment slack
type Attachment struct {
	Fallback   string   `json:"fallback"`
	Color      string   `json:"color"`
	PreText    string   `json:"pretext"`
	AuthorName string   `json:"author_name"`
	AuthorLink string   `json:"author_link"`
	AuthorIcon string   `json:"author_icon"`
	Title      string   `json:"title"`
	TitleLink  string   `json:"title_link"`
	Text       string   `json:"text"`
	ImageURL   string   `json:"image_url"`
	Fields     []*Field `json:"fields"`
	Footer     string   `json:"footer"`
	FooterIcon string   `json:"footer_icon"`
	MarkDownIn []string `json:"mrkdwn_in"`
}

// Payload slack
type Payload struct {
	Parse       string       `json:"parse,omitempty"`
	Username    string       `json:"username,omitempty"`
	IconURL     string       `json:"icon_url,omitempty"`
	IconEmoji   string       `json:"icon_emoji,omitempty"`
	Channel     string       `json:"channel,omitempty"`
	Text        string       `json:"text,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

func getEnvironmentVariable() ServerInfor {
	return ServerInfor{
		Username:     os.Getenv("SONAR_USERNAME"),
		Password:     os.Getenv("SONAR_PASSWORD"),
		SlackChanel:  os.Getenv("SLACK_CHANNEL"),
		SlackHookURL: os.Getenv("SLACK_HOOK_URL"),
		ProjectName:  os.Getenv("PROJECT_ALIAS_NAME"),
		SonarURL:     os.Getenv("SONAR_URL"),
	}
}

func fetchState(serverInfor ServerInfor) *resty.Response {
	resq, err := resty.
		R().
		SetQueryParams(map[string]string{
			"metricKeys":   "bugs, duplicated_lines_density, code_smells, alert_status, sqale_index",
			"componentKey": serverInfor.ProjectName,
		}).
		SetBasicAuth(serverInfor.Username, serverInfor.Password).
		Get(serverInfor.SonarURL + "api/measures/component")
	if err != nil {
		return nil
	}
	return resq
}

func convertToNotif(status SonarStatus) (content NotifContent) {
	notifContent := NotifContent{
		ID:   status.Component.ID,
		Key:  status.Component.Key,
		Name: status.Component.Name,
	}
	for _, val := range status.Component.Measures {
		switch val.Metric {
		case "bugs":
			notifContent.Bugs = val.Value
		case "alert_status":
			notifContent.AlertStatus = val.Value
			switch notifContent.AlertStatus {
			case "ERROR":
				notifContent.Color = "danger"
				notifContent.Status = "DANGER"
			case "WARN":
				notifContent.Color = "warning"
				notifContent.Status = "WARNING"
			case "OK":
				notifContent.Color = "good"
				notifContent.Status = "GREAT!"
			}
		case "sqale_index":
			debtHours, err := strconv.ParseFloat(val.Value, 64)
			if err != nil {
				notifContent.SqaleIndex = "0"
			} else {
				debtDay := math.Ceil(debtHours / (60.0 * 8.0))
				notifContent.SqaleIndex = strconv.FormatFloat(debtDay, 'f', -1, 64)
			}
		case "code_smells":
			notifContent.CodeSmells = val.Value
		case "duplicated_lines_density":
			notifContent.DuplicatedLinesDensity = val.Value
		}
	}
	return notifContent
}

func (notifContent NotifContent) text() string {
	return "DANGER \n *" +
		notifContent.Bugs + " bugs*\n Technical debt: *" + notifContent.SqaleIndex +
		" days*\n Duplicated: * " + notifContent.DuplicatedLinesDensity +
		" %*\n * " + notifContent.CodeSmells + " * Code Smells"
}

func manualSendSlack(serverInfor ServerInfor, notifContent NotifContent) {
	attachment := Attachment{
		Color:      notifContent.Color,
		Text:       notifContent.text(),
		AuthorName: "Sonar Qube",
		Title:      "Review code: " + serverInfor.ProjectName,
		MarkDownIn: []string{"text"},
	}
	payload := Payload{
		Channel:     serverInfor.SlackChanel,
		Attachments: []Attachment{attachment},
		Username:    "CI-Bot",
		IconEmoji:   ":monkey_face:",
	}
	resp, err := resty.R().SetBody(payload).Post(serverInfor.SlackHookURL)
	fmt.Println(resp, err)
}

func main() {
	serverInfor := getEnvironmentVariable()
	result := fetchState(serverInfor)
	if result == nil {
		return
	}
	var status SonarStatus
	err := json.Unmarshal(result.Body(), &status)
	if err != nil {
		return
	}
	notifContent := convertToNotif(status)
	manualSendSlack(serverInfor, notifContent)
}
