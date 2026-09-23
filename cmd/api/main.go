// Package main provides the entry point, dependency container and runtime setup for the GOTP API.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/codercollo/gOTP/internal/data"
	"github.com/codercollo/gOTP/internal/guard"
	"github.com/codercollo/gOTP/internal/insights"
	"github.com/codercollo/gOTP/internal/jsonlog"
	"github.com/codercollo/gOTP/internal/otp"
	"github.com/codercollo/gOTP/internal/sms"
)

// version is the app release version
const version = "1.0.0"

// config holds runtime settings loaded from flags or env variables.
type config struct {
	port int
	env  string
	otp  struct {
		length      int           // length of generated OTP codes
		ttl         time.Duration // time-to-live before expiration
		maxAttempts int           // max invalid attempts before lockout
	}
	sms struct {
		username string // Africa's Talking API username
		apiKey   string // Africa's Talking API key
		senderID string // Alphanumeric sender ID
		sandbox  bool   // Africa's Talking sandbox endpoint
	}
	insights struct {
		enabled bool // Enable SIM swap check before dispatch
	}
	limiter struct {
		rps     float64 // Allowed requests per second
		burst   int     // Maximum burst capacity
		enabled bool    // Toggle IP rate limiting
	}
	guard struct {
		velocityCapacity float64       // Max velocity burst capacity
		velocityRefill   float64       // Velocity token refill rate
		ratioPrefixLen   int           // Phone prefix length for tracking
		ratioMinSends    int           // Minimum sends before checking ratio
		ratioMax         float64       // Max allowed send-to-verify ratio
		fanOutWindow     time.Duration // Window for tracking distinct destinations
		fanOutFloor      int           // Minimum distinct destinations before blocking
		fanOutFactor     float64       // EWMA multiplier for adaptive threshold
		fanOutAlpha      float64       // EWMA smoothing factor
	}
	apiKeys string
}

// application is the shared dependency container for handlers and middleware.
type application struct {
	config       config
	logger       *jsonlog.Logger
	models       data.Models
	guard        *guard.Guard
	sms          sms.Sender
	swap         insights.Checker
	limiter      *ipLimiter
	apiKeyHashes map[string]bool
	wg           sync.WaitGroup // tracks background tasks for graceful shutdown

}

