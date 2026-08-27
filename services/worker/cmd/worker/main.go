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

// Note: repositories variable declared but routes.Configure() does not yet
// receive repositories. Next step: wire repos into routes/service layer.

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

	connString := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s",
		dbUser, dbPass, dbHost, dbPort, dbName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.NewPostgresConnection(ctx, connString)
	if err != nil {
		log.Fatal("Failed to connect to database", "error", err)
	}
	defer pool.Close()

	repositories := repo.New(pool)

	services := service.NewServices(repositories)

	grpcClient, grpcErr := grpcclient.NewClient(os.Getenv("GRPC_AI_HOST"))
	if grpcErr != nil {
		grpcClient = nil
		log.Info("AI Core gRPC not available", "error", grpcErr)
	} else {
		defer grpcClient.Close()
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
