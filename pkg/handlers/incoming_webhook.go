package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

type Annotations []Annotation

type Annotation struct {
	ID           int64    `json:"id"`
	AlertID      int64    `json:"alertId"`
	AlertName    string   `json:"alertName"`
	DashboardID  int64    `json:"dashboardId"`
	DashboardUID string   `json:"dashboardUID"`
	PanelID      int64    `json:"panelId"`
	UserID       int64    `json:"userId"`
	NewState     string   `json:"newState"`
	PrevState    string   `json:"prevState"`
	Created      int64    `json:"created"`
	Updated      int64    `json:"updated"`
	Time         int64    `json:"time"`
	TimeEnd      int64    `json:"timeEnd"`
	Text         string   `json:"text"`
	Tags         []string `json:"tags"`
	Login        string   `json:"login"`
	Email        string   `json:"email"`
	AvatarURL    string   `json:"avatarUrl"`
	Data         Data     `json:"data"`
}

type Data struct {
	Values *Values `json:"values,omitempty"`
}

type Values struct {
	A *float64 `json:"A,omitempty"`
	B *float64 `json:"B,omitempty"`
	C *int64   `json:"C,omitempty"`
}

var reAlertName = regexp.MustCompile(`alertname=([^,]*)`)

func ProcessIncomingMessage() echo.HandlerFunc {
	return func(c echo.Context) error {

		tsNow := time.Now().UnixMilli()
		tsSince := tsNow - (1000 * 60 * 10) // last 10 minutes
		grafanaUrl := os.Getenv("GRAFANA_URL")
		url := fmt.Sprintf("%s/api/annotations/?from=%d&to=%d", grafanaUrl, tsSince, tsNow)

		client := &http.Client{}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}

		req.Header.Add("Authorization", os.Getenv("GRAFANA_BASIC_AUTH"))
		time.Sleep(time.Second * 5) // Wait for grafana to populate Annotations
		res, err := client.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}

		var ans Annotations
		err = json.Unmarshal(body, &ans)
		if err != nil {
			return err
		}

		var sb strings.Builder
		for _, an := range ans {
			if an.Data.Values.A != nil && strings.Contains(an.NewState, "Alerting") {
				matched := reAlertName.FindString(an.Text)
				if matched != "" {
					matchedval := strings.Replace(matched, "alertname=", "", -1)
					sb.WriteString(matchedval + `\n`)
					sb.WriteString("actual value is: " + fmt.Sprintf("%v", *an.Data.Values.A) + `\n`)
					// sb.WriteString(fmt.Sprintf("actual value is: %v\n", *an.Data.Values.A))
				}
			}
		}
		sbResult := sb.String()
		// log.Println("result: ", sbResult)
		if len(sbResult) == 0 {
			return nil
		}
		// if err = PostTelegramMessage(sbResult); err != nil {
		// 	return err
		// }

		if err = PostDiscordMessage(sbResult); err != nil {
			return err
		}

		return c.String(200, sbResult)
	}
}

func PostTelegramMessage(msg string) error {
	API_KEY := os.Getenv("TELEGRAM_KEY")
	data := `{"text":"` + msg + `","chat_id":"` + os.Getenv("TELEGRAM_CHAT_ID") + `"}`
	buf := bytes.NewBuffer([]byte(data))
	_, err := http.Post("https://api.telegram.org/bot"+API_KEY+"/sendMessage", "application/json", buf)
	return err
}

func PostDiscordMessage(msg string) error {
	WEBHOOK_URL := os.Getenv("DISCORD_HOME_AUTOMATION_HOOK")
	data := `{"content":"` + msg + `"}`
	buf := bytes.NewBuffer([]byte(data))
	_, err := http.Post(WEBHOOK_URL, "application/json", buf)
	return err
}
