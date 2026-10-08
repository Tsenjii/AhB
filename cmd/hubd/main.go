package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Tsenjii/AhB/internal/config"
	"github.com/Tsenjii/AhB/internal/hub"
)

func main() {
	defaultConfig := os.Getenv("AIHUB_CONFIG")
	if defaultConfig == "" {
		defaultConfig = "config.json"
	}
	configPath := flag.String("config", defaultConfig, "path to hub config")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Reserve the Hub port before creating any provider child processes.
	// Otherwise a second AhB instance may exit on bind failure while leaving
	// newly spawned sidecars behind.
	listener, err := net.Listen("tcp", cfg.Listen)
	if err != nil {
		log.Fatalf("listen %s: %v", cfg.Listen, err)
	}
	defer listener.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	h := hub.New(cfg)
	h.Start(ctx)

	server := &http.Server{
		Addr:              cfg.Listen,
		Handler:           h.Handler(),
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("android-ai-hub listening on %s", cfg.Listen)
		errCh <- server.Serve(listener)
	}()

	var serveErr error
	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr = err
			log.Printf("hub HTTP server stopped unexpectedly: %v", err)
		}
	}

	// Shut down the HTTP listener and then wait for all supervised sidecars
	// and health probes to exit before the Hub process disappears.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	shutdownCancel()
	cancel()
	h.Wait()

	if serveErr != nil {
		log.Fatalf("server: %v", serveErr)
	}
}
