package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"crashlens/api"
	"crashlens/db"
	"crashlens/fireworks"
	"crashlens/middleware"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	// Check for API key in production
	apiKey := os.Getenv("CRASHLENS_API_KEY")
	railwayEnv := os.Getenv("RAILWAY_ENVIRONMENT")

	if railwayEnv == "production" && apiKey == "" {
		log.Fatal("CRASHLENS_API_KEY must be set in production (RAILWAY_ENVIRONMENT=production)")
	}

	if apiKey == "" {
		log.Println("⚠️  Running without API key - ALL routes are unprotected (development mode)")
	}

	// Initialize database
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./crashlens.db"
	}
	database, err := db.New(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	// Initialize Fireworks AI client (optional)
	fwClient := fireworks.NewClient()
	if fwClient != nil {
		log.Println("Fireworks AI client initialized")
	} else {
		log.Println("Running without Fireworks AI (set FIREWORKS_API_KEY to enable)")
	}

	// Create API server
	server := &api.Server{
		DB:       database,
		FWClient: fwClient,
	}

	// Setup router
	r := mux.NewRouter()

	// Rate limiter for public write operations (10 requests per minute per IP)
	rateLimiter := middleware.NewIPRateLimiter(10)
	rateLimiter.Cleanup() // Start background cleanup

	// Public routes (read-only + safe operations)
	r.HandleFunc("/health", server.HealthHandler).Methods("GET")
	r.HandleFunc("/workloads", server.GetWorkloadsHandler).Methods("GET")
	r.HandleFunc("/workloads/{id}", server.GetWorkloadHandler).Methods("GET")
	r.HandleFunc("/workloads/{id}/logs", server.GetWorkloadLogsHandler).Methods("GET")
	r.HandleFunc("/workloads/{id}/metrics", server.GetWorkloadMetricsHandler).Methods("GET")
	r.HandleFunc("/summary", server.GetSummaryHandler).Methods("GET")

	// Rate-limited public routes
	rateLimitedRouter := r.PathPrefix("").Subrouter()
	rateLimitedRouter.Use(rateLimiter.Middleware)
	// Only runs whitelisted templates, no RCE risk, but rate limited to prevent spam
	rateLimitedRouter.HandleFunc("/workloads/run", server.RunWorkloadHandler).Methods("POST")
	// Calls Fireworks AI, rate limited to prevent credit burning
	rateLimitedRouter.HandleFunc("/workloads/{id}/diagnose", server.DiagnoseWorkloadHandler).Methods("POST")

	// Protected routes (write/delete/destructive) - require API key in production
	protectedRouter := r.PathPrefix("").Subrouter()
	protectedRouter.Use(middleware.RequireAPIKey)
	protectedRouter.HandleFunc("/workloads", server.CreateWorkloadHandler).Methods("POST")
	protectedRouter.HandleFunc("/workloads/clear", server.ClearAllWorkloadsHandler).Methods("DELETE")
	protectedRouter.HandleFunc("/workloads/{id}", server.UpdateWorkloadHandler).Methods("PUT")
	protectedRouter.HandleFunc("/workloads/{id}", server.DeleteWorkloadHandler).Methods("DELETE")

	// CORS - Allow all Vercel preview deployments by using AllowOriginFunc
	handler := cors.New(cors.Options{
		AllowOriginFunc: func(origin string) bool {
			// Allow localhost
			if origin == "http://localhost:3000" || origin == "http://localhost:3001" {
				return true
			}
			// Allow any Vercel deployment
			if strings.HasPrefix(origin, "https://frontend-") && strings.Contains(origin, ".vercel.app") {
				return true
			}
			return false
		},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization", "ngrok-skip-browser-warning"},
	}).Handler(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("CrashLens Backend starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
