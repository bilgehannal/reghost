package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bilgehannal/reghost/internal/config"
	"github.com/bilgehannal/reghost/internal/dns"
	"github.com/bilgehannal/reghost/internal/utils"
	"github.com/bilgehannal/reghost/internal/watcher"
)

const (
	configPath = "/etc/reghost.yml"
	logPath    = "/var/log/reghost.log"
)

func main() {
	// Check if running as root
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "Error: reghostd must be run as root")
		fmt.Fprintln(os.Stderr, "Please run with sudo or as root user")
		os.Exit(1)
	}

	// Initialize logger
	logger, err := utils.NewLogger(logPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	logger.Info("=== Starting reghostd ===")

	// Load initial configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		logger.Error("Failed to load config: %v", err)
		os.Exit(1)
	}

	logger.Info("Loaded config from: %s", configPath)

	// Log configuration details
	config.LogConfigInfo(cfg, logger)

	// Get active records
	activeRecords := cfg.GetActiveRecords()
	if len(activeRecords) == 0 {
		logger.Error("No active records found")
		os.Exit(1)
	}

	// Create DNS cache
	cache := dns.NewCache(activeRecords)

	// Create DNS server
	server := dns.NewServer(cache, logger)

	// Start DNS server
	if err := server.Start(); err != nil {
		logger.Error("Failed to start DNS server: %v", err)
		os.Exit(1)
	}

	logger.Info("DNS server started successfully on %s:53", server.GetBindIP())

	// Create config watcher
	w, err := watcher.NewWatcher(configPath, logger, func(newCfg *config.Config) error {
		logger.Info("Reloading configuration...")

		// Log new configuration details
		config.LogConfigInfo(newCfg, logger)

		// Update cache with new active records
		newRecords := newCfg.GetActiveRecords()
		cache.Update(newRecords)

		// Update resolver files based on new active records
		if err := server.UpdateResolverFiles(newRecords); err != nil {
			logger.Warn("Failed to update resolver files: %v", err)
		}

		logger.Info("✓ Configuration reloaded successfully")
		return nil
	})
	if err != nil {
		logger.Error("Failed to create watcher: %v", err)
		os.Exit(1)
	}
	defer w.Close()

	// Start watching
	w.Start()
	logger.Info("Started watching config file for changes")

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Wait for termination signal
	sig := <-sigChan
	logger.Info("Received signal: %v", sig)
	logger.Info("Shutting down gracefully...")

	// Shutdown server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Error during shutdown: %v", err)
	}

	logger.Info("=== reghostd stopped ===")
}
