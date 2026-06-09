package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
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
	} else if len(jwtSecret) < 32 {
		log.Fatal("JWT_SECRET must be at least 32 characters; use a long, random value or leave it unset to auto-generate a secure key")
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

	// Explicit timeouts defend against slow-header (Slowloris) connections that
	// would otherwise hold request goroutines open forever. WriteTimeout is
	// deliberately omitted: /api/ask runs a 60s Claude budget and a shorter
	// server-wide WriteTimeout would truncate legitimate streamed answers.
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("FeedNest backend starting on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown: wait for SIGINT/SIGTERM, then drain in-flight requests
	// so the deferred db.Close()/sched.Stop() actually run (log.Fatal previously
	// os.Exit'd past them on every container stop).
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown error: %v", err)
	}
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
