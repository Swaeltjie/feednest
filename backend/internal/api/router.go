package api

import (
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/feednest/backend/internal/api/handlers"
	"github.com/feednest/backend/internal/scheduler"
	"github.com/feednest/backend/internal/store"
)

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

	// Public routes
	r.Post("/api/auth/register", auth.Register)
	r.Post("/api/auth/login", auth.Login)
	r.Post("/api/auth/refresh", auth.Refresh)
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

		opmlH := handlers.NewOPMLHandler(queries)
		r.Post("/api/opml/import", opmlH.Import)
		r.Get("/api/opml/export", opmlH.Export)
	})

	return r
}
