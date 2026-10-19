package main

import (
	"crashlens/api"
	"crashlens/db"
	"crashlens/fireworks"
	"crashlens/middleware"
	"crashlens/runner"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIndividualKeysIsolateEveryEndpoint(t *testing.T) {
	t.Setenv("CRASHLENS_API_KEY", "operator")
	d, err := db.New(filepath.Join(t.TempDir(), "keys.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	keyA, tokenA, err := d.IssueAPIKey("Alice", "")
	if err != nil {
		t.Fatal(err)
	}
	_, tokenB, err := d.IssueAPIKey("Bob", "")
	if err != nil {
		t.Fatal(err)
	}
	alice, err := d.CreateWorkload("Alice secret", "ML_JOB", "failed", keyA.OwnerID)
	if err != nil {
		t.Fatal(err)
	}
	legacy, _ := d.CreateWorkload("Legacy", "ML_JOB", "failed")
	// Any accidental cross-owner AI request will be counted.
	aiCalls := 0
	fw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { aiCalls++; w.WriteHeader(500) }))
	defer fw.Close()
	manager := runner.NewManager(d, 1, 2, time.Second)
	defer manager.Close()
	h := buildRouter(&api.Server{DB: d, Runner: manager, FWClient: &fireworks.Client{BaseURL: fw.URL, Client: fw.Client()}}, middleware.NewIPRateLimiter(600), "demo")
	request := func(method, path, token, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	for _, suffix := range []string{"", "/logs", "/metrics"} {
		if w := request("GET", fmt.Sprintf("/workloads/%d%s", alice, suffix), tokenB, ""); w.Code != 404 {
			t.Fatalf("cross-owner read: %d %s", w.Code, w.Body)
		}
	}
	for _, route := range []struct{ method, suffix string }{{"PUT", ""}, {"DELETE", ""}, {"POST", "/diagnose"}} {
		w := request(route.method, fmt.Sprintf("/workloads/%d%s", alice, route.suffix), tokenB, `{"status":"succeeded","owner_id":"legacy"}`)
		if w.Code != 404 {
			t.Fatalf("cross-owner mutation: %d", w.Code)
		}
	}
	if aiCalls != 0 {
		t.Fatal("unauthorized diagnosis reached AI")
	}
	for _, token := range []string{tokenA, tokenB, "operator"} {
		w := request("GET", "/workloads", token, "")
		if w.Code != 200 {
			t.Fatal(w.Body)
		}
		var workloads []db.Workload
		json.Unmarshal(w.Body.Bytes(), &workloads)
		expected := 0
		if token != tokenB {
			expected = 1
		}
		if len(workloads) != expected {
			t.Fatalf("leaked list: %s", w.Body)
		}
	}
	if w := request("GET", "/summary", tokenB, ""); !strings.Contains(w.Body.String(), `"total_workloads":0`) {
		t.Fatal(w.Body)
	}
	if w := request("GET", "/workloads", "", ""); w.Code != 401 {
		t.Fatal("demo leaked multi-user data")
	}
	if w := request("POST", "/api-keys", tokenA, `{"name":"intruder"}`); w.Code != 403 {
		t.Fatal("user issued key")
	}
	if w := request("GET", "/api-keys", tokenA, ""); w.Code != 403 {
		t.Fatal("user listed keys")
	}
	if w := request("DELETE", "/api-keys/"+keyA.ID, tokenB, ""); w.Code != 403 {
		t.Fatal("user revoked key")
	}
	// Creating with a forged owner never changes server-owned identity.
	w := request("POST", "/workloads", tokenB, `{"name":"Bob","owner_id":"legacy"}`)
	if w.Code != 201 {
		t.Fatal(w.Body)
	}
	var created struct{ ID int }
	json.Unmarshal(w.Body.Bytes(), &created)
	if w := request("GET", fmt.Sprintf("/workloads/%d", created.ID), "operator", ""); w.Code != 404 {
		t.Fatal("forged ownership")
	}
	// Queued templates retain the authenticated owner's identity.
	w = request("POST", "/workloads/run", tokenB, `{"template":"dependency_error"}`)
	if w.Code != 202 {
		t.Fatal(w.Body)
	}
	var run struct {
		ID int `json:"workload_id"`
	}
	json.Unmarshal(w.Body.Bytes(), &run)
	if w := request("GET", fmt.Sprintf("/workloads/%d", run.ID), tokenB, ""); w.Code != 200 {
		t.Fatal("managed ownership lost")
	}
	if w := request("GET", fmt.Sprintf("/workloads/%d", run.ID), tokenA, ""); w.Code != 404 {
		t.Fatal("managed workload leaked")
	}
	if w := request("DELETE", "/workloads/clear", tokenB, ""); w.Code != 200 {
		t.Fatal(w.Body)
	}
	if _, err := d.GetWorkload(fmt.Sprint(alice), keyA.OwnerID); err != nil {
		t.Fatal("clear deleted another owner")
	}
	if _, err := d.GetWorkload(fmt.Sprint(legacy), "legacy"); err != nil {
		t.Fatal("clear deleted legacy")
	}
	// Revocation applies immediately even to existing clients; rotation keeps ownership.
	rotated, rotatedToken, err := d.IssueAPIKey("Alice replacement", keyA.OwnerID)
	if err != nil {
		t.Fatal(err)
	}
	if rotated.OwnerID != keyA.OwnerID {
		t.Fatal("rotation changed ownership")
	}
	if w := request("DELETE", "/api-keys/"+keyA.ID, "operator", ""); w.Code != 204 {
		t.Fatal(w.Body)
	}
	if w := request("GET", "/session", tokenA, ""); w.Code != 401 {
		t.Fatal("revoked key accepted")
	}
	if w := request("GET", fmt.Sprintf("/workloads/%d", alice), rotatedToken, ""); w.Code != 200 {
		t.Fatal("rotation lost workload access")
	}
	w = request("POST", "/api-keys", "operator", `{"name":"new user"}`)
	if w.Code != 201 || !strings.Contains(w.Body.String(), `"api_key":"cl_`) {
		t.Fatal(w.Body)
	}
	if w := request("GET", "/api-keys", "operator", ""); strings.Contains(w.Body.String(), tokenA) || strings.Contains(w.Body.String(), "key_hash") {
		t.Fatal("keys leaked secrets")
	}
	var stored string
	if err := d.QueryRow("SELECT key_hash FROM api_keys WHERE id=?", keyA.ID).Scan(&stored); err != nil || stored == tokenA || stored != db.KeyHash(tokenA) {
		t.Fatal("key not hashed")
	}
}
