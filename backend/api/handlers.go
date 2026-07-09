package api

import (
	"encoding/json"
	"net/http"

	"crashlens/backend/db"
	"crashlens/backend/diagnosis"
	"crashlens/backend/fireworks"
	"crashlens/backend/runner"

	"github.com/gorilla/mux"
)

type Server struct {
	DB       *db.DB
	FWClient *fireworks.Client
}

func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) GetWorkloadsHandler(w http.ResponseWriter, r *http.Request) {
	workloads, err := s.DB.GetWorkloads()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workloads)
}

func (s *Server) GetWorkloadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	workload, err := s.DB.GetWorkload(id)
	if err != nil {
		http.Error(w, "Workload not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workload)
}

func (s *Server) RunWorkloadHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string `json:"name"`
		ScriptPath string `json:"script_path"`
		Type       string `json:"type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create workload entry
	id, err := s.DB.CreateWorkload(req.Name, req.Type, "pending")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Run job asynchronously
	go runner.RunPythonJob(req.ScriptPath, int(id), s.DB)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"workload_id": id,
		"status":      "pending",
		"message":     "Job queued for execution",
	})
}

func (s *Server) GetWorkloadLogsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	workload, err := s.DB.GetWorkload(id)
	if err != nil {
		http.Error(w, "Workload not found", http.StatusNotFound)
		return
	}

	logs := ""
	if workload.JobLogs != nil {
		logs = *workload.JobLogs
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"workload_id": workload.ID,
		"name":        workload.Name,
		"logs":        logs,
	})
}

func (s *Server) GetWorkloadMetricsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	workload, err := s.DB.GetWorkload(id)
	if err != nil {
		http.Error(w, "Workload not found", http.StatusNotFound)
		return
	}

	metrics := ""
	if workload.GPUMetrics != nil {
		metrics = *workload.GPUMetrics
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"workload_id": workload.ID,
		"name":        workload.Name,
		"metrics":     metrics,
	})
}

func (s *Server) GetSummaryHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := s.DB.GetStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) DiagnoseWorkloadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	workload, err := s.DB.GetWorkload(id)
	if err != nil {
		http.Error(w, "Workload not found", http.StatusNotFound)
		return
	}

	// Run diagnosis
	report := diagnosis.RunDiagnosis(workload, s.FWClient)

	// Store diagnosis in database
	reportJSON, _ := json.Marshal(report)
	reportStr := string(reportJSON)
	workload.FailureReport = &reportStr
	s.DB.UpdateWorkload(id, workload)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}
