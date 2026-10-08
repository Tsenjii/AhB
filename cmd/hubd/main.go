package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Tsenjii/android-ai-hub/internal/config"
	"github.com/Tsenjii/android-ai-hub/internal/hub"
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
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
		return
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}