package util

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
)

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
