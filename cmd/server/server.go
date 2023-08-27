package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"

	"github.com/lu1a/live-explan/config"
	"github.com/lu1a/live-explan/internal/api"
)

func startServerDeps() {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.DebugLevel)

	// set up signal handling
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	// init data-worker DB connection
	/**
		TODO: Remove this entirely.
		In the future, the data-worker will only be accessible via kafka and API.
	*/
	db, err := sqlx.Open("postgres", config.MainConfig.GetString("TSDB_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	pingErr := db.Ping()
    if pingErr != nil {
        log.Fatal("Error connecting to the database: ", pingErr)
    }

	// start the api
	server := api.Create(stop, db, log)

	// execute more code here...

	// wait for interrupt signal
	<-stop

	log.Info("Received interrupt signal. Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Errorf("Error shutting down server: %s", err)
	}
	log.Info("Server stopped.")
}
