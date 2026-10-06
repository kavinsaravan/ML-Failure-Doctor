package api

import (
	"encoding/json"
	"log"
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

func (s *Server) CreateWorkloadHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string `json:"name"`
		Type   string `json:"type"`
		Status string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.Type == "" {
		req.Type = "ML_JOB"
	}
	if req.Status == "" {
		req.Status = "pending"
	}

	// Create workload
	id, err := s.DB.CreateWorkload(req.Name, req.Type, req.Status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     id,
		"name":   req.Name,
		"type":   req.Type,
		"status": req.Status,
	})
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

func (s *Server) UpdateWorkloadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Get existing workload
	existing, err := s.DB.GetWorkload(id)
	if err != nil {
		http.Error(w, "Workload not found", http.StatusNotFound)
		return
	}

	// Decode updates
	var updates db.Workload
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Reports are invalidated whenever their input changes.
	if (updates.Status != "" && updates.Status != existing.Status) || updates.JobLogs != nil || updates.GPUMetrics != nil || updates.FailureType != nil || updates.RuntimeSeconds != nil {
		existing.FailureReport = nil
		if updates.FailureType == nil {
			existing.FailureType = nil
		}
	}
	if updates.StartedAt != nil {
		existing.StartedAt = updates.StartedAt
	}
	if updates.FinishedAt != nil {
		existing.FinishedAt = updates.FinishedAt
	}
	if updates.CheckpointState != nil {
		existing.CheckpointState = updates.CheckpointState
	}
	// Merge updates with existing data (only update non-nil fields)
	if updates.Status != "" {
		existing.Status = updates.Status
	}
	if updates.FailureType != nil {
		existing.FailureType = updates.FailureType
	}
	if updates.RuntimeSeconds != nil {
		if *updates.RuntimeSeconds < 0 {
			http.Error(w, "runtime_seconds cannot be negative", http.StatusBadRequest)
			return
		}
		existing.RuntimeSeconds = updates.RuntimeSeconds
		if updates.WastedGPUSeconds == nil {
			existing.WastedGPUSeconds = nil
		}
	}
	if updates.ExitCode != nil {
		existing.ExitCode = updates.ExitCode
	}
	if updates.WastedGPUSeconds != nil {
		existing.WastedGPUSeconds = updates.WastedGPUSeconds
	}
	if updates.JobLogs != nil {
		existing.JobLogs = updates.JobLogs
	}
	if updates.GPUMetrics != nil {
		existing.GPUMetrics = updates.GPUMetrics
	}

	if err := s.DB.UpdateWorkload(id, existing); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existing)
}

func (s *Server) DeleteWorkloadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	_, err := s.DB.Exec("DELETE FROM workloads WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) ClearAllWorkloadsHandler(w http.ResponseWriter, r *http.Request) {
	_, err := s.DB.Exec("DELETE FROM workloads")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "All workloads cleared successfully",
	})
}

func (s *Server) RunWorkloadHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Template string `json:"template"`
		Type     string `json:"type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set default type if not provided
	if req.Type == "" {
		req.Type = "ML_JOB"
	}

	// SECURITY: Only allow predefined templates, never accept arbitrary script paths
	templateMap := map[string]string{
		"gpu_oom":            "./jobs/gpu_oom.py",
		"missing_checkpoint": "./jobs/missing_checkpoint.py",
		"dependency_error":   "./jobs/dependency_error.py",
		"data_path_error":    "./jobs/data_path_error.py",
		"timeout":            "./jobs/timeout.py",
		"successful":         "./jobs/successful_training.py",
	}

	scriptPath, ok := templateMap[req.Template]
	if !ok {
		http.Error(w, "Invalid template name. Valid templates: gpu_oom, missing_checkpoint, dependency_error, data_path_error, timeout, successful", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		req.Name = "Test Job: " + req.Template
	}

	// Create workload entry
	id, err := s.DB.CreateWorkload(req.Name, req.Type, "pending")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Run job asynchronously
	go func() {
		if _, err := runner.RunPythonJob(scriptPath, int(id), s.DB); err != nil {
			log.Printf("Job %d failed: %v", id, err)
		}
	}()

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

	if workload.Status != "failed" {
		http.Error(w, "Only failed workloads can be diagnosed", http.StatusConflict)
		return
	}
	// If AI diagnosis already exists, return cached result (idempotent)
	// Only cache AI results - rule-based fallback may improve with code changes
	if r.URL.Query().Get("refresh") != "true" && workload.FailureReport != nil && *workload.FailureReport != "" {
		var cachedReport diagnosis.Report
		if err := json.Unmarshal([]byte(*workload.FailureReport), &cachedReport); err == nil {
			if cachedReport.Source == "ai" {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(cachedReport)
				return
			}
		}
	}

	// Run diagnosis (may call AI model which costs money)
	report := diagnosis.RunDiagnosis(workload, s.FWClient)

	// Persist both sources so reloads and MCP queries retain fallback reports.
	reportJSON, err := json.Marshal(report)
	if err != nil {
		http.Error(w, "Failed to encode diagnosis", http.StatusInternalServerError)
		return
	}
	// Update only the report and reject results if inputs changed during the AI call.
	result, err := s.DB.Exec(`UPDATE workloads SET failure_report = ? WHERE id = ?
  AND status = 'failed' AND job_logs IS ? AND gpu_metrics IS ? AND failure_type IS ? AND runtime_seconds IS ?`,
		string(reportJSON), id, workload.JobLogs, workload.GPUMetrics, workload.FailureType, workload.RuntimeSeconds)
	if err != nil {
		http.Error(w, "Failed to save diagnosis", http.StatusInternalServerError)
		return
	}
	count, err := result.RowsAffected()
	if err != nil || count == 0 {
		http.Error(w, "Workload changed during diagnosis; retry", http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}
