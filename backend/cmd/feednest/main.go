package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/feednest/backend/internal/api"
	"github.com/feednest/backend/internal/scheduler"
	"github.com/feednest/backend/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./feednest.db"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" || jwtSecret == "change-me-in-production" {
		log.Fatal("FATAL: JWT_SECRET must be set to a strong random value (not the default)")
	}

	db, err := store.NewDB(dbPath)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer db.Close()

	queries := store.New(db)

	sched := scheduler.New(queries, 5*time.Minute)
	sched.Start()

	router := api.NewRouter(queries, jwtSecret, sched)
	defer sched.Stop()

	log.Printf("FeedNest backend starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
