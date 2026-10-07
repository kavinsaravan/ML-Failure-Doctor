package main

import (
	"crashlens/api"
	"crashlens/db"
	"crashlens/fireworks"
	"crashlens/middleware"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestUserDiagnosesNeverSpendOperatorCredits(t *testing.T) {
	t.Setenv("CRASHLENS_API_KEY", "operator")
	d, err := db.New(filepath.Join(t.TempDir(), "byok.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	key, token, err := d.IssueAPIKey("Alice", "")
	if err != nil {
		t.Fatal(err)
	}
	id, _ := d.CreateWorkload("failed GPU job", "ML_JOB", "failed", key.OwnerID)
	logs := "RuntimeError: CUDA out of memory"
	_, err = d.Exec("UPDATE workloads SET job_logs=? WHERE id=?", logs, id)
	if err != nil {
		t.Fatal(err)
	}
	legacy, _ := d.CreateWorkload("legacy GPU job", "ML_JOB", "failed")
	d.Exec("UPDATE workloads SET job_logs=? WHERE id=?", logs, legacy)
	calls := 0
	failed := false
	transport := transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		expected := "Bearer user-fireworks-secret"
		if strings.Contains(r.URL.Path, "operator") {
			expected = "Bearer operator-fireworks-secret"
		}
		if r.Header.Get("Authorization") != expected {
			t.Errorf("wrong payer")
		}
		if !strings.Contains(r.URL.Path, "operator") {
			if r.URL.Host != "api.fireworks.ai" {
				t.Error("credential sent to non-Fireworks host")
			}
			var input fireworks.Request
			json.NewDecoder(r.Body).Decode(&input)
			if input.Model != "accounts/fireworks/models/user-model" {
				t.Error("user model not used")
			}
		}
		status := 200
		body := `{"choices":[{"message":{"tool_calls":[{"function":{"name":"diagnose_ml_failure","arguments":"{\"root_cause\":\"Allocation too large\",\"evidence\":[\"CUDA out of memory\"],\"recommended_fixes\":[\"Reduce allocation\"],\"safe_to_retry\":false}"}}]}}]}`
		if failed {
			status = 401
			body = "user-fireworks-secret should never leak"
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	client := &http.Client{Transport: transport}
	server := &api.Server{DB: d, FWClient: &fireworks.Client{APIKey: "operator-fireworks-secret", Model: "operator-model", BaseURL: "https://api.fireworks.ai/operator", Client: client}, FireworksHTTPClient: client}
	h := buildRouter(server, middleware.NewIPRateLimiter(600), "private")
	request := func(id int64, auth, key, model string, refresh bool) *httptest.ResponseRecorder {
		path := fmt.Sprintf("/workloads/%d/diagnose", id)
		if refresh {
			path += "?refresh=true"
		}
		r := httptest.NewRequest("POST", path, nil)
		r.Header.Set("Authorization", "Bearer "+auth)
		r.Header.Set("X-Fireworks-API-Key", key)
		r.Header.Set("X-Fireworks-Model", model)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	w := request(id, token, "", "", true)
	if w.Code != 200 || calls != 0 || !strings.Contains(w.Body.String(), "Add your Fireworks") {
		t.Fatalf("operator fallback used: %d %s", calls, w.Body)
	}
	w = request(id, token, "user-fireworks-secret", "", true)
	if w.Code != 400 || calls != 0 {
		t.Fatal("missing model not rejected")
	}
	w = request(id, token, "user-fireworks-secret", "accounts/fireworks/models/user-model", true)
	if w.Code != 200 || calls != 1 || !strings.Contains(w.Body.String(), `"source":"ai"`) {
		t.Fatalf("user AI failed: %s", w.Body)
	}
	w = request(id, token, "", "", false)
	if w.Code != 200 || calls != 1 {
		t.Fatal("saved report triggered billing")
	}
	failed = true
	w = request(id, token, "user-fireworks-secret", "accounts/fireworks/models/user-model", true)
	if w.Code != 200 || calls != 2 || !strings.Contains(w.Body.String(), `"source":"rules"`) {
		t.Fatal("failed user request used operator fallback")
	}
	if strings.Contains(w.Body.String(), "user-fireworks-secret") {
		t.Fatal("secret in report")
	}
	workload, _ := d.GetWorkload(fmt.Sprint(id))
	if workload.FailureReport == nil || strings.Contains(*workload.FailureReport, "user-fireworks-secret") {
		t.Fatal("secret persisted")
	}
	failed = false
	w = request(legacy, "operator", "", "", true)
	if w.Code != 200 || calls != 3 || !strings.Contains(w.Body.String(), `"source":"ai"`) {
		t.Fatal("operator billing broken")
	}
	w = request(legacy, token, "user-fireworks-secret", "accounts/fireworks/models/user-model", true)
	if w.Code != 404 || calls != 3 {
		t.Fatal("cross-owner diagnosis billed")
	}
}
