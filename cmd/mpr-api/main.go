package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cbschuld/mpr-6zhmaut-golang-api/internal/amp"
	"github.com/cbschuld/mpr-6zhmaut-golang-api/internal/api"
	"github.com/cbschuld/mpr-6zhmaut-golang-api/internal/config"
	"github.com/cbschuld/mpr-6zhmaut-golang-api/internal/serial"
)

//go:embed all:dist
var webEmbed embed.FS

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	// Set up structured logging
	var level slog.Level
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	logger.Info("starting mpr-6zhmaut-api",
		"device", cfg.Device,
		"target_baud_rate", cfg.TargetBaudRate,
		"port", cfg.Port,
		"amp_count", cfg.AmpCount,
		"poll_interval", cfg.PollInterval.String(),
		"health_interval", cfg.HealthInterval.String(),
	)

	// Create components
	events := amp.NewEventLog(100)
	port := serial.NewPort(cfg.Device, logger)
	queue := serial.NewQueue(port, logger)
	state := amp.NewStateMachine(logger, events)
	cache := amp.NewZoneCache()
	controller := amp.NewController(cfg, port, queue, state, cache, events, logger)

	// Set up context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start the command queue
	queue.Start(ctx)

	// Connect to the amp (probe + baud step-up)
	if err := controller.Start(ctx); err != nil {
		logger.Error("failed to connect to amp", "error", err)
		os.Exit(1)
	}

	// Start background poller and health monitor
	poller := amp.NewPoller(controller, queue, state, cache, cfg.PollInterval, cfg.CmdTimeout, cfg.AmpCount, logger)
	poller.Start(ctx)

	healthMonitor := amp.NewHealthMonitor(controller, queue, state, cfg.HealthInterval, cfg.CmdTimeout, logger)
	healthMonitor.Start(ctx)

	// Prepare embedded web UI filesystem (strip "web" prefix so files are at root)
	webFS, err := fs.Sub(webEmbed, "dist")
	if err != nil {
		logger.Error("failed to load embedded web UI", "error", err)
		os.Exit(1)
	}

	// Create and start HTTP server
	server := api.NewServer(cfg, controller, cache, state, queue, events, healthMonitor, port, logger, webFS)
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      server.Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("http server listening", "port", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server error", "error", err)
			cancel()
		}
	}()

	// Wait for shutdown signal
	select {
	case sig := <-sigCh:
		logger.Info("received signal, shutting down", "signal", sig)
	case <-ctx.Done():
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	httpServer.Shutdown(shutdownCtx)
	queue.Stop()
	port.Close()
	logger.Info("shutdown complete")
}
