package main

import (
	"crashlens/api"
	"crashlens/db"
	"crashlens/middleware"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestPrivateAndDemoAccessPolicy(t *testing.T) {
	t.Setenv("CRASHLENS_API_KEY", "test-key")
	d, err := db.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	for _, mode := range []string{"private", "demo"} {
		handler := buildRouter(&api.Server{DB: d}, middleware.NewIPRateLimiter(60), mode)
		for _, path := range []string{"/workloads", "/workloads/999", "/workloads/999/logs", "/workloads/999/metrics", "/summary"} {
			r := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if mode == "private" && w.Code != 401 {
				t.Fatalf("private %s leaked: %d", path, w.Code)
			}
			if mode == "demo" && w.Code == 401 {
				t.Fatalf("demo reads blocked: %s", path)
			}
		}
		for _, route := range [][2]string{{"POST", "/workloads"}, {"POST", "/workloads/run"}, {"POST", "/workloads/999/diagnose"}, {"DELETE", "/workloads/clear"}, {"PUT", "/workloads/999"}} {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(route[0], route[1], nil))
			if w.Code != 401 {
				t.Fatalf("unprotected mutation: %v %d", route, w.Code)
			}
		}
		w := httptest.NewRecorder()
		request := httptest.NewRequest("GET", "/session", nil)
		request.Header.Set("Authorization", "Bearer test-key")
		handler.ServeHTTP(w, request)
		if w.Code != 200 {
			t.Fatalf("session rejected: %s", w.Body.String())
		}
		request = httptest.NewRequest("GET", "/workloads", nil)
		request.Header.Set("Authorization", "Bearer test-key")
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, request)
		if w.Code != 200 {
			t.Fatal(w.Code)
		}
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "/health", nil))
		if w.Code != 200 {
			t.Fatal("health must stay public")
		}
	}
}
