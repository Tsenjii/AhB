package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
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
	configureTermuxRestart(h)
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

 // configureTermuxRestart exposes a one-click restart only from an AhB
 // Android prebuilt running at its own bin/hubd path. Nothing else is
 // restarted, and the browser cannot supply executable names or arguments.
func configureTermuxRestart(h *hub.Hub) {
	if runtime.GOOS != "android" {
		return
	}
	exe, err := os.Executable()
	if err != nil || filepath.Base(exe) != "hubd" {
		return
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(exe), ".."))
	script := filepath.Join(root, "scripts", "start-termux.sh")
	if fi, err := os.Stat(script); err != nil || fi.IsDir() {
		return
	}
	h.SetRestartHandler(func() error {
		logPath := filepath.Join(root, "logs", "ui-restart.log")
		out, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		defer out.Close()
		cmd := exec.Command("bash", script, "--restart-from-pid", strconv.Itoa(os.Getpid()))
		cmd.Dir = root
		cmd.Stdin = nil
		cmd.Stdout = out
		cmd.Stderr = out
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if err := cmd.Start(); err != nil {
			return err
		}
		// The child continues after hubd has gracefully shut down its sidecars.
		return cmd.Process.Release()
	})
}
