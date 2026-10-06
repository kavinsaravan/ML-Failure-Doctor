package main

import (
	"context"
	"crashlens/api"
	"crashlens/db"
	"crashlens/fireworks"
	"crashlens/middleware"
	"crashlens/runner"
	"errors"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func positiveEnv(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		log.Fatalf("%s must be a positive integer", name)
	}
	return n
}
func buildRouter(server *api.Server, limiter *middleware.IPRateLimiter, mode string) http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/health", server.HealthHandler).Methods("GET")
	session := r.PathPrefix("/session").Subrouter()
	session.Use(middleware.RequireAPIKey)
	session.HandleFunc("", server.SessionHandler).Methods("GET")
	reads := r.NewRoute().Subrouter()
	if mode != "demo" {
		reads.Use(middleware.RequireAPIKey)
	}
	reads.HandleFunc("/workloads", server.GetWorkloadsHandler).Methods("GET")
	reads.HandleFunc("/workloads/{id}", server.GetWorkloadHandler).Methods("GET")
	reads.HandleFunc("/workloads/{id}/logs", server.GetWorkloadLogsHandler).Methods("GET")
	reads.HandleFunc("/workloads/{id}/metrics", server.GetWorkloadMetricsHandler).Methods("GET")
	reads.HandleFunc("/summary", server.GetSummaryHandler).Methods("GET")
	writes := r.NewRoute().Subrouter()
	writes.Use(middleware.RequireAPIKey)
	writes.HandleFunc("/workloads", server.CreateWorkloadHandler).Methods("POST")
	writes.HandleFunc("/workloads/clear", server.ClearAllWorkloadsHandler).Methods("DELETE")
	writes.HandleFunc("/workloads/{id}", server.UpdateWorkloadHandler).Methods("PUT")
	writes.HandleFunc("/workloads/{id}", server.DeleteWorkloadHandler).Methods("DELETE")
	expensive := r.NewRoute().Subrouter()
	expensive.Use(middleware.RequireAPIKey, limiter.Middleware)
	expensive.HandleFunc("/workloads/run", server.RunWorkloadHandler).Methods("POST")
	expensive.HandleFunc("/workloads/{id}/diagnose", server.DiagnoseWorkloadHandler).Methods("POST")
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		request.Body = http.MaxBytesReader(w, request.Body, 1024*1024)
		r.ServeHTTP(w, request)
	})
}
func main() {
	if (os.Getenv("APP_ENV") == "production" || os.Getenv("RAILWAY_ENVIRONMENT") == "production") && os.Getenv("CRASHLENS_API_KEY") == "" {
		log.Fatal("CRASHLENS_API_KEY is required in production")
	}
	if os.Getenv("CRASHLENS_API_KEY") == "" {
		log.Println("Development access: no API key configured")
	}
	mode := os.Getenv("ACCESS_MODE")
	if mode == "" {
		mode = "private"
	}
	if mode != "private" && mode != "demo" {
		log.Fatal("ACCESS_MODE must be private or demo")
	}
	path := os.Getenv("DATABASE_PATH")
	if path == "" {
		path = "./crashlens.db"
	}
	database, err := db.New(path)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()
	if err := database.RecoverManagedJobs("backend restarted before completion"); err != nil {
		log.Fatal(err)
	}
	manager := runner.NewManager(database, positiveEnv("JOB_CONCURRENCY", 2), positiveEnv("JOB_QUEUE_SIZE", 16), time.Duration(positiveEnv("JOB_TIMEOUT_SECONDS", 300))*time.Second)
	defer manager.Close()
	server := &api.Server{DB: database, FWClient: fireworks.NewClient(), Runner: manager}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	limiter := middleware.NewIPRateLimiter(positiveEnv("JOB_REQUESTS_PER_MINUTE", 60))
	if err := limiter.SetTrustedProxies(strings.Split(os.Getenv("TRUSTED_PROXY_CIDRS"), ",")); err != nil {
		log.Fatal(err)
	}
	limiter.Cleanup(ctx)
	origins := os.Getenv("ALLOWED_ORIGINS")
	if origins == "" {
		origins = "http://localhost:3000,http://localhost:3001"
	}
	handler := cors.New(cors.Options{AllowedOrigins: strings.Split(origins, ","), AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, AllowedHeaders: []string{"Content-Type", "Authorization", "ngrok-skip-browser-warning"}}).Handler(buildRouter(server, limiter, mode))
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	httpServer := &http.Server{Addr: ":" + port, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 90 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		httpServer.Shutdown(shutdown)
	}()
	log.Printf("CrashLens backend on port %s (%s access)", port, mode)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
