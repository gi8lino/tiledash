package server

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// RunHTTPServer starts the HTTP server and handles shutdown on context cancellation.
func RunHTTPServer(ctx context.Context, mux http.Handler, addr string, logger *slog.Logger) error {
	// Create an HTTP server with the mux as the handler.
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Start the HTTP server in its own goroutine.
	go func() {
		logger.Info("Starting server", "address", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server error", "err", err)
		}
	}()

	// Use a WaitGroup to wait for shutdown to complete.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Wait until the context is canceled.
		<-ctx.Done()
		logger.Info("Shutting down server")
		// Create a context with timeout for the shutdown.
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("Server shutdown error", "err", err)
		}
	}()

	wg.Wait() // Wait for the shutdown goroutine to complete.

	return nil
}
