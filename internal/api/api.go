package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/lu1a/live-explan/config"
	"github.com/lu1a/live-explan/internal/util"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

func checkAuth(request *http.Request) bool {
	token := request.Header.Get("Authorization")
	if token == "" {
		return false
	}
	AUTH_TOKENS := strings.Split(config.MainConfig.GetString("AUTH_TOKENS"), ",")
	token = strings.TrimPrefix(token, "Bearer ")

	found := false
	for _, whitelistToken := range AUTH_TOKENS {
		if token == whitelistToken {
			found = true
			break
		}
	}

	return found
}

type VisitorLog struct {
	ID                 *int      `json:"id"`
    ForUser            int       `db:"for_user" json:"for_user"`
    VisitedAt          time.Time `db:"visited_at" json:"visited_at"`
    URLPath            string    `db:"url_path" json:"url_path"`
    IPAddress          string    `db:"ip_address" json:"ip_address"`
    IPISP              string    `db:"ip_isp" json:"ip_isp"`
    IPCountry          *string   `db:"ip_country" json:"ip_country"`
    IPCity             *string   `db:"ip_city" json:"ip_city"`
    IPZip              *string   `db:"ip_zip" json:"ip_zip"`
    IPLatitude         *string   `db:"ip_latitude" json:"ip_latitude"`
    IPLongitude        *string   `db:"ip_longitude" json:"ip_longitude"`
    Browser            string    `db:"browser" json:"browser"`
    OperatingSystem    string    `db:"operating_system" json:"operating_system"`
    IsMobile           bool      `db:"is_mobile" json:"is_mobile"`
    RefererURL         string    `db:"referer_url" json:"referer_url"`
    PreferredLanguages string    `db:"preferred_languages" json:"preferred_languages"`
    Cookies            string    `db:"cookies" json:"cookies"`
    Body               string    `db:"body" json:"body"`
}

func insertNewVisitorLog(log *logrus.Logger, db *sqlx.DB, logEntry *VisitorLog) {
    query := `
        INSERT INTO visitor_log (
            for_user, visited_at, url_path, ip_address,
            ip_isp, ip_country, ip_city, ip_zip, ip_latitude,
			ip_longitude, browser, operating_system, is_mobile,
            referer_url, preferred_languages, cookies, body
        )
        VALUES (
            :for_user, :visited_at, :url_path, :ip_address,
            :ip_isp, :ip_country, :ip_city, :ip_zip, :ip_latitude,
			:ip_longitude, :browser, :operating_system, :is_mobile,
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

	// Health endpoint
	router.Get("/health", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		if _, writeErr := writer.Write([]byte("OK")); writeErr != nil {
			log.Error("Error writing OK message:", writeErr)
		}
	})

	router.Get("/visitor-log-entries", func(writer http.ResponseWriter, request *http.Request) {
		isAuthed := checkAuth(request)
		if !isAuthed {
			http.Error(writer, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var visitorLogs []VisitorLog
		err := db.Select(&visitorLogs, "SELECT * FROM visitor_log")
		if err != nil {
			http.Error(writer, "Internal Server Error", http.StatusInternalServerError)
			log.Fatal(err)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(writer).Encode(visitorLogs)
		if err != nil {
			log.Fatal(err)
		}
	})

	router.Get("/unique-ips-by-country", func(writer http.ResponseWriter, request *http.Request) {
		isAuthed := checkAuth(request)
		if !isAuthed {
			http.Error(writer, "Unauthorized", http.StatusUnauthorized)
			return
		}

		type CountryCount struct {
			Country  string `db:"ip_country" json:"country"`
			IPCount  int    `db:"ip_count" json:"count"`
		}

		var countryCounts []CountryCount
		err := db.Select(&countryCounts, 
			`SELECT
				ip_country,
				COUNT(DISTINCT ip_address) AS ip_count
			FROM
				visitor_log
			GROUP BY
				ip_country
			ORDER BY
				ip_count DESC`,
		)
		if err != nil {
			http.Error(writer, "Internal Server Error", http.StatusInternalServerError)
			log.Fatal(err)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(writer).Encode(countryCounts)
		if err != nil {
			log.Fatal(err)
		}
	})

	router.Post("/visitor-log-entry", func(writer http.ResponseWriter, request *http.Request) {
		isAuthed := checkAuth(request)
		if !isAuthed {
			http.Error(writer, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var logEntry VisitorLog
    	err := json.NewDecoder(request.Body).Decode(&logEntry)
		if err != nil {
			log.Fatal(err)
		}

		insertNewVisitorLog(log, db, &logEntry)
	})

	router.Post("/contact", func(writer http.ResponseWriter, request *http.Request) {
		isAuthed := checkAuth(request)
		if !isAuthed {
			http.Error(writer, "Unauthorized", http.StatusUnauthorized)
			return
		}

		type ContactJSON struct {
			SenderAddress string `json:"sender_address" validate:"required"`
			Subject       string `json:"subject" validate:"required"`
			Content       string `json:"content" validate:"required"`
		}
		
		var contactData ContactJSON
		err := json.NewDecoder(request.Body).Decode(&contactData)
		if err != nil {
			log.Error(err)
			http.Error(writer, "Invalid JSON", http.StatusBadRequest)
			return
		}
	
		senderAddress := contactData.SenderAddress
		subject := contactData.Subject
		content := contactData.Content

		if senderAddress == "" || content == "" {
			http.Error(writer, "{\"error\":\"Required field(s) missing.\"}", http.StatusBadRequest)
			return
		}
		if !util.IsValidEmail(senderAddress) {
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

		err = util.SendTelegramMessage(log, senderAddress, subject, content)
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
