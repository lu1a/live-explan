package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/lu1a/live-explan/config"
	"github.com/sirupsen/logrus"
)

func SendTelegramMessage(log *logrus.Logger, sender_address, subject, content string) error {
	token := config.MainConfig.GetString("TELEGRAM_TOKEN")
	chatID := config.MainConfig.GetString("TELEGRAM_CHATID")
	msg := "Subject: " + subject + "\r\n" +
		content + "\r\n" +
		"From " + sender_address
	url := fmt.Sprintf("%s/sendMessage", fmt.Sprintf("https://api.telegram.org/bot%s", token))
	body, _ := json.Marshal(map[string]string{
		"chat_id": chatID,
		"text":    msg,
	})

	response, err := http.Post(
		url,
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return err
	}

	// Close the request at the end
	defer response.Body.Close()

	if response.StatusCode < 200 && response.StatusCode >= 300 {
		return fmt.Errorf("Bad response")
	}

	log.Info("A contact request was sent")
	return nil
}

func RespondwithJSON(w http.ResponseWriter, log *logrus.Logger, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		log.Error("Couldn't create JSON from payload", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err = w.Write(response)
	if err != nil {
		log.Error("Something went wrong when writing out response", "error", err)
	}
}

func RespondWithError(w http.ResponseWriter, log *logrus.Logger, code int, msg string) {
	RespondwithJSON(w, log, code, map[string]string{"message": msg})
}
