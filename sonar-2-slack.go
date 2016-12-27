package main

import (
	"encoding/json"
	"math"
	"strconv"

	"os"

	"github.com/ashwanthkumar/slack-go-webhook"
	"github.com/go-resty/resty"
)

// ServerInfor read from env
type ServerInfor struct {
	username     string
	password     string
	sonarURL     string
	slackHookURL string
	projectName  string
	slackChanel  string
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
	id                     string
	key                    string
	name                   string
	color                  string
	alertStatus            string
	status                 string
	sqaleIndex             string
	bugs                   string
	duplicatedLinesDensity string
	codeSmells             string
}

// Field slack
type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// Attachment slack
type Attachment struct {
	Fallback   *string  `json:"fallback"`
	Color      *string  `json:"color"`
	PreText    *string  `json:"pretext"`
	AuthorName *string  `json:"author_name"`
	AuthorLink *string  `json:"author_link"`
	AuthorIcon *string  `json:"author_icon"`
	Title      *string  `json:"title"`
	TitleLink  *string  `json:"title_link"`
	Text       *string  `json:"text"`
	ImageUrl   *string  `json:"image_url"`
	Fields     []*Field `json:"fields"`
	Footer     *string  `json:"footer"`
	FooterIcon *string  `json:"footer_icon"`
}

// Payload slack
type Payload struct {
	Parse       string       `json:"parse,omitempty"`
	Username    string       `json:"username,omitempty"`
	IconUrl     string       `json:"icon_url,omitempty"`
	IconEmoji   string       `json:"icon_emoji,omitempty"`
	Channel     string       `json:"channel,omitempty"`
	Text        string       `json:"text,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

func getEnvironmentVariable() ServerInfor {
	return ServerInfor{
		username:     os.Getenv("SONAR_USERNAME"),
		password:     os.Getenv("SONAR_PASSWORD"),
		slackChanel:  os.Getenv("SLACK_CHANNEL"),
		slackHookURL: os.Getenv("SLACK_HOOK_URL"),
		projectName:  os.Getenv("PROJECT_ALIAS_NAME"),
		sonarURL:     os.Getenv("SONAR_URL"),
	}
}

func fetchState(serverInfor ServerInfor) *resty.Response {
	resq, err := resty.
		R().
		SetQueryParams(map[string]string{
			"metricKeys":   "bugs, duplicated_lines_density, code_smells, alert_status, sqale_index",
			"componentKey": serverInfor.projectName,
		}).
		SetBasicAuth(serverInfor.username, serverInfor.password).
		Get(serverInfor.sonarURL + "api/measures/component")
	if err != nil {
		return nil
	}
	return resq
}

func convertToNotif(status SonarStatus) (content NotifContent) {
	notifContent := NotifContent{
		id:   status.Component.ID,
		key:  status.Component.Key,
		name: status.Component.Name,
	}
	for _, val := range status.Component.Measures {
		switch val.Metric {
		case "bugs":
			notifContent.bugs = val.Value
		case "alert_status":
			notifContent.alertStatus = val.Value
			switch notifContent.alertStatus {
			case "ERROR":
				notifContent.color = "danger"
				notifContent.status = "DANGER"
			case "WARN":
				notifContent.color = "warning"
				notifContent.status = "WARNING"
			case "OK":
				notifContent.color = "good"
				notifContent.status = "GREAT!"
			}
		case "sqale_index":
			debtHours, err := strconv.ParseFloat(val.Value, 64)
			if err != nil {
				notifContent.sqaleIndex = "0"
			} else {
				debtDay := math.Ceil(debtHours / (60.0 * 8.0))
				notifContent.sqaleIndex = strconv.FormatFloat(debtDay, 'f', -1, 64)
			}
		case "code_smells":
			notifContent.codeSmells = val.Value
		case "duplicated_lines_density":
			notifContent.duplicatedLinesDensity = val.Value
		}
	}
	return notifContent
}

func sendToSlack(serverInfor ServerInfor, notifContent NotifContent) {
	attachment := slack.Attachment{}
	payload := slack.Payload{
		Channel:     serverInfor.slackChanel,
		Attachments: []slack.Attachment{attachment},
	}
	slack.Send(serverInfor.slackHookURL, "", payload)
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
	sendToSlack(serverInfor, notifContent)
}
