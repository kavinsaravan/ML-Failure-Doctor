package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/cors"
)

type Workload struct {
	ID                 int       `json:"id"`
	Name               string    `json:"name"`
	Type               string    `json:"type"` // ML_JOB or AGENT_RUN
	Status             string    `json:"status"` // pending, running, failed, succeeded
	FailureType        *string   `json:"failure_type,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	StartedAt          *time.Time `json:"started_at,omitempty"`
	FinishedAt         *time.Time `json:"finished_at,omitempty"`
	RuntimeSeconds     *float64  `json:"runtime_seconds,omitempty"`
	ExitCode           *int      `json:"exit_code,omitempty"`
	WastedGPUSeconds   *float64  `json:"wasted_gpu_seconds,omitempty"`
	JobLogs            *string   `json:"job_logs,omitempty"`
	GPUMetrics         *string   `json:"gpu_metrics,omitempty"`
	CheckpointState    *string   `json:"checkpoint_state,omitempty"`
	FailureReport      *string   `json:"failure_report,omitempty"`
	AgentSteps         *string   `json:"agent_steps,omitempty"`
	ToolCalls          *string   `json:"tool_calls,omitempty"`
	ModelCalls         *string   `json:"model_calls,omitempty"`
	TraceEvents        *string   `json:"trace_events,omitempty"`
}

type DiagnosisRequest struct {
	WorkloadID int `json:"workload_id"`
}

type DiagnosisResponse struct {
	RootCause       string   `json:"root_cause"`
	Evidence        []string `json:"evidence"`
	RecommendedFix  string   `json:"recommended_fix"`
	SafeToRetry     bool     `json:"safe_to_retry"`
	DiagnosedAt     time.Time `json:"diagnosed_at"`
}

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./crashlens.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize database
	if err := initDB(); err != nil {
		log.Fatal(err)
	}

	// Create router
	r := mux.NewRouter()

	// API routes
	r.HandleFunc("/api/workloads", getWorkloads).Methods("GET")
	r.HandleFunc("/api/workloads", createWorkload).Methods("POST")
	r.HandleFunc("/api/workloads/{id}", getWorkload).Methods("GET")
	r.HandleFunc("/api/workloads/{id}", updateWorkload).Methods("PUT")
	r.HandleFunc("/api/workloads/{id}/diagnose", diagnoseWorkload).Methods("POST")
	r.HandleFunc("/api/stats", getStats).Methods("GET")

	// Health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}).Methods("GET")

	// CORS
	handler := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	}).Handler(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func initDB() error {
	schema := `
	CREATE TABLE IF NOT EXISTS workloads (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		status TEXT NOT NULL,
		failure_type TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		started_at TIMESTAMP,
		finished_at TIMESTAMP,
		runtime_seconds REAL,
		exit_code INTEGER,
		wasted_gpu_seconds REAL,
		job_logs TEXT,
		gpu_metrics TEXT,
		checkpoint_state TEXT,
		failure_report TEXT,
		agent_steps TEXT,
		tool_calls TEXT,
		model_calls TEXT,
		trace_events TEXT
	);
	`
	_, err := db.Exec(schema)
	return err
}

func getWorkloads(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT id, name, type, status, failure_type, created_at, started_at,
		       finished_at, runtime_seconds, exit_code, wasted_gpu_seconds
		FROM workloads
		ORDER BY created_at DESC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	workloads := []Workload{}
	for rows.Next() {
		var w Workload
		err := rows.Scan(&w.ID, &w.Name, &w.Type, &w.Status, &w.FailureType,
			&w.CreatedAt, &w.StartedAt, &w.FinishedAt, &w.RuntimeSeconds,
			&w.ExitCode, &w.WastedGPUSeconds)
		if err != nil {
			log.Println("Error scanning row:", err)
			continue
		}
		workloads = append(workloads, w)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workloads)
}

func getWorkload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var workload Workload
	err := db.QueryRow(`
		SELECT id, name, type, status, failure_type, created_at, started_at,
		       finished_at, runtime_seconds, exit_code, wasted_gpu_seconds,
		       job_logs, gpu_metrics, checkpoint_state, failure_report,
		       agent_steps, tool_calls, model_calls, trace_events
		FROM workloads WHERE id = ?
	`, id).Scan(
		&workload.ID, &workload.Name, &workload.Type, &workload.Status,
		&workload.FailureType, &workload.CreatedAt, &workload.StartedAt,
		&workload.FinishedAt, &workload.RuntimeSeconds, &workload.ExitCode,
		&workload.WastedGPUSeconds, &workload.JobLogs, &workload.GPUMetrics,
		&workload.CheckpointState, &workload.FailureReport, &workload.AgentSteps,
		&workload.ToolCalls, &workload.ModelCalls, &workload.TraceEvents,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Workload not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workload)
}

func createWorkload(w http.ResponseWriter, r *http.Request) {
	var workload Workload
	if err := json.NewDecoder(r.Body).Decode(&workload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := db.Exec(`
		INSERT INTO workloads (name, type, status)
		VALUES (?, ?, ?)
	`, workload.Name, workload.Type, workload.Status)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	workload.ID = int(id)
	workload.CreatedAt = time.Now()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(workload)
}

func updateWorkload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var workload Workload
	if err := json.NewDecoder(r.Body).Decode(&workload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := db.Exec(`
		UPDATE workloads
		SET name = ?, type = ?, status = ?, failure_type = ?,
		    started_at = ?, finished_at = ?, runtime_seconds = ?,
		    exit_code = ?, wasted_gpu_seconds = ?,
		    job_logs = ?, gpu_metrics = ?, checkpoint_state = ?,
		    failure_report = ?, agent_steps = ?, tool_calls = ?,
		    model_calls = ?, trace_events = ?
		WHERE id = ?
	`, workload.Name, workload.Type, workload.Status, workload.FailureType,
		workload.StartedAt, workload.FinishedAt, workload.RuntimeSeconds,
		workload.ExitCode, workload.WastedGPUSeconds, workload.JobLogs,
		workload.GPUMetrics, workload.CheckpointState, workload.FailureReport,
		workload.AgentSteps, workload.ToolCalls, workload.ModelCalls,
		workload.TraceEvents, id)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func diagnoseWorkload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Get workload data
	var workload Workload
	err := db.QueryRow(`
		SELECT id, name, type, status, failure_type, job_logs, gpu_metrics,
		       checkpoint_state, agent_steps, tool_calls, trace_events
		FROM workloads WHERE id = ?
	`, id).Scan(
		&workload.ID, &workload.Name, &workload.Type, &workload.Status,
		&workload.FailureType, &workload.JobLogs, &workload.GPUMetrics,
		&workload.CheckpointState, &workload.AgentSteps, &workload.ToolCalls,
		&workload.TraceEvents,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Run diagnosis (will call Gemma via Fireworks AI)
	diagnosis := runDiagnosis(workload)

	// Store diagnosis in database
	diagnosisJSON, _ := json.Marshal(diagnosis)
	diagnosisStr := string(diagnosisJSON)

	_, err = db.Exec(`
		UPDATE workloads SET failure_report = ? WHERE id = ?
	`, diagnosisStr, id)

	if err != nil {
		log.Println("Error updating failure report:", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(diagnosis)
}

func getStats(w http.ResponseWriter, r *http.Request) {
	stats := make(map[string]interface{})

	// Total workloads
	var total int
	db.QueryRow("SELECT COUNT(*) FROM workloads").Scan(&total)
	stats["total_workloads"] = total

	// Failed workloads
	var failed int
	db.QueryRow("SELECT COUNT(*) FROM workloads WHERE status = 'failed'").Scan(&failed)
	stats["failed_workloads"] = failed

	// Total wasted GPU seconds
	var wastedGPU float64
	db.QueryRow("SELECT COALESCE(SUM(wasted_gpu_seconds), 0) FROM workloads WHERE status = 'failed'").Scan(&wastedGPU)
	stats["wasted_gpu_seconds"] = wastedGPU

	// Failure types breakdown
	rows, _ := db.Query("SELECT failure_type, COUNT(*) FROM workloads WHERE failure_type IS NOT NULL GROUP BY failure_type")
	defer rows.Close()

	failureTypes := make(map[string]int)
	for rows.Next() {
		var ftype string
		var count int
		rows.Scan(&ftype, &count)
		failureTypes[ftype] = count
	}
	stats["failure_types"] = failureTypes

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
