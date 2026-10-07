package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"crashlens/db"
	"crashlens/diagnosis"
	"crashlens/fireworks"
	"crashlens/middleware"
	"crashlens/runner"

	"github.com/gorilla/mux"
)

type Server struct {
	DB                  *db.DB
	FWClient            *fireworks.Client
	Runner              *runner.Manager
	FireworksHTTPClient *http.Client
}

func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	mode := os.Getenv("ACCESS_MODE")
	if mode == "" {
		mode = "private"
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "auth_required": s.authRequired(mode), "access_mode": mode})
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
	id, err := s.DB.CreateWorkload(req.Name, req.Type, req.Status, middleware.RequestIdentity(r).Owner)
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
	workloads, err := s.DB.GetWorkloads(middleware.RequestIdentity(r).Owner)
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

	workload, err := s.DB.GetWorkload(id, middleware.RequestIdentity(r).Owner)
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
	existing, err := s.DB.GetWorkload(id, middleware.RequestIdentity(r).Owner)
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

	previousStatus := existing.Status
	if updates.Status == "running" && (existing.Status == "failed" || existing.Status == "succeeded") {
		http.Error(w, "Completed workloads cannot return to running", http.StatusConflict)
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

	if err := s.DB.UpdateWorkloadIfStatus(id, existing, previousStatus); err != nil {
		if errors.Is(err, db.ErrWorkloadChanged) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existing)
}

func (s *Server) DeleteWorkloadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if _, err := s.DB.GetWorkload(id, middleware.RequestIdentity(r).Owner); err != nil {
		http.Error(w, "Workload not found", 404)
		return
	}
	_, err := s.DB.Exec("DELETE FROM workloads WHERE id = ? AND owner_id = ?", id, middleware.RequestIdentity(r).Owner)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) ClearAllWorkloadsHandler(w http.ResponseWriter, r *http.Request) {
	_, err := s.DB.Exec("DELETE FROM workloads WHERE owner_id = ?", middleware.RequestIdentity(r).Owner)
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
		"gpu_oom":            "gpu_oom.py",
		"missing_checkpoint": "missing_checkpoint.py",
		"dependency_error":   "dependency_error.py",
		"data_path_error":    "data_path_error.py",
		"timeout":            "timeout.py",
		"successful":         "successful_training.py",
	}

	scriptPath, ok := templateMap[req.Template]
	if !ok {
		http.Error(w, "Invalid template name. Valid templates: gpu_oom, missing_checkpoint, dependency_error, data_path_error, timeout, successful", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		req.Name = "Test Job: " + req.Template
	}

	jobsDir := os.Getenv("JOBS_DIR")
	if jobsDir == "" {
		jobsDir = "./jobs"
		if _, err := os.Stat(jobsDir); err != nil {
			jobsDir = "../jobs"
		}
	}
	if s.Runner == nil {
		http.Error(w, "Job runner unavailable", http.StatusServiceUnavailable)
		return
	}
	id, err := s.Runner.Submit(req.Name, req.Type, filepath.Join(jobsDir, scriptPath), middleware.RequestIdentity(r).Owner)
	if err != nil {
		if errors.Is(err, runner.ErrQueueFull) || errors.Is(err, runner.ErrStopped) {
			w.Header().Set("Retry-After", "5")
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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

	workload, err := s.DB.GetWorkload(id, middleware.RequestIdentity(r).Owner)
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

	workload, err := s.DB.GetWorkload(id, middleware.RequestIdentity(r).Owner)
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
	stats, err := s.DB.GetStats(middleware.RequestIdentity(r).Owner)
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

	workload, err := s.DB.GetWorkload(id, middleware.RequestIdentity(r).Owner)
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

	// User credentials are request-scoped and never fall back to the operator account.
	var client *fireworks.Client
	key := strings.TrimSpace(r.Header.Get("X-Fireworks-API-Key"))
	model := strings.TrimSpace(r.Header.Get("X-Fireworks-Model"))
	if key != "" {
		if model == "" || len(model) > 256 || len(key) > 4096 {
			http.Error(w, "Provide a Fireworks model identifier with your Fireworks key", 400)
			return
		}
		client = fireworks.NewRequestClient(key, model)
		if s.FireworksHTTPClient != nil {
			client.Client = s.FireworksHTTPClient
		}
	} else if middleware.RequestIdentity(r).Admin {
		client = s.FWClient
	}
	report := diagnosis.RunDiagnosis(workload, client)
	if report.Source == "rules" {
		if client == nil {
			report.AIUnavailableReason = "Add your Fireworks API key and a tool-capable model to generate an AI diagnosis."
		} else {
			report.AIUnavailableReason = "Fireworks did not return a usable diagnosis. Check your key, model access, credits, and tool-calling support."
		}
	}

	// Persist both sources so reloads and MCP queries retain fallback reports.
	reportJSON, err := json.Marshal(report)
	if err != nil {
		http.Error(w, "Failed to encode diagnosis", http.StatusInternalServerError)
		return
	}
	// Update only the report and reject results if inputs changed during the AI call.
	result, err := s.DB.Exec(`UPDATE workloads SET failure_report = ? WHERE id = ? AND owner_id = ?
  AND status = 'failed' AND job_logs IS ? AND gpu_metrics IS ? AND failure_type IS ? AND runtime_seconds IS ?`,
		string(reportJSON), id, middleware.RequestIdentity(r).Owner, workload.JobLogs, workload.GPUMetrics, workload.FailureType, workload.RuntimeSeconds)
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

func (s *Server) SessionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"authenticated": true, "auth_required": s.authRequired("private"), "owner_id": middleware.RequestIdentity(r).Owner, "admin": middleware.RequestIdentity(r).Admin})
}

func (s *Server) authRequired(mode string) bool {
	hasKeys, err := s.DB.HasAPIKeys()
	return err != nil || hasKeys || (os.Getenv("CRASHLENS_API_KEY") != "" && mode != "demo")
}
func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if !middleware.RequestIdentity(r).Admin {
		http.Error(w, "Operator API key required", 403)
		return false
	}
	return true
}
func (s *Server) IssueKeyHandler(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	var input struct {
		Name    string `json:"name"`
		OwnerID string `json:"owner_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || len(input.Name) == 0 || len(input.Name) > 100 {
		http.Error(w, "A name of 1–100 characters is required", 400)
		return
	}
	// Reissue keys only for existing owners; ownership cannot be chosen by ordinary users.
	if input.OwnerID != "" {
		keys, err := s.DB.GetAPIKeys()
		if err != nil {
			http.Error(w, "Key lookup failed", 500)
			return
		}
		found := false
		for _, key := range keys {
			if key.OwnerID == input.OwnerID {
				found = true
			}
		}
		if !found {
			http.Error(w, "Owner not found", 404)
			return
		}
	}
	key, token, err := s.DB.IssueAPIKey(input.Name, input.OwnerID)
	if err != nil {
		http.Error(w, "Key creation failed", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(map[string]interface{}{"id": key.ID, "owner_id": key.OwnerID, "name": key.Name, "api_key": token})
}
func (s *Server) ListKeysHandler(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	keys, err := s.DB.GetAPIKeys()
	if err != nil {
		http.Error(w, "Key lookup failed", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keys)
}
func (s *Server) RevokeKeyHandler(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	found, err := s.DB.RevokeAPIKey(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Revocation failed", 500)
		return
	}
	if !found {
		http.Error(w, "Active key not found", 404)
		return
	}
	w.WriteHeader(204)
}
