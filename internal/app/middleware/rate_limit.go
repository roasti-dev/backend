package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimitConfig configures the per-IP rate limiter.
// Set Enabled=false to bypass rate limiting (e.g. in debug mode or tests).
type RateLimitConfig struct {
	Enabled bool
	// RPS is the sustained request rate per IP (requests per second).
	RPS rate.Limit
	// Burst is the maximum burst size per IP.
	Burst int
	// CleanupAfter controls how long an idle IP entry is kept in memory.
	CleanupAfter time.Duration
}

// DefaultRateLimitConfig returns a relaxed config suitable for non-critical apps.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Enabled:      true,
		RPS:          10,
		Burst:        30,
		CleanupAfter: 5 * time.Minute,
	}
}

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type rateLimiter struct {
	cfg      RateLimitConfig
	mu       sync.Mutex
	limiters map[string]*ipLimiter
}

func newRateLimiter(cfg RateLimitConfig) *rateLimiter {
	rl := &rateLimiter{
		cfg:      cfg,
		limiters: make(map[string]*ipLimiter),
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) get(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, ok := rl.limiters[ip]
	if !ok {
		entry = &ipLimiter{
			limiter: rate.NewLimiter(rl.cfg.RPS, rl.cfg.Burst),
		}
		rl.limiters[ip] = entry
	}
	entry.lastSeen = time.Now()
	return entry.limiter
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cfg.CleanupAfter)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		for ip, entry := range rl.limiters {
			if time.Since(entry.lastSeen) > rl.cfg.CleanupAfter {
				delete(rl.limiters, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimit returns a per-IP rate limiting middleware.
// If cfg.Enabled is false, the middleware is a no-op.
func RateLimit(cfg RateLimitConfig) func(http.Handler) http.Handler {
	if !cfg.Enabled {
		return func(next http.Handler) http.Handler { return next }
	}
	rl := newRateLimiter(cfg)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}
			if !rl.get(ip).Allow() {
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
