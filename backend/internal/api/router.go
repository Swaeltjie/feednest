package api

import (
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/feednest/backend/internal/api/handlers"
	"github.com/feednest/backend/internal/scheduler"
	"github.com/feednest/backend/internal/store"
)

// rateLimiter implements a simple per-IP rate limiter for auth endpoints.
type rateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	window   time.Duration
	max      int
	stop     chan struct{}
}

func newRateLimiter(window time.Duration, max int) *rateLimiter {
	rl := &rateLimiter{
		attempts: make(map[string][]time.Time),
		window:   window,
		max:      max,
		stop:     make(chan struct{}),
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, times := range rl.attempts {
				var valid []time.Time
				for _, t := range times {
					if now.Sub(t) < rl.window {
						valid = append(valid, t)
					}
				}
				if len(valid) == 0 {
					delete(rl.attempts, ip)
				} else {
					rl.attempts[ip] = valid
				}
			}
			rl.mu.Unlock()
		case <-rl.stop:
			return
		}
	}
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	var valid []time.Time
	for _, t := range rl.attempts[ip] {
		if now.Sub(t) < rl.window {
			valid = append(valid, t)
		}
	}
	if len(valid) >= rl.max {
		rl.attempts[ip] = valid
		return false
	}
	rl.attempts[ip] = append(valid, now)
	return true
}

func rateLimitMiddleware(rl *rateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
				ip = strings.SplitN(fwd, ",", 2)[0]
				ip = strings.TrimSpace(ip)
			}
			if !rl.allow(ip) {
				http.Error(w, `{"error":"too many requests, please try again later"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func NewRouter(queries *store.Queries, jwtSecret string, sched *scheduler.Scheduler) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	// Security headers
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
			w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")
			next.ServeHTTP(w, r)
		})
	})

	// Global request body size limit (1MB)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, 1*1024*1024)
			next.ServeHTTP(w, r)
		})
	})

	allowedOrigins := []string{"http://localhost:5173", "http://localhost:3000"}
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		allowedOrigins = strings.Split(origins, ",")
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Swagger UI
	r.Get("/api/docs", swaggerUI)
	r.Get("/api/docs/openapi.yaml", openapiYAML)

	auth := NewAuthHandler(queries, jwtSecret)

	// Rate limiter for auth endpoints: 10 attempts per minute per IP
	authRL := newRateLimiter(1*time.Minute, 10)

	// Public routes
	r.Group(func(r chi.Router) {
		r.Use(rateLimitMiddleware(authRL))
		r.Post("/api/auth/register", auth.Register)
		r.Post("/api/auth/login", auth.Login)
		r.Post("/api/auth/refresh", auth.Refresh)
	})
	r.Get("/api/auth/user-count", auth.UserCount)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(AuthMiddleware(jwtSecret))

		categoriesH := handlers.NewCategoryHandler(queries)
		r.Get("/api/categories", categoriesH.List)
		r.Post("/api/categories", categoriesH.Create)
		r.Put("/api/categories/{id}", categoriesH.Update)
		r.Delete("/api/categories/{id}", categoriesH.Delete)

		feedsH := handlers.NewFeedHandler(queries, sched)
		r.Get("/api/feeds", feedsH.List)
		r.Post("/api/feeds", feedsH.Create)
		r.Put("/api/feeds/{id}", feedsH.Update)
		r.Delete("/api/feeds/{id}", feedsH.Delete)

		r.Post("/api/feeds/{id}/retry", feedsH.Retry)

		discoverH := handlers.NewDiscoverHandler()
		r.Post("/api/feeds/discover", discoverH.Discover)

		articlesH := handlers.NewArticleHandler(queries)
		r.Get("/api/articles", articlesH.List)
		r.Post("/api/articles/mark-all-read", articlesH.MarkAllRead)
		r.Post("/api/articles/bulk", articlesH.Bulk)
		r.Post("/api/articles/catch-up", articlesH.CatchUp)
		r.Get("/api/articles/{id}", articlesH.Get)
		r.Put("/api/articles/{id}", articlesH.Update)
		r.Post("/api/articles/{id}/dismiss", articlesH.Dismiss)

		tagsH := handlers.NewTagHandler(queries)
		r.Get("/api/tags", tagsH.List)
		r.Post("/api/articles/{id}/tags", tagsH.AddToArticle)
		r.Delete("/api/articles/{id}/tags/{tag}", tagsH.RemoveFromArticle)

		eventsH := handlers.NewEventHandler(queries)
		r.Post("/api/events", eventsH.Create)

		settingsH := handlers.NewSettingsHandler(queries)
		r.Get("/api/settings", settingsH.Get)
		r.Put("/api/settings", settingsH.Update)

		rulesH := handlers.NewRulesHandler(queries)
		r.Get("/api/rules", rulesH.List)
		r.Post("/api/rules", rulesH.Create)
		r.Put("/api/rules/{id}", rulesH.Update)
		r.Delete("/api/rules/{id}", rulesH.Delete)

		opmlH := handlers.NewOPMLHandler(queries)
		r.Post("/api/opml/import", opmlH.Import)
		r.Get("/api/opml/export", opmlH.Export)
	})

	return r
}
