package api

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lu1a/live-explan/internal/util"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

var limiter = rate.NewLimiter(rate.Every(time.Hour/10), 1)

func Create(stop chan os.Signal, log *logrus.Logger) *http.Server {
	router := chi.NewRouter()
	// Health endpoint
	router.Get("/health", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		if _, writeErr := writer.Write([]byte("OK")); writeErr != nil {
			log.Error("Error writing OK message:", writeErr)
		}
	})

	router.Get("/", func(writer http.ResponseWriter, request *http.Request) {
		filePath, err := filepath.Abs("./internal/api/pages/faux-terminal.html")
		if err != nil {
			log.Fatal(err)
		}
		http.ServeFile(writer, request, filePath)
	})

	router.Post("/contact", func(writer http.ResponseWriter, request *http.Request) {
		sender_address := request.FormValue("sender_address")
		subject := request.FormValue("subject")
		content := request.FormValue("content")

		if sender_address == "" || content == "" {
			http.Error(writer, "{\"error\":\"Required field(s) missing.\"}", http.StatusBadRequest)
			return
		}
		if !util.IsValidEmail(sender_address) {
			http.Error(writer, "{\"error\":\"Email address is not valid.\"}", http.StatusBadRequest)
			return
		}
		if subject == "" {
			subject = "Contact request"
		}

		if !limiter.Allow() {
			http.Error(writer, "{\"error\":\"Too many requests; please wait ~10 more minutes.\"}", http.StatusTooManyRequests)
			return
		}

		err := util.SendTelegramMessage(log, sender_address, subject, content)
		if err != nil {
			http.Error(writer, "{\"error\":\"Something went wrong with my contacting service!\"}", http.StatusInternalServerError)
		}
	})

	// create an HTTP server
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// start the server in a separate goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("Error starting server: %s", err)
			os.Exit(1)
		}
	}()
	log.Info("✅ API is up.")

	return server
}
