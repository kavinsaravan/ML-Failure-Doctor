package api

import (
	"crashlens/db"
	"crashlens/diagnosis"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestSDKLifecycleAndFallbackPersistence(t *testing.T) {
	database, err := db.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	s := &Server{DB: database}
	r := mux.NewRouter()
	r.HandleFunc("/workloads", s.CreateWorkloadHandler).Methods("POST")
	r.HandleFunc("/workloads/{id}", s.UpdateWorkloadHandler).Methods("PUT")
	r.HandleFunc("/workloads/{id}/logs", s.GetWorkloadLogsHandler).Methods("GET")
	r.HandleFunc("/workloads/{id}/metrics", s.GetWorkloadMetricsHandler).Methods("GET")
	r.HandleFunc("/workloads/{id}/diagnose", s.DiagnoseWorkloadHandler).Methods("POST")
	call := func(method, path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
		return w
	}
	created := call("POST", "/workloads", `{"name":"SDK training","status":"running"}`)
	if created.Code != 201 {
		t.Fatal(created.Body.String())
	}
	var response struct {
		ID int `json:"id"`
	}
	json.Unmarshal(created.Body.Bytes(), &response)
	id := fmt.Sprint(response.ID)
	w, err := database.GetWorkload(id)
	if err != nil || w.StartedAt == nil {
		t.Fatalf("missing start: %+v %v", w, err)
	}
	metrics := `[{"source":"test-fixture","gpu_memory_used_mb":128,"gpu_memory_total_mb":1024,"gpu_memory_percent":12.5}]`
	upload, _ := json.Marshal(map[string]string{"job_logs": "Training step 1", "gpu_metrics": metrics})
	if live := call("PUT", "/workloads/"+id, string(upload)); live.Code != 200 {
		t.Fatal(live.Body.String())
	}
	for _, field := range []struct{ endpoint, key, expected string }{
		{"logs", "logs", "Training step 1"}, {"metrics", "metrics", metrics},
	} {
		result := call("GET", "/workloads/"+id+"/"+field.endpoint, "")
		var body map[string]interface{}
		if result.Code != 200 || json.Unmarshal(result.Body.Bytes(), &body) != nil || body[field.key] != field.expected {
			t.Fatalf("SDK telemetry not readable: %s", result.Body.String())
		}
	}
	updated := call("PUT", "/workloads/"+id, `{"status":"failed","runtime_seconds":12.5,"exit_code":1,"job_logs":"ModuleNotFoundError: no module named torch"}`)
	if updated.Code != 200 {
		t.Fatal(updated.Body.String())
	}
	w, err = database.GetWorkload(id)
	if err != nil || w.FinishedAt == nil || w.FailureType == nil || *w.FailureType != "DEPENDENCY_ERROR" || w.WastedGPUSeconds == nil || *w.WastedGPUSeconds != 12.5 {
		t.Fatalf("incomplete lifecycle: %+v %v", w, err)
	}
	diagnosed := call("POST", "/workloads/"+id+"/diagnose", "")
	if diagnosed.Code != 200 {
		t.Fatal(diagnosed.Body.String())
	}
	w, _ = database.GetWorkload(id)
	if w.FailureReport == nil {
		t.Fatal("fallback not persisted")
	}
	var report diagnosis.Report
	if err := json.Unmarshal([]byte(*w.FailureReport), &report); err != nil {
		t.Fatal(err)
	}
	if report.Source != "rules" || report.SafeToRetry {
		t.Fatalf("incorrect fallback: %+v", report)
	}
	stats, err := database.GetStats()
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := stats["gpu_platform"]; exists {
		t.Fatal("ambiguous GPU field still present")
	}
	if stats["wasted_gpu_seconds"] != 12.5 || stats["failure_types"].(map[string]int)["DEPENDENCY_ERROR"] != 1 {
		t.Fatal(stats)
	}
	changed := call("PUT", "/workloads/"+id, `{"job_logs":"RuntimeError: CUDA out of memory"}`)
	if changed.Code != 200 {
		t.Fatal(changed.Body.String())
	}
	w, _ = database.GetWorkload(id)
	if w.FailureReport != nil || w.FailureType == nil || *w.FailureType != "GPU_OUT_OF_MEMORY" {
		t.Fatal("stale report or classification")
	}
	success := call("POST", "/workloads", `{"name":"successful job","status":"running"}`)
	if success.Code != 201 {
		t.Fatal(success.Body.String())
	}
	json.Unmarshal(success.Body.Bytes(), &response)
	id = fmt.Sprint(response.ID)
	if done := call("PUT", "/workloads/"+id, `{"status":"succeeded","exit_code":0}`); done.Code != 200 {
		t.Fatal(done.Body.String())
	}
	late := call("PUT", "/workloads/"+id, `{"status":"running","job_logs":"late upload"}`)
	if late.Code != http.StatusConflict {
		t.Fatal("late uploader revived completed workload")
	}
	diagnosed = call("POST", "/workloads/"+id+"/diagnose", "")
	if diagnosed.Code != http.StatusConflict {
		t.Fatal("success must not be diagnosed")
	}
}

func TestRegistrationAndTerminalRetriesPreserveCompletedWorkload(t *testing.T) {
	d, err := db.New(filepath.Join(t.TempDir(), "retry.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	s := &Server{DB: d}
	r := mux.NewRouter()
	r.HandleFunc("/workloads", s.CreateWorkloadHandler).Methods("POST")
	r.HandleFunc("/workloads/{id}", s.UpdateWorkloadHandler).Methods("PUT")
	call := func(method, path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
		return w
	}
	registration := `{"name":"retryable training","status":"running","client_run_id":"same-run"}`
	first := call("POST", "/workloads", registration)
	if first.Code != 201 {
		t.Fatal(first.Body)
	}
	terminal := `{"status":"failed","runtime_seconds":4,"job_logs":"CUDA out of memory","wasted_gpu_seconds":8}`
	if w := call("PUT", "/workloads/1", terminal); w.Code != 200 {
		t.Fatal(w.Body)
	}
	report := `{"source":"ai","root_cause":"Saved explanation"}`
	if _, err := d.Exec("UPDATE workloads SET failure_report=? WHERE id=1", report); err != nil {
		t.Fatal(err)
	}
	retry := call("POST", "/workloads", registration)
	if retry.Code != 200 {
		t.Fatal(retry.Body)
	}
	var existing db.Workload
	if err := json.Unmarshal(retry.Body.Bytes(), &existing); err != nil || existing.ID != 1 || existing.Status != "failed" {
		t.Fatal("registration retry revived or duplicated workload", retry.Body)
	}
	// A lost PUT response can also be retried without discarding an existing report.
	if w := call("PUT", "/workloads/1", `{"status":"failed","runtime_seconds":4,"job_logs":"CUDA out of memory"}`); w.Code != 200 {
		t.Fatal(w.Body)
	}
	after, err := d.GetWorkload("1")
	if err != nil || after.FailureReport == nil || *after.FailureReport != report || after.WastedGPUSeconds == nil || *after.WastedGPUSeconds != 8 {
		t.Fatal("identical terminal retry changed saved diagnosis or measured waste", err)
	}
	if w := call("POST", "/workloads", `{"name":"different name","client_run_id":"same-run"}`); w.Code != 409 {
		t.Fatal("reused ID with different identity was accepted", w.Code)
	}
	if w := call("POST", "/workloads", `{"name":"invalid run","client_run_id":"`+strings.Repeat("x", 129)+`"}`); w.Code != 400 {
		t.Fatal("oversized client run ID was accepted")
	}
	list, err := d.GetWorkloads()
	if err != nil || len(list) != 1 {
		t.Fatal("retry created a second job", err)
	}
}
