package main

import (
	"crashlens/api"
	"crashlens/db"
	"crashlens/middleware"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestWorkspaceKeysRequiredInEveryEnvironment(t *testing.T) {
	for _, operator := range []string{"", "test-key"} {
		t.Run("operator="+operator, func(t *testing.T) {
			t.Setenv("CRASHLENS_API_KEY", operator)
			t.Setenv("APP_ENV", "development")
			// A leftover deployment setting must not restore anonymous access.
			t.Setenv("ACCESS_MODE", "demo")
			d, err := db.New(filepath.Join(t.TempDir(), "test.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer d.Close()
			handler := buildRouter(&api.Server{DB: d}, middleware.NewIPRateLimiter(60))
			for _, route := range [][2]string{
				{"GET", "/session"}, {"GET", "/workloads"}, {"GET", "/workloads/999"},
				{"GET", "/workloads/999/logs"}, {"GET", "/workloads/999/metrics"}, {"GET", "/summary"},
				{"POST", "/workloads"}, {"POST", "/workloads/999/diagnose"}, {"DELETE", "/workloads/clear"},
				{"PUT", "/workloads/999"}, {"DELETE", "/workloads/999"}, {"GET", "/api-keys"},
			} {
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, httptest.NewRequest(route[0], route[1], nil))
				if w.Code != 401 {
					t.Fatalf("anonymous request accepted: %v %d", route, w.Code)
				}
			}
			key, token, err := d.IssueAPIKey("Workspace", "")
			if err != nil {
				t.Fatal(err)
			}
			for _, path := range []string{"/session", "/workloads", "/summary"} {
				request := httptest.NewRequest("GET", path, nil)
				request.Header.Set("Authorization", "Bearer "+token)
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, request)
				if w.Code != 200 {
					t.Fatalf("workspace request failed: %s %s", path, w.Body)
				}
			}
			if _, err := d.RevokeAPIKey(key.ID); err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest("GET", "/session", nil)
			request.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, request)
			if w.Code != 401 {
				t.Fatal("revoked key accepted")
			}
			if operator != "" {
				request = httptest.NewRequest("GET", "/session", nil)
				request.Header.Set("Authorization", "Bearer "+operator)
				w = httptest.NewRecorder()
				handler.ServeHTTP(w, request)
				if w.Code != 200 {
					t.Fatal("operator key rejected")
				}
			}
			w = httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest("GET", "/health", nil))
			var health struct {
				Required bool `json:"auth_required"`
			}
			if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &health) != nil || !health.Required {
				t.Fatalf("health must remain public and advertise required authentication: %s", w.Body)
			}
		})
	}
}
