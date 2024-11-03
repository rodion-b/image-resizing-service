package main

import (
	"context"
	"fmt"
	"images-resizing-service/config"
	"images-resizing-service/handlers"
	"images-resizing-service/services"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"golang.org/x/sync/errgroup"
)

func main() {
	// Setting up context with SIGTERM and SIGINT signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	// Creating error group with shared context
	errGroup, ctx := errgroup.WithContext(ctx)

	// Initialize LRU cache
	cache, err := lru.New(1024)
	if err != nil {
		log.Panicf("Failed to create cache: %v", err)
	}

	// Initialize resizing service with cache
	svc := &services.ResizingService{Cache: cache}

	// Setup HTTP handlers
	mux := http.NewServeMux()
	mux.Handle("/v1/resize", handlers.ResizeHandler(svc))
	mux.Handle("/v1/image/", handlers.GetImageHandler(svc))

	server := &http.Server{
		Addr:    config.Hostport,
		Handler: mux,
	}
	log.Printf("Starting server on %s", config.Hostport)

	// Goroutine to start the server
	errGroup.Go(func() error {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
		return nil
	})

	// Goroutine to handle shutdown on context cancellation
	errGroup.Go(func() error {
		<-ctx.Done()
		log.Print("Shutting down server...")

		// Context with timeout for graceful shutdown
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown failed: %w", err)
		}
		log.Print("Server shut down gracefully")
		return nil
	})

	// Wait for all goroutines to finish, logging any errors
	if err := errGroup.Wait(); err != nil {
		slog.Error(
			"Error occurred",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
}
