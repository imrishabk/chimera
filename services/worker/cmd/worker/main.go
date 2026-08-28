package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"charm.land/log/v2"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"

	"github.com/imrishabk/chimera/services/worker/internal/database"
	grpcclient "github.com/imrishabk/chimera/services/worker/internal/grpc/client"
	"github.com/imrishabk/chimera/services/worker/internal/handler"
	"github.com/imrishabk/chimera/services/worker/internal/middleware"
	"github.com/imrishabk/chimera/services/worker/internal/repo"
	"github.com/imrishabk/chimera/services/worker/internal/routes"
	"github.com/imrishabk/chimera/services/worker/internal/service"
)

func main() {
	_ = godotenv.Load()

	dbHost := os.Getenv("DB_HOSTNAME")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USERNAME")
	dbPass := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_DATABASE")

	if dbHost == "" {
		dbHost = "127.0.0.1"
	}
	if dbPort == "" {
		dbPort = "5432"
	}

	log.Info("Creating database pool on", "host", dbHost, "port", dbPort, "database", dbName)
	connString := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s",
		dbUser, dbPass, dbHost, dbPort, dbName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.NewPostgresConnection(ctx, connString)
	if err != nil {
		log.Fatal("Failed to connect to database", "error", err)
	}
	defer pool.Close()
	log.Info("Successfully created database pool")
	repositories := repo.New(pool)

	grpcHost := os.Getenv("GRPC_AI_HOST")
	log.Info("Creating GRPC client on", "host", grpcHost)
	grpcClient, grpcErr := grpcclient.NewClient(grpcHost)
	if grpcErr != nil {
		grpcClient = nil
		log.Info("AI Core gRPC not available", "error", grpcErr)
	} else {
		defer grpcClient.Close()
	}
	log.Info("Successfully created GRPC client")
	services := service.NewServices(repositories)
	if grpcClient != nil {
		services.RAG = service.NewRAGService(grpcClient)
	}

	handlers := handler.NewHandlers(services)

	r := chi.NewRouter()
	r.Use(middleware.ErrorHandler)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			next.ServeHTTP(w, r)
		})
	})

	r.HandleFunc("/", defaultRoute)
	r.Mount("/api", routes.Configure(services, handlers))

	log.Info("Registering Routes")
	err = chi.Walk(r, func(method, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		log.Info("\t",
			"method", method,
			"route", route,
			"middlewares", len(middlewares),
		)
		return nil
	})
	if err != nil {
		log.Error("failed to walk through logs", "error", err)
	} else {
		log.Info("Successfully registered all routes")
	}

	log.Info("Starting server", "port", 8000, "db_connected", true)
	if err := http.ListenAndServe(":8000", r); err != nil {
		log.Fatal("failed to start the server!")
	}
}

func defaultRoute(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<body style="background:#1e1e1e;color:#d4d4d4"><pre>
		99 104 105 109 101 114 97

		██████╗██╗  ██╗██╗███╗   ███╗███████╗██████╗  █████╗
		██╔════╝██║  ██║██║████╗ ████║██╔════╝██╔══██╗██╔══██╗
		██║     ███████║██║██╔████╔██║█████╗  ██████╔╝███████║
		██║     ██╔══██║██║██║╚██╔╝██║██╔══╝  ██╔══██╗██╔══██║
		╚██████╗██║  ██║██║██║ ╚═╝ ██║███████╗██║  ██║██║  ██║
		╚═════╝╚═╝  ╚═╝╚═╝╚═╝     ╚═╝╚══════╝╚═╝  ╚═╝╚═╝  ╚═╝
		
		Welcome to chimera API.
		do visit 
		<a href="https://github.com/imrishabk/chimera">github</a>
		for more info.
		</pre></body>
		`))
}
