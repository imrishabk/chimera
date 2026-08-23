package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func UserRoutes() chi.Router {
	r := chi.NewRouter()

	r.Post("/register", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Register Route"))
	})

	r.Post("/login", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Login Route"))
	})

	return r
}
