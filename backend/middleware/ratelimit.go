package middleware

import (
	"context"
	"fmt"
	"golang.org/x/time/rate"
	"math"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"
)

type ipEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}
type IPRateLimiter struct {
	limiters map[string]*ipEntry
	mu       sync.Mutex
	rate     rate.Limit
	burst    int
	trusted  []netip.Prefix
}

func NewIPRateLimiter(requestsPerMinute int) *IPRateLimiter {
	if requestsPerMinute < 1 {
		requestsPerMinute = 1
	}
	burst := requestsPerMinute / 6
	if burst < 1 {
		burst = 1
	}
	return &IPRateLimiter{limiters: make(map[string]*ipEntry), rate: rate.Limit(float64(requestsPerMinute) / 60), burst: burst}
}

// Configure before serving. Forwarded headers are ignored unless the direct peer is trusted.
func (rl *IPRateLimiter) SetTrustedProxies(cidrs []string) error {
	var trusted []netip.Prefix
	for _, cidr := range cidrs {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(cidr)
		if err != nil {
			return fmt.Errorf("invalid trusted proxy %q", cidr)
		}
		trusted = append(trusted, prefix)
	}
	rl.trusted = trusted
	return nil
}
func (rl *IPRateLimiter) isTrusted(ip netip.Addr) bool {
	for _, prefix := range rl.trusted {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}
func (rl *IPRateLimiter) clientIP(r *http.Request) string {
	host := r.RemoteAddr
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	peer, err := netip.ParseAddr(host)
	if err != nil {
		return host
	}
	peer = peer.Unmap()
	if !rl.isTrusted(peer) {
		return peer.String()
	}
	hops := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	candidate := peer
	for i := len(hops) - 1; i >= 0; i-- {
		if !rl.isTrusted(candidate) {
			break
		}
		ip, err := netip.ParseAddr(strings.TrimSpace(hops[i]))
		if err != nil {
			return peer.String()
		}
		candidate = ip.Unmap()
	}
	return candidate.String()
}
func (rl *IPRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := rl.clientIP(r)
		now := time.Now()
		rl.mu.Lock()
		entry, exists := rl.limiters[ip]
		if !exists {
			// Bound tracking memory without evicting active clients and resetting their limits.
			if len(rl.limiters) >= 10000 {
				rl.mu.Unlock()
				http.Error(w, "Rate limiter capacity reached", http.StatusTooManyRequests)
				return
			}
			entry = &ipEntry{limiter: rate.NewLimiter(rl.rate, rl.burst)}
			rl.limiters[ip] = entry
		}
		entry.lastSeen = now
		allowed := entry.limiter.Allow()
		rl.mu.Unlock()
		if !allowed {
			w.Header().Set("Retry-After", fmt.Sprint(int(math.Ceil(1/float64(rl.rate)))))
			http.Error(w, "Rate limit exceeded. Try again later.", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (rl *IPRateLimiter) Cleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				rl.mu.Lock()
				for ip, entry := range rl.limiters {
					if now.Sub(entry.lastSeen) > 15*time.Minute {
						delete(rl.limiters, ip)
					}
				}
				rl.mu.Unlock()
			}
		}
	}()
}
