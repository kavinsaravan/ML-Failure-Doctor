package api

import (
	"encoding/json"
	"net/http"

	"crashlens/db"
	"crashlens/diagnosis"
	"crashlens/fireworks"
	"crashlens/runner"

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
		Template   string `json:"template"`
		Type       string `json:"type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set default type if not provided
	if req.Type == "" {
		req.Type = "ML_JOB"
	}

	// Map template to script path if provided
	if req.Template != "" && req.ScriptPath == "" {
		templateMap := map[string]string{
			"gpu_oom":            "../jobs/gpu_oom.py",
			"missing_checkpoint": "../jobs/missing_checkpoint.py",
			"dependency_error":   "../jobs/dependency_error.py",
			"data_path_error":    "../jobs/data_path_error.py",
			"timeout":            "../jobs/timeout.py",
			"successful":         "../jobs/successful_training.py",
			"tool_loop_agent":    "../agents/tool_loop_agent.py",
		}

		if path, ok := templateMap[req.Template]; ok {
			req.ScriptPath = path
			if req.Name == "" {
				req.Name = "Test Job: " + req.Template
			}
		} else {
			http.Error(w, "Invalid template name", http.StatusBadRequest)
			return
		}
	}

	// Validate script path
	if req.ScriptPath == "" {
		http.Error(w, "script_path or template is required", http.StatusBadRequest)
		return
	}

	// Set default name if not provided
	if req.Name == "" {
		req.Name = "Workload " + req.Type
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
