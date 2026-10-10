package api

import (
	"crashlens/db"
	"github.com/gorilla/mux"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidationBoundary(t *testing.T) {
	database, err := db.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	s := &Server{DB: database}
	r := mux.NewRouter()
	r.HandleFunc("/workloads", s.CreateWorkloadHandler).Methods("POST")
	r.HandleFunc("/workloads/{id}", s.UpdateWorkloadHandler).Methods("PUT")
	call := func(method, path, body string, code int) {
		t.Helper()
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
		if w.Code != code {
			t.Fatalf("%s: got %d: %s", body, w.Code, w.Body.String())
		}
	}
	for _, body := range []string{`{"name":" "}`, `{"name":"test","status":"invalid"}`, `{"name":"test","type":"unknown"}`} {
		call("POST", "/workloads", body, 400)
	}
	call("POST", "/workloads", `{"name":"test","status":"running"}`, 201)
	for _, body := range []string{`{"status":"invalid"}`, `{"runtime_seconds":-1}`, `{"gpu_metrics":"{}"}`, `{"gpu_metrics":"null"}`, `{"gpu_metrics":"[null]"}`, `{"gpu_metrics":"[1]"}`, `{"gpu_metrics":"[{}]"}`, `{"gpu_metrics":"[{\"gpu_memory_percent\":\"bad\"}]"}`, `{"wasted_gpu_seconds":-2}`} {
		call("PUT", "/workloads/1", body, 400)
	}
	call("PUT", "/workloads/1", `{"status":"pending"}`, 409)
	call("PUT", "/workloads/1", `{"gpu_metrics":"[{\"gpu_memory_used_mb\":12,\"gpu_memory_total_mb\":10,\"gpu_memory_percent\":120,\"gpu_utilization_percent\":null,\"source\":\"torch.mps\"}]"}`, 200)
	call("PUT", "/workloads/1", `{"status":"failed"}`, 200)
	for _, status := range []string{"running", "pending", "succeeded"} {
		call("PUT", "/workloads/1", `{"status":"`+status+`"}`, 409)
	}
	call("PUT", "/workloads/1", `{"status":"failed","job_logs":"final logs"}`, 200)
}