func main() {
	var cfg config

	// server flags
	flag.IntVar(&cfg.port, "port", envInt("PORT", 4000), "API server port")
	flag.StringVar(&cfg.env, "env", envStr("ENV", "development"), "Environment (development|staging|production)")

	// otp flags
	flag.IntVar(&cfg.otp.length, "otp-length", envInt("OTP_LENGTH", otp.DefaultLength), "OTP code length")
	flag.DurationVar(&cfg.otp.ttl, "otp-ttl", envDur("OTP_TTL", otp.DefaultTTL), "OTP time-to-live")
	flag.IntVar(&cfg.otp.maxAttempts, "otp-max-attempts", envInt("OTP_MAX_ATTEMPTS", otp.DefaultMaxAttempts), "Max verify attempts before lockout")

	// SMS provider settings
	flag.StringVar(&cfg.sms.username, "at-username", envStr("AT_USERNAME", ""), "Africa's Talking username")
	flag.StringVar(&cfg.sms.apiKey, "at-api-key", envStr("AT_API_KEY", ""), "Africa's Talking API key")
	flag.StringVar(&cfg.sms.senderID, "at-sender-id", envStr("AT_SENDER_ID", ""), "Africa's Talking sender ID")
	flag.BoolVar(&cfg.sms.sandbox, "at-sandbox", envBool("AT_SANDBOX", true), "Use the AT sandbox host")

	// SIM swap settings
	flag.BoolVar(&cfg.insights.enabled, "sim-swap-check", envBool("SIM_SWAP_CHECK", false), "Enable the SIM-swap check before sending")

	// IP rate limiter settings
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", envFloat("LIMITER_RPS", 2), "Per-IP requests/sec")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", envInt("LIMITER_BURST", 5), "Per-IP burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", envBool("LIMITER_ENABLED", true), "Enable per-IP rate limiting")

	// Fraud guard settings
	flag.Float64Var(&cfg.guard.velocityCapacity, "guard-velocity-capacity", envFloat("GUARD_VELOCITY_CAPACITY", 5), "Velocity burst allowance per key")
	flag.Float64Var(&cfg.guard.velocityRefill, "guard-velocity-refill", envFloat("GUARD_VELOCITY_REFILL", 0.2), "Velocity refill tokens/sec")
	flag.IntVar(&cfg.guard.ratioPrefixLen, "guard-ratio-prefix-len", envInt("GUARD_RATIO_PREFIX_LEN", 6), "Digits defining a number prefix")
	flag.IntVar(&cfg.guard.ratioMinSends, "guard-ratio-min-sends", envInt("GUARD_RATIO_MIN_SENDS", 20), "Sends before the ratio rule engages")
	flag.Float64Var(&cfg.guard.ratioMax, "guard-ratio-max", envFloat("GUARD_RATIO_MAX", 10), "Max sends/verifies before throttling")

	// Fan-out detection settings.
	flag.DurationVar(&cfg.guard.fanOutWindow, "guard-fanout-window", envDur("GUARD_FANOUT_WINDOW", time.Minute), "Fan-out detection window")
	flag.IntVar(&cfg.guard.fanOutFloor, "guard-fanout-floor", envInt("GUARD_FANOUT_FLOOR", 5), "Minimum distinct destinations before blocking")
	flag.Float64Var(&cfg.guard.fanOutFactor, "guard-fanout-factor", envFloat("GUARD_FANOUT_FACTOR", 3), "Fan-out EWMA threshold multiplier")
	flag.Float64Var(&cfg.guard.fanOutAlpha, "guard-fanout-alpha", envFloat("GUARD_FANOUT_ALPHA", 0.3), "Fan-out EWMA smoothing factor")

	// Auth settings
	flag.StringVar(&cfg.apiKeys, "api-keys", envStr("API_KEYS", ""), "Comma-separated tenant API keys (empty disables auth)")

	flag.Parse()

	// Initialize logger
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	// Build application dependency container.
	app := &application{
		config:       cfg,
		logger:       logger,
		models:       data.NewModels(),
		guard:        guard.New(guardConfig(cfg)),
		limiter:      newIPLimiter(cfg.limiter.rps, cfg.limiter.burst, cfg.limiter.enabled),
		apiKeyHashes: hashKeys(cfg.apiKeys),
	}

	// Initialize SMS sender
	if cfg.sms.apiKey != "" && cfg.sms.username != "" {
		app.sms = sms.New(cfg.sms.username, cfg.sms.apiKey, cfg.sms.senderID, cfg.sms.sandbox)
	} else {
		app.sms = sms.Discard
	}

	// Initialize SIM swap checker
	if cfg.insights.enabled && cfg.sms.apiKey != "" && cfg.sms.username != "" {
		app.swap = insights.New(cfg.sms.username, cfg.sms.apiKey, cfg.sms.sandbox)
	} else {
		app.swap = insights.NoopChecker{}
	}

	// start HTTP server
	if err := app.serve(); err != nil {
		logger.PrintFatal(err, nil)
	}
}

// guardConfig converts application configs into a guard Config.
func guardConfig(cfg config) guard.Config {
	return guard.Config{
		VelocityCapacity: cfg.guard.velocityCapacity,
		VelocityRefill:   cfg.guard.velocityRefill,
		RatioPrefixLen:   cfg.guard.ratioPrefixLen,
		RatioMinSends:    cfg.guard.ratioMinSends,
		RatioMax:         cfg.guard.ratioMax,
		FanOutWindow:     cfg.guard.fanOutWindow,
		FanOutFloor:      cfg.guard.fanOutFloor,
		FanOutFactor:     cfg.guard.fanOutFactor,
		FanOutAlpha:      cfg.guard.fanOutAlpha,
	}
}

// hashKeys parses a CSV API keys into a SHA-256 lookup set.
func hashKeys(csv string) map[string]bool {
	out := make(map[string]bool)
	for _, k := range strings.Split(csv, ",") {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		sum := sha256.Sum256([]byte(k))
		out[hex.EncodeToString(sum[:])] = true
	}
	return out
}

// envStr returns the environment variable for key or the fallback value.
func envStr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}

	return fallback
}

// envInt return the environment variable for key parsed as an int.
func envInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

// envDur returns the environment variable for key parsed as a duration.
func envDur(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

// envFloat returns the environment variable for key parsed as a float64 or the fallback value.
func envFloat(key string, fallback float64) float64 {
	if v, ok := os.LookupEnv(key); ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

// envBool returns the environment variable for key parsed as a boolean or the fallback value.
func envBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
