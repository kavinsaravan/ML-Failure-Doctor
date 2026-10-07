package middleware

import (
	"fmt"
	"golang.org/x/time/rate"
	"math"
	"net/http"
)

// Shared process-wide budget for public creation, independent of IP identity.
func GlobalRateLimit(requestsPerMinute, burst int) func(http.Handler) http.Handler {
	limiter := rate.NewLimiter(rate.Limit(float64(requestsPerMinute)/60), burst)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				w.Header().Set("Retry-After", fmt.Sprint(int(math.Ceil(60/float64(requestsPerMinute)))))
				http.Error(w, "Workspace creation is busy. Try again later.", 429)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
