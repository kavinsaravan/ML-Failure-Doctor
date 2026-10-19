package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPortsAndSpoofedForwardedHeadersDoNotResetLimit(t *testing.T) {
	limiter := NewIPRateLimiter(10)
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	for i, remote := range []string{"192.0.2.1:1000", "192.0.2.1:2000"} {
		r := httptest.NewRequest("POST", "/", nil)
		r.RemoteAddr = remote
		r.Header.Set("X-Forwarded-For", []string{"203.0.113.1", "203.0.113.2"}[i])
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		expected := 200
		if i == 1 {
			expected = 429
		}
		if w.Code != expected {
			t.Fatalf("port bypass: %d", w.Code)
		}
	}
}
func TestTrustedProxyWalksFromRightToLeft(t *testing.T) {
	limiter := NewIPRateLimiter(10)
	if err := limiter.SetTrustedProxies([]string{"10.0.0.0/8"}); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	r.Header.Set("X-Forwarded-For", "198.51.100.99, 192.0.2.25, 10.0.0.2")
	if limiter.clientIP(r) != "192.0.2.25" {
		t.Fatal(limiter.clientIP(r))
	}
	r.RemoteAddr = "[2001:db8::1]:1234"
	if limiter.clientIP(r) != "2001:db8::1" {
		t.Fatal(limiter.clientIP(r))
	}
	if err := limiter.SetTrustedProxies([]string{"not-a-cidr"}); err == nil {
		t.Fatal("invalid proxy accepted")
	}
}
func TestSmallRateLimitHasNonzeroBurst(t *testing.T) {
	limiter := NewIPRateLimiter(1)
	if limiter.burst != 1 {
		t.Fatal(limiter.burst)
	}
}
