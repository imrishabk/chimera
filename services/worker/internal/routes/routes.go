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

	authMiddleware := middleware.AuthMiddlewareValidated(svc.Auth)

	r.Route("/auth", func(r chi.Router) {
		r.Method(http.MethodPost, "/register", handler.AppHandler(handlers.Auth.Register))
		r.Method(http.MethodPost, "/login", handler.AppHandler(handlers.Auth.Login))
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Method(http.MethodPut, "/{userId}", handler.AppHandler(handlers.Auth.UpdateUser))
			r.Method(http.MethodPost, "/refresh", handler.AppHandler(handlers.Auth.Refresh))
			r.Method(http.MethodPost, "/logout", handler.AppHandler(handlers.Auth.Logout))
			r.Method(http.MethodPost, "/logout/all", handler.AppHandler(handlers.Auth.LogoutAll))
		})
	})

	r.Route("/session", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Method(http.MethodPost, "/", handler.AppHandler(handlers.Session.Create))
		r.Method(http.MethodGet, "/list/{userId}", handler.AppHandler(handlers.Session.List))
		r.Method(http.MethodGet, "/{sessionId}", handler.AppHandler(handlers.Session.Get))
	})

	r.Route("/chat", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Method(http.MethodPost, "/", handler.AppHandler(handlers.Chat.Send))
		r.Method(http.MethodPost, "/stream", handler.AppHandler(handlers.Chat.Stream))
		r.Method(http.MethodGet, "/", handler.AppHandler(handlers.Chat.List))
	})

	r.Route("/ingestion", func(r chi.Router) {
		r.Use(authMiddleware)
		if handlers.Ingest != nil {
			r.Method(http.MethodPost, "/", handler.AppHandler(handlers.Ingest.Push))
			r.Method(http.MethodPost, "/upload", handler.AppHandler(handlers.Ingest.Upload))
			r.Method(http.MethodPost, "/fetch", handler.AppHandler(handlers.Ingest.FetchURLs))
			r.Method(http.MethodGet, "/{jobId}", handler.AppHandler(handlers.Ingest.Get))
			r.Method(http.MethodGet, "/list/{sessionId}", handler.AppHandler(handlers.Ingest.List))
		} else {
			r.Method(http.MethodPost, "/", handler.AppHandler(func(w http.ResponseWriter, r *http.Request) error {
				w.WriteHeader(http.StatusServiceUnavailable)
				//nolint:errcheck
				w.Write([]byte(`{"success":false,"error":"ingestion service unavailable — AI Core not connected"}`))
				return nil
			}))
		}
	})

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		if handlers.Query != nil {
			r.Method(http.MethodPost, "/query", handler.AppHandler(handlers.Query.Submit))
		} else {
			r.Method(http.MethodPost, "/query", handler.AppHandler(func(w http.ResponseWriter, r *http.Request) error {
				w.WriteHeader(http.StatusServiceUnavailable)
				//nolint:errcheck
				w.Write([]byte(`{"success":false,"error":"query service unavailable — AI Core not connected"}`))
				return nil
			}))
		}
	})

	// Health Route (public)
	r.Method(http.MethodGet, "/health", handler.AppHandler(func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusOK)
		//nolint:errcheck
		w.Write([]byte(`{"success":true,"status":"healthy"}`))
		return nil
	}))

	return r
}
