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
	// Initialize database
	database, err := db.New("./crashlens.db")
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

	// Public routes (read-only + safe operations)
	r.HandleFunc("/health", server.HealthHandler).Methods("GET")
	r.HandleFunc("/workloads", server.GetWorkloadsHandler).Methods("GET")
	r.HandleFunc("/workloads/{id}", server.GetWorkloadHandler).Methods("GET")
	r.HandleFunc("/workloads/{id}/logs", server.GetWorkloadLogsHandler).Methods("GET")
	r.HandleFunc("/workloads/{id}/metrics", server.GetWorkloadMetricsHandler).Methods("GET")
	r.HandleFunc("/summary", server.GetSummaryHandler).Methods("GET")
	// Safe to leave public: only runs whitelisted templates, no RCE risk
	r.HandleFunc("/workloads/run", server.RunWorkloadHandler).Methods("POST")
	// Safe to leave public: read-only diagnosis (consider adding rate limiting)
	r.HandleFunc("/workloads/{id}/diagnose", server.DiagnoseWorkloadHandler).Methods("POST")

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
