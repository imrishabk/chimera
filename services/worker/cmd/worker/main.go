package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"charm.land/log/v2"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/imrishabk/chimera/services/worker/internal/database"
	grpcclient "github.com/imrishabk/chimera/services/worker/internal/grpc/client"
	"github.com/imrishabk/chimera/services/worker/internal/handler"
	"github.com/imrishabk/chimera/services/worker/internal/middleware"
	"github.com/imrishabk/chimera/services/worker/internal/repo"
	"github.com/imrishabk/chimera/services/worker/internal/routes"
	"github.com/imrishabk/chimera/services/worker/internal/service"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("FAILED to load .env file", "error", err)
	}
}

func main() {
	// Server initialization & serve
	srv := initializeServer()
	log.Info("Starting server", "port", 8000, "db_connected", true)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("failed to start server", "error", err)
		}
	}()

	// Graceful shutdown of the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info("Signal received to shutdown server", "signal", sig)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("forced shutdown", "error", err)
	}
	log.Info("Server shutdown gracefully", "error", nil)
}

func initializeServer() *http.Server {
	// Create a database pool
	pool, err := createDatabasePool()
	if err != nil {
		log.Fatal("Failed to connect to database", "error", err)
	}
	defer pool.Close()
	log.Info("Successfully created database pool")

	// Create a global repo using that pool
	repositories := repo.New(pool)

	// Setup GRPC client
	grpcClient, err := createGRPCClient()
	if err != nil {
		grpcClient = nil
		log.Fatal("AI Core gRPC not available", "error", err)
	}
	defer func() {
		if err := grpcClient.Close(); err != nil {
			log.Warn("Error while closing GRPC Client", "error", err)
		}
	}()
	log.Info("Successfully created GRPC client")

	// Create services using repositories
	services := service.NewServices(repositories)
	if grpcClient != nil {
		rag := service.NewRAGService(grpcClient)
		services.RAG = rag
		services.IngestJob = service.NewIngestJobService(repositories.IngestJob, rag)
	}

	// Create handlers using the services
	handlers := handler.NewHandlers(services)

	// Setup Router
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "/stream") {
				next.ServeHTTP(w, r)
				return
			}
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

	// Listen and serve the routes
	// CORS wraps the whole router (not chi Use) so preflight OPTIONS is
	// answered before chi's 405 handling.
	srv := &http.Server{
		Addr:    ":8000",
		Handler: middleware.CORS(r),
	}
	return srv
	// log.Info("Starting server", "port", 8000, "db_connected", true)
	// if err := http.ListenAndServe(":8000", middleware.CORS(r)); err != nil {
	// 	log.Fatal("failed to start the server!", "error", err)
	// }
}

func createDatabasePool() (*pgxpool.Pool, error) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	connString := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s",
		dbUser, dbPass, dbHost, dbPort, dbName)
	pool, err := database.NewPostgresConnection(ctx, connString)
	return pool, err
}

func createGRPCClient() (*grpcclient.Client, error) {
	grpcHost := os.Getenv("GRPC_AI_HOST")
	log.Info("Creating GRPC client on", "host", grpcHost)
	grpcClient, grpcErr := grpcclient.NewClient(grpcHost)
	return grpcClient, grpcErr
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
