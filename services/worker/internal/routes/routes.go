package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/imrishabk/chimera/services/worker/internal/handler"
	"github.com/imrishabk/chimera/services/worker/internal/middleware"
	"github.com/imrishabk/chimera/services/worker/internal/service"
)

func Configure(svc *service.Services, handlers *handler.Handlers) chi.Router {
	r := chi.NewRouter()

	// Choose validated middleware when Auth service is available (checks DB expiry),
	// otherwise fallback to simple presence check for tests.
	authMiddleware := middleware.AuthMiddleware
	if svc != nil && svc.Auth != nil {
		authMiddleware = middleware.AuthMiddlewareValidated(svc.Auth)
	}

	// Auth Routes (public)
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", handlers.Auth.Register)
		r.Post("/login", handlers.Auth.Login)
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Put("/{userId}", handlers.Auth.UpdateUser)
			r.Post("/refresh", handlers.Auth.Refresh)
			r.Post("/logout", handlers.Auth.Logout)
			r.Post("/logout/all", handlers.Auth.LogoutAll)
			r.Delete("/{userId}", handlers.Auth.DeleteUser)
		})
	})

	// Session Routes (protected)
	r.Route("/session", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/", handlers.Session.Create)
		r.Get("/list/{userId}", handlers.Session.List)
		r.Get("/{sessionId}", handlers.Session.Get)
	})

	// Chat Routes (protected)
	r.Route("/chat", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/", handlers.Chat.Send)
		r.Post("/stream", handlers.Chat.Stream)
		r.Get("/", handlers.Chat.List)
	})

	// Ingestion Routes (protected) — job tracking via PLAN.md
	r.Route("/ingestion", func(r chi.Router) {
		r.Use(authMiddleware)
		if handlers.Ingest != nil {
			r.Post("/", handlers.Ingest.Push)
			r.Get("/{jobId}", handlers.Ingest.Get)
			r.Get("/list/{sessionId}", handlers.Ingest.List)
		} else {
			r.Post("/", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(`{"success":false,"error":"ingestion service unavailable — AI Core not connected"}`))
			})
		}
	})

	// Query Route (protected)
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		if handlers.Query != nil {
			r.Post("/query", handlers.Query.Submit)
		} else {
			r.Post("/query", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(`{"success":false,"error":"query service unavailable — AI Core not connected"}`))
			})
		}
	})

	// Health Route (public)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true,"status":"healthy"}`))
	})

	return r
}
