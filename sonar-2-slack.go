package main

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/go-resty/resty"
)

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
	alertStatus            string
	sqaleIndex             string
	bugs                   string
	duplicatedLinesDensity string
	codeSmells             string
}

func fetchState(domain string, username string, password string) *resty.Response {
	resq, err := resty.
		R().
		SetQueryParams(map[string]string{
			"metricKeys":   "bugs, duplicated_lines_density, code_smells, alert_status, sqale_index",
			"componentKey": "gu_came_web_v2",
		}).
		SetBasicAuth(username, password).
		Get(domain + "api/measures/component")
	if err != nil {
		return nil
	}
	return resq
}

func convertToNotif(status SonarStatus) (content NotifContent) {
	notifContent := NotifContent{
		ID:    status.Component.ID,
		Key:   status.Component.Key,
		Name:  status.Component.Name,
		Color: "Green",
	}
	for _, val := range status.Component.Measures {
		switch val.Metric {
		case "bugs":
			notifContent.bugs = val.Value
		case "alert_status":
			notifContent.alertStatus = val.Value
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

func main() {
	domain, username, password, webHook := "http://sonarqube.geekup.vn/", "temp", "gu123451", "https://hooks.slack.com/services/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"
	result := fetchState(domain, username, password)
	if result == nil {
		return
	}
	var status SonarStatus
	err := json.Unmarshal(result.Body(), &status)
	if err != nil {
		return
	}
	notifContent := convertToNotif(status)
	fmt.Println(notifContent, webHook)
}
