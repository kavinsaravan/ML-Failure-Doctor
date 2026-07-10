package main

import (
	"log"
	"net/http"
	"os"

	"crashlens/api"
	"crashlens/db"
	"crashlens/fireworks"

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

	// API routes
	r.HandleFunc("/health", server.HealthHandler).Methods("GET")
	r.HandleFunc("/workloads", server.CreateWorkloadHandler).Methods("POST")
	r.HandleFunc("/workloads", server.GetWorkloadsHandler).Methods("GET")
	r.HandleFunc("/workloads/{id}", server.GetWorkloadHandler).Methods("GET")
	r.HandleFunc("/workloads/run", server.RunWorkloadHandler).Methods("POST")
	r.HandleFunc("/workloads/{id}/logs", server.GetWorkloadLogsHandler).Methods("GET")
	r.HandleFunc("/workloads/{id}/metrics", server.GetWorkloadMetricsHandler).Methods("GET")
	r.HandleFunc("/workloads/{id}/diagnose", server.DiagnoseWorkloadHandler).Methods("POST")
	r.HandleFunc("/summary", server.GetSummaryHandler).Methods("GET")

	// Agent Observability routes
	r.HandleFunc("/agent-runs", server.CreateAgentRunHandler).Methods("POST")
	r.HandleFunc("/agent-runs", server.GetAgentRunsHandler).Methods("GET")
	r.HandleFunc("/agent-runs/{id}", server.GetAgentRunHandler).Methods("GET")
	r.HandleFunc("/agent-runs/{id}", server.UpdateAgentRunHandler).Methods("PUT")
	r.HandleFunc("/agent-runs/{id}/steps", server.GetAgentRunStepsHandler).Methods("GET")
	r.HandleFunc("/agent-runs/{id}/diagnose", server.DiagnoseAgentRunHandler).Methods("POST")
	r.HandleFunc("/agent-steps", server.CreateAgentStepHandler).Methods("POST")

	// CORS
	handler := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:3000", "http://localhost:3001"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	}).Handler(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("CrashLens Backend starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
