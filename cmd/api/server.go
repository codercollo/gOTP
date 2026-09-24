// Package main manages HTTP server lifecycle and graceful shutdown handling.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// serve configures, starts and gracefully shuts down the HTTP server.
func (app *application) serve() error {
	// configure HTTP server with safety timeouts and structured logger.
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", app.config.port),
		Handler:           app.routes(),
		IdleTimeout:       time.Minute,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		ErrorLog:          log.New(app.logger, "", 0),
	}

	// Channel to capture errors during graceful shutdown.
	shutdownError := make(chan error)

	// Listen for interrupt signals in background to trigger graceful shutdown
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		app.logger.PrintInfo("shutting down server", map[string]string{
			"signal": s.String(),
		})

		// Allow 20 second timeout window for draining requests.
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		// Stop accepting connections and drain active HTTP requests
		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
			return
		}

		// Wait for active background workers to complete.
		app.logger.PrintInfo("completing background tasks", map[string]string{
			"addr": srv.Addr,
		})
		app.wg.Wait()
		shutdownError <- nil
	}()

	app.logger.PrintInfo("starting server", map[string]string{
		"addr": srv.Addr,
		"env":  app.config.env,
	})

	// Start listening for incoming connections
	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// Wait for background shutdown routine to conclude
	if err := <-shutdownError; err != nil {
		return err
	}

	app.logger.PrintInfo("stopped server", map[string]string{
		"addr": srv.Addr,
	})

	return nil
}
