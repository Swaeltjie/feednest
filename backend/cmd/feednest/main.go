package main

import (
	"log"
	"net/http"
	"os"

	"github.com/feednest/backend/internal/api"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router := api.NewRouter()

	log.Printf("FeedNest backend starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
