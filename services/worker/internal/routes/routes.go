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

	// Auth Routes (public)
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", handlers.Auth.Register)
		r.Post("/login", handlers.Auth.Login)
		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware)
			r.Put("/{userId}", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true,"data":"Update User"}`))
		})
			r.Post("/refresh", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true,"data":"Refresh Token"}`))
		})
			r.Delete("/{userId}", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true,"data":"Delete user"}`))
		})
		})
	})

	// Session Routes (protected)
	r.Route("/session", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware)
		r.Post("/", handlers.Session.Create)
		r.Get("/list/{userId}", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true,"data":"List Sessions"}`))
		})
		r.Get("/{sessionId}", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true,"data":"Get session info"}`))
		})
	})

	// Chat Routes (protected)
	r.Route("/chat", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware)
		r.Post("/", handlers.Chat.Send)
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true,"data":"Get Chats"}`))
		})
	})

	// Ingestion Route (protected)
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware)
		r.Post("/ingestion", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true,"data":"Ingestion push"}`))
		})
	})

	// Query Route (protected)
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware)
		r.Post("/query", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true,"data":"Query submission"}`))
		})
	})

	// Health Route (public)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true,"status":"healthy"}`))
	})

	return r
}
