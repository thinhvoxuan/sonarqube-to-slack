package main

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"os"

	"github.com/caarlos0/env"
	"github.com/go-resty/resty"
)

// ServerInfor read from env
type ServerInfor struct {
	Username     string `env:"SONAR_USERNAME"`
	Password     string `env:"SONAR_PASSWORD"`
	SonarURL     string `env:"SONAR_URL"`
	SlackHookURL string `env:"SLACK_HOOK_URL"`
	ProjectName  string `env:"PROJECT_ALIAS_NAME"`
	SlackChanel  string `env:"SLACK_CHANNEL" envDefault:"#general"`
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
	Coverage 	      			 string
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

func fetchState(serverInfor ServerInfor) *resty.Response {
	resq, err := resty.
		R().
		SetQueryParams(map[string]string{
			"metricKeys":   "bugs, duplicated_lines_density, code_smells, alert_status, sqale_index, coverage",
			"componentKey": serverInfor.ProjectName,
		}).
		SetBasicAuth(serverInfor.Username, serverInfor.Password).
		Get(serverInfor.SonarURL + "/api/measures/component")
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
		case "coverage":
			notifContent.Coverage = val.Value

		}
	}
	return notifContent
}

func (notifContent NotifContent) text() string {
	return notifContent.Status + " \n *" +
		notifContent.Bugs + " bugs*\n Technical debt: *" + notifContent.SqaleIndex +
		" days*\n Duplicated: *" + notifContent.DuplicatedLinesDensity +
		"%* \n *" + notifContent.CodeSmells + "* Code Smells" + 
		"%* \n *" + notifContent.Coverage + "* Code coverage"

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
	serverInfor := ServerInfor{}
	err := env.Parse(&serverInfor)
	if err != nil {
		fmt.Println("could not read env")
		os.Exit(1)
	}
	result := fetchState(serverInfor)
	if result == nil {
		fmt.Println("could not fetch information ")
		os.Exit(1)
	}
	var status SonarStatus
	error := json.Unmarshal(result.Body(), &status)
	if error != nil {
		fmt.Println("could not parse data: ", error)
		os.Exit(1)
	}
	notifContent := convertToNotif(status)
	manualSendSlack(serverInfor, notifContent)
}
