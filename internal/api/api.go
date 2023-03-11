package api

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

func Create(stop chan os.Signal, log *logrus.Logger) *http.Server {
	router := chi.NewRouter()
	// Health endpoint
	router.Get("/health", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		if _, writeErr := writer.Write([]byte("OK")); writeErr != nil {
			log.Error("Error writing OK message:", writeErr)
		}
	})

	// hello world
	router.Get("/", func(writer http.ResponseWriter, request *http.Request) {
		filePath, err := filepath.Abs("./internal/api/pages/helloworld.html")
		if err != nil {
			log.Fatal(err)
		}
		http.ServeFile(writer, request, filePath)
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
