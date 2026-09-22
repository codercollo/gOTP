// Package main provides the entry point, dependency container and runtime setup for the GOTP API.
package main

import (
	"flag"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/codercollo/gOTP/internal/data"
	"github.com/codercollo/gOTP/internal/jsonlog"
	"github.com/codercollo/gOTP/internal/otp"
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
}

// application is the shared dependency container for handlers and middleware.
type application struct {
	config config
	logger *jsonlog.Logger
	models data.Models    // data access layer
	wg     sync.WaitGroup // tracks background tasks for graceful shutdown
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

	flag.Parse()

	// initialize logger
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	// build application container
	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(),
	}

	// start HTTP server
	if err := app.serve(); err != nil {
		logger.PrintFatal(err, nil)
	}
}

// envStr return the environment variable for key .
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
