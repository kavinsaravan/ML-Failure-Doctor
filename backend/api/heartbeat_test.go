package api

import (
	"crashlens/db"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func TestHeartbeatLossAndRecoveryPreserveLifecycle(t *testing.T) {
	database, err := db.New(filepath.Join(t.TempDir(), "heartbeats.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	id, err := database.CreateWorkload("training", "ML_JOB", "running")
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{DB: database}
	router := mux.NewRouter()
	router.HandleFunc("/workloads", server.GetWorkloadsHandler).Methods("GET")
	router.HandleFunc("/workloads/{id}", server.GetWorkloadHandler).Methods("GET")
	router.HandleFunc("/workloads/{id}", server.UpdateWorkloadHandler).Methods("PUT")
	call := func(method, path, body string) *httptest.ResponseRecorder {
		result := httptest.NewRecorder()
		router.ServeHTTP(result, httptest.NewRequest(method, path, strings.NewReader(body)))
		if result.Code != 200 {
			t.Fatalf("%s %s: %d %s", method, path, result.Code, result.Body)
		}
		return result
	}
	path := fmt.Sprintf("/workloads/%d", id)
	check := func(stale bool, status string) {
		var w db.Workload
		result := call("GET", path, "")
		if err := json.Unmarshal(result.Body.Bytes(), &w); err != nil {
			t.Fatal(err)
		}
		if w.HeartbeatStale != stale || w.Status != status {
			t.Fatalf("incorrect monitoring state: %+v", w)
		}
		if stale && (w.FinishedAt != nil || w.FailureType != nil) {
			t.Fatal("lost contact was treated as a confirmed failure")
		}
		var list []db.Workload
		result = call("GET", "/workloads", "")
		if err := json.Unmarshal(result.Body.Bytes(), &list); err != nil || len(list) != 1 || list[0].HeartbeatStale != stale {
			t.Fatalf("list heartbeat state mismatch: %s", result.Body)
		}
	}
	expire := func() {
		_, err := database.Exec("UPDATE workloads SET last_heartbeat_at=? WHERE id=?", time.Now().Add(-db.HeartbeatTimeout-time.Second), id)
		if err != nil {
			t.Fatal(err)
		}
	}
	check(false, "running")
	expire()
	check(true, "running")
	// Server time is authoritative; a client cannot hide a stale heartbeat.
	result := httptest.NewRecorder()
	router.ServeHTTP(result, httptest.NewRequest("PUT", path, strings.NewReader(`{"last_heartbeat_at":"2099-01-01T00:00:00Z"}`)))
	if result.Code != 400 {
		t.Fatal("client-controlled heartbeat accepted")
	}
	check(true, "running")
	call("PUT", path, `{"status":"running"}`)
	check(false, "running")
	expire()
	// An outbox replay must still finalize a job that was stale during an outage.
	call("PUT", path, `{"status":"succeeded","runtime_seconds":10,"exit_code":0}`)
	check(false, "succeeded")
}
