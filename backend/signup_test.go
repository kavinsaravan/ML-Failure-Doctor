package main

import (
	"crashlens/api"
	"crashlens/db"
	"crashlens/middleware"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublicWorkspaceCreation(t *testing.T) {
	t.Setenv("CRASHLENS_API_KEY", "operator")
	d, err := db.New(filepath.Join(t.TempDir(), "signup.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	h := buildRouter(&api.Server{DB: d}, middleware.NewIPRateLimiter(60), "private")
	request := func(method, path, body, ip, token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.RemoteAddr = ip + ":1234"
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	type workspace struct {
		Key   string `json:"api_key"`
		Owner string `json:"owner_id"`
	}
	var users []workspace
	for i := 0; i < 2; i++ {
		w := request("POST", "/workspaces", `{"name":"Same display name"}`, fmt.Sprintf("192.0.2.%d", i+1), "")
		if w.Code != 201 || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal(w.Code, w.Body)
		}
		var user workspace
		if err := json.Unmarshal(w.Body.Bytes(), &user); err != nil {
			t.Fatal(err)
		}
		users = append(users, user)
	}
	if users[0].Key == users[1].Key || users[0].Owner == users[1].Owner {
		t.Fatal("shared identity")
	}
	for _, user := range users {
		owner, err := d.ResolveAPIKey(user.Key)
		if err != nil || owner != user.Owner {
			t.Fatal("invalid key")
		}
		w := request("GET", "/session", "", "192.0.2.1", user.Key)
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"admin":false`) {
			t.Fatal("public signup granted admin")
		}
		w = request("POST", "/workloads", `{"name":"private","status":"running"}`, "192.0.2.1", user.Key)
		if w.Code != 201 {
			t.Fatal(w.Body)
		}
	}
	for _, user := range users {
		w := request("GET", "/workloads", "", "192.0.2.1", user.Key)
		var list []db.Workload
		json.Unmarshal(w.Body.Bytes(), &list)
		if w.Code != 200 || len(list) != 1 {
			t.Fatal("public keys not isolated", w.Body)
		}
		if w := request("GET", "/api-keys", "", "192.0.2.1", user.Key); w.Code != 403 {
			t.Fatal("public key can list keys")
		}
	}
	if w := request("POST", "/workspaces", `{"name":"again"}`, "192.0.2.1", ""); w.Code != 429 || w.Header().Get("Retry-After") != "20" {
		t.Fatal("per-IP creation not limited", w.Code, w.Header())
	}
	if w := request("POST", "/workspaces", `{"name":"intruder","owner_id":"legacy"}`, "192.0.2.3", ""); w.Code != 400 {
		t.Fatal("owner injection accepted")
	}
	if w := request("POST", "/workspaces", `{"name":" "}`, "192.0.2.4", ""); w.Code != 400 {
		t.Fatal("empty name accepted")
	}
	// Five attempts exhaust the global burst even across different IPs.
	if w := request("POST", "/workspaces", `{"name":"next"}`, "192.0.2.5", ""); w.Code != 429 {
		t.Fatal("global creation not limited", w.Code)
	}
	if w := request("GET", "/workspaces", "", "192.0.2.1", ""); w.Code == 200 {
		t.Fatal("public secret listing")
	}
	t.Setenv("ALLOW_WORKSPACE_CREATION", "false")
	disabled := buildRouter(&api.Server{DB: d}, middleware.NewIPRateLimiter(60), "private")
	w := httptest.NewRecorder()
	disabled.ServeHTTP(w, httptest.NewRequest("POST", "/workspaces", strings.NewReader(`{"name":"disabled"}`)))
	if w.Code != 403 {
		t.Fatal("creation not disabled")
	}
	w = httptest.NewRecorder()
	disabled.ServeHTTP(w, httptest.NewRequest("GET", "/health", nil))
	if !strings.Contains(w.Body.String(), `"workspace_creation_enabled":false`) {
		t.Fatal("disabled feature not advertised")
	}
}
