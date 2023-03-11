package util

import (
	"encoding/json"
	"net/http"
	"net/smtp"

	"github.com/sirupsen/logrus"

	"github.com/lu1a/live-explan/config"
)

func EmailHandler(w http.ResponseWriter, log *logrus.Logger, sender_address, subject, content string) {
	lewis_email_address := config.MainConfig.GetString("LEWIS_EMAIL_ADDRESS")
	lewis_email_password := config.MainConfig.GetString("LEWIS_EMAIL_PASSWORD")

	// Set up authentication for sending email
	auth := smtp.PlainAuth("", lewis_email_address, lewis_email_password, "smtp.gmail.com")

	// Compose the email message
	msg := "From: " + lewis_email_address + "\r\n" +
		"To: recipient@gmail.com\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		content + "\r\n" +
		"From " + sender_address

	// Send the email
	err := smtp.SendMail("smtp.gmail.com:587", auth, lewis_email_address, []string{lewis_email_address}, []byte(msg))
	if err != nil {
		RespondWithError(w, log, 500, err.Error())
		return
	}

	// Respond with a success message
	RespondwithJSON(w, log, 200, "Email sent successfully")
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
