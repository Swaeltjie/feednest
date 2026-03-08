package main

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
		// Auto-generate and persist a secure JWT secret
		jwtSecret = loadOrGenerateSecret(filepath.Dir(dbPath))
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

// loadOrGenerateSecret reads a JWT secret from a file in dataDir, or generates
// a cryptographically secure 256-bit key and persists it for future restarts.
func loadOrGenerateSecret(dataDir string) string {
	secretFile := filepath.Join(dataDir, ".jwt_secret")

	if data, err := os.ReadFile(secretFile); err == nil {
		secret := strings.TrimSpace(string(data))
		if len(secret) >= 32 {
			log.Println("JWT_SECRET loaded from", secretFile)
			return secret
		}
	}

	// Generate 32 bytes (256-bit) of cryptographic randomness
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		log.Fatalf("failed to generate JWT secret: %v", err)
	}
	secret := hex.EncodeToString(key)

	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		log.Fatalf("failed to create data directory: %v", err)
	}
	if err := os.WriteFile(secretFile, []byte(secret+"\n"), 0o600); err != nil {
		log.Fatalf("failed to persist JWT secret: %v", err)
	}

	log.Println("JWT_SECRET auto-generated and saved to", secretFile)
	return secret
}
