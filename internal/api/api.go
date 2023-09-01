package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/lu1a/live-explan/internal/util"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

type VisitorLog struct {
    ForUser            int       `db:"for_user"`
    VisitedAt          time.Time `db:"visited_at"`
    URLPath            string    `db:"url_path"`
    IPAddress          string    `db:"ip_address"`
    Geolocation        string    `db:"geolocation"`
    IPISP              string    `db:"ip_isp"`
    Browser            string    `db:"browser"`
    OperatingSystem    string    `db:"operating_system"`
    IsMobile           bool      `db:"is_mobile"`
    RefererURL         string    `db:"referer_url"`
    PreferredLanguages string    `db:"preferred_languages"`
    Cookies            string    `db:"cookies"`
    Body               string    `db:"body"`
}

func insertVisitorLog(log *logrus.Logger, db *sqlx.DB, request *http.Request) {
    var cookiesStrBuilder strings.Builder
	for _, cookie := range request.Cookies() {
		cookiesStrBuilder.WriteString(fmt.Sprintf("%s: %s\n", cookie.Name, cookie.Value))
	}
	cookiesString := cookiesStrBuilder.String()

	bodyContent := ""
	contentType := strings.TrimSpace(strings.Split(request.Header.Get("Content-Type"), ";")[0])

	switch contentType {
	case "application/json":
		// Handle JSON content type
		bodyBytes, _ := io.ReadAll(request.Body)
		bodyContent = string(bodyBytes)
	case "multipart/form-data":
		err := request.ParseMultipartForm(32 << 20) // Max memory is 32 MB
		if err != nil {
			log.Fatal("Unable to parse form", err)
			return
		}
		var formDataStrBuilder strings.Builder
		for key, values := range request.MultipartForm.Value {
			formDataStrBuilder.WriteString(fmt.Sprintf("%s: %s\n", key, strings.Join(values, ", ")))
		}

		bodyContent = formDataStrBuilder.String()
	default:
		// TODO: Handle other content types
		// I might need to read and process the raw body here
	}

	realIP := request.Header.Get("X-Real-IP")
    if realIP == "" {
        realIP = request.RemoteAddr
    }

    logEntry := VisitorLog{
        ForUser:          1, // Assuming the user ID as me!
        VisitedAt:        time.Now().UTC(),
        URLPath:          request.URL.Path,
        IPAddress:        realIP,
		/** 
			I'll probably fill Geolocation/IPISP out later if I can ever be bothered 
			getting a geo-ip API licence
		*/
		Browser: request.Header.Get("User-Agent"),
		OperatingSystem: request.Header.Get("Sec-Ch-Ua-Platform"),
		IsMobile: request.Header.Get("Sec-Ch-Ua-Mobile") == "?1",
		RefererURL: request.Referer(),
		PreferredLanguages: request.Header.Get("Accept-Language"),
		Cookies: cookiesString,
		Body: bodyContent,
    }

    query := `
        INSERT INTO visitor_log (
            for_user, visited_at, url_path, ip_address, geolocation,
            ip_isp, browser, operating_system, is_mobile,
            referer_url, preferred_languages, cookies, body
        )
        VALUES (
            :for_user, :visited_at, :url_path, :ip_address, :geolocation,
            :ip_isp, :browser, :operating_system, :is_mobile,
            :referer_url, :preferred_languages, :cookies, :body
        )
    `

    _, err := db.NamedExec(query, logEntry)
	if err != nil {
		log.Fatal(err)
	}
}

func insertNewVisitorLog(log *logrus.Logger, db *sqlx.DB, logEntry *VisitorLog) {
    query := `
        INSERT INTO visitor_log (
            for_user, visited_at, url_path, ip_address, geolocation,
            ip_isp, browser, operating_system, is_mobile,
            referer_url, preferred_languages, cookies, body
        )
        VALUES (
            :for_user, :visited_at, :url_path, :ip_address, :geolocation,
            :ip_isp, :browser, :operating_system, :is_mobile,
            :referer_url, :preferred_languages, :cookies, :body
        )
    `

    _, err := db.NamedExec(query, logEntry)
	if err != nil {
		log.Fatal(err)
	}
}

var limiter = rate.NewLimiter(rate.Every(time.Hour/10), 1)

func Create(stop chan os.Signal, db *sqlx.DB, log *logrus.Logger) *http.Server {
	router := chi.NewRouter()

	// TODO: check authentication here

	// Health endpoint
	router.Get("/health", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		if _, writeErr := writer.Write([]byte("OK")); writeErr != nil {
			log.Error("Error writing OK message:", writeErr)
		}
	})

	router.Post("/visitor-log-entry", func(writer http.ResponseWriter, request *http.Request) {
		var logEntry VisitorLog
    	err := json.NewDecoder(request.Body).Decode(&logEntry)
		if err != nil {
			log.Fatal(err)
		}

		insertNewVisitorLog(log, db, &logEntry)

		filePath, err := filepath.Abs("./internal/api/pages/faux-terminal.html")
		if err != nil {
			log.Fatal(err)
		}
		http.ServeFile(writer, request, filePath)
	})

	router.Post("/contact", func(writer http.ResponseWriter, request *http.Request) {
		insertVisitorLog(log, db, request)

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
