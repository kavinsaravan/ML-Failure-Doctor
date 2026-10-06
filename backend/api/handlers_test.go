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
	call("PUT", "/workloads/"+id, `{"status":"succeeded","exit_code":0}`)
	diagnosed = call("POST", "/workloads/"+id+"/diagnose", "")
	if diagnosed.Code != http.StatusConflict {
		t.Fatal("success must not be diagnosed")
	}
}
