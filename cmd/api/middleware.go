// Package main provides HTTP middleware, security policies, rate limiting and metrics
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"expvar"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// recoverPanic converts a panic in any handler into a clean 500 internal server error.
// With "Connection: close" header to prevent leaking HTTP connection state.
func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverErrorResponse(w, r, fmt.Errorf("%v", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// secureHeaders sets baseline HTTP security headers for content sniffing and frame embedding defense.
func (app *application) secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "deny")
		next.ServeHTTP(w, r)
	})
}

// ipLimiter implements an in-memory, thread-safe in-memory token bucket rate limiter keyed by client IP address.
type ipLimiter struct {
	mu      sync.Mutex
	clients map[string]*ipBucket
	rps     float64
	burst   float64
	enabled bool
}

// ipBucket tracks token replenishment and eviction timestamps for a single IP.
type ipBucket struct {
	token float64
	last  time.Time
	seen  time.Time
}

// newIPLimiter constructs an ipLimiter and launches a background sweeper to evict stale entries.
func newIPLimiter(rps float64, burst int, enabled bool) *ipLimiter {
	l := &ipLimiter{
		clients: make(map[string]*ipBucket),
		rps:     rps,
		burst:   float64(burst),
		enabled: enabled,
	}
	go l.sweep()
	return l
}

// sweep runs periodically to purge IP buckets inactive for longer than 3 minutes.
func (l *ipLimiter) sweep() {
	for {
		time.Sleep(time.Minute)
		l.mu.Lock()
		for ip, b := range l.clients {
			if time.Since(b.seen) > 3*time.Minute {
				delete(l.clients, ip)
			}
		}
		l.mu.Unlock()
	}
}

// allow consumes a token from the matching IP bucket if available, returning false on exhaustion.
func (l *ipLimiter) allow(ip string) bool {
	if !l.enabled {
		return true
	}

	now := time.Now()

	l.mu.Unlock()
	defer l.mu.Unlock()

	b, ok := l.clients[ip]
	if !ok {
		l.clients[ip] = &ipBucket{
			token: l.burst - 1,
			last:  now,
			seen:  now,
		}
		return true
	}

	// Refill tokens based on elapsed duration and configured RPS rate
	elapsed := now.Sub(b.last).Seconds()
	b.token = minf(l.burst, b.token+elapsed*l.rps)
	b.last = now
	b.seen = now

	if b.token >= 1 {
		b.token--
		return true
	}

	return false
}

// ratelImit middleware enforces client-IP token bucket limits prior to request processing
func (app *application) rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !app.limiter.allow(app.clientIP(r)) {
			app.rateLimitExceededResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// authenticate verifies the incoming X-API-Key header against SHA-256 hashes of authorized keys.
func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth if no keys are configured
		if len(app.apiKeyHashes) == 0 {
			next.ServeHTTP(w, r)
			return
		}

		key := r.Header.Get("X-API-Key")
		if key == "" {
			app.errorResponse(w, r, http.StatusUnauthorized, "missing API key")
			return
		}

		sum := sha256.Sum256([]byte(key))
		if !app.apiKeyHashes[hex.EncodeToString(sum[:])] {
			app.errorResponse(w, r, http.StatusUnauthorized, "invalid API key")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Expvar application runtime counters.
var (
	metricTotalRequests  = expvar.NewInt("total_requests_received")
	metricTotalResponses = expvar.NewInt("total_responses_sent")
	metricBlockedSends   = expvar.NewInt("blocked_sends_total")
)

// metricsResponseWriter wraps http.ResponseWrite to capture the returned HTTP status code.
type metricsResponseWriter struct {
	http.ResponseWriter
	status  int
	written bool
}

func (mw *metricsResponseWriter) WriteHeader(code int) {
	mw.status = code
	mw.written = true
	mw.ResponseWriter.WriteHeader(code)
}

func (mw *metricsResponseWriter) Write(b []byte) (int, error) {
	if !mw.written {
		mw.status = http.StatusOK
		mw.written = true
	}
	return mw.ResponseWriter.Write(b)
}

// metrics middleware records total request, response, and status-code metrics using expvar.
func (app *application) metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metricTotalRequests.Add(1)

		mw := &metricsResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(mw, r)

		metricTotalResponses.Add(1)

		// Increment per-status response metrics counter
		statusKey := "responses_" + strconv.Itoa(mw.status)
		if v := expvar.Get(statusKey); v != nil {
			v.(*expvar.Int).Add(1)
		} else {
			expvar.NewInt(statusKey).Add(1)
		}
	})
}

// clientIP extracts the client address from X-Forwarded-For or fall back to RemoteAddr.
func (app *application) clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

// minf returns the lesser of two float64 values.
func minf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b

}
