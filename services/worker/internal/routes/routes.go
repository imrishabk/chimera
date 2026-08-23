package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func Configure() chi.Router {
	r := chi.NewRouter()

	// Auth Routes
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Register Route"))
		})
		r.Post("/login", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Login Route"))
		})
		r.Group(func(r chi.Router) {
			r.Put("/{userId}", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("Update User"))
			})
			r.Post("/refresh", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("Refresh Token"))
			})
			r.Delete("/{userId}", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("Delete user"))
			})
		})
	})

	// Session Routes
	r.Route("/session", func(r chi.Router) {
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Create Session"))
		})
		r.Get("/list/{userId}", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("List Sessions"))
		})
		r.Get("/{sessionId}", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Get session info"))
		})
	})

	// Chat Routes
	r.Route("/chat", func(r chi.Router) {
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Send a chat"))
		})
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Get Chats"))
		})
	})

	// Ingestion Route
	r.Group(func(r chi.Router) {
		r.Post("/ingestion", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Ingestion push"))
		})
	})

	// Query Route
	r.Group(func(r chi.Router) {
		r.Post("/query", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Query submission"))
		})
	})

	// Health Route
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Healthy"))
	})

	return r
}
