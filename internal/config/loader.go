package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bilgehannal/reghost/internal/utils"
	"github.com/bilgehannal/reghost/pkg/reghost"
	"gopkg.in/yaml.v3"
)

const (
	// DefaultConfigPath is the default location for the config file
	DefaultConfigPath = "/etc/reghost.yml"
)

// defaultConfig returns the default configuration
func defaultConfig() *reghost.Config {
	return &reghost.Config{
		ActiveRecord: "default",
		Records: map[string][]reghost.Record{
			"default": {
				{
					Domain: "reghost.local",
					IP:     "127.0.0.1",
				},
			},
		},
	}
}

// Load reads and parses the configuration file
func Load(path string) (*reghost.Config, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Create default config
		if err := createDefaultConfig(path); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return defaultConfig(), nil
	}

	// Read config file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config reghost.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// createDefaultConfig creates a default configuration file
func createDefaultConfig(path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal default config to YAML
	config := defaultConfig()
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal default config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Reload reloads the configuration from disk
func Reload(path string) (*reghost.Config, error) {
	return Load(path)
}

// LogConfigInfo logs detailed information about the configuration
func LogConfigInfo(cfg *reghost.Config, logger *utils.Logger) {
	if cfg == nil || logger == nil {
		return
	}

	logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	logger.Info("📋 Configuration Summary")
	logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	logger.Info("🔹 Active Record: %s", cfg.ActiveRecord)
	logger.Info("")

	// Get active records
	activeRecords := cfg.GetActiveRecords()
	if len(activeRecords) == 0 {
		logger.Warn("⚠️  No active records found!")
		return
	}

	logger.Info("🔹 Active Rules: %d rule(s)", len(activeRecords))
	logger.Info("")

	// Log each rule
	for i, record := range activeRecords {
		logger.Info("  Rule #%d:", i+1)
		logger.Info("    Domain: %s", record.Domain)
		logger.Info("    IP:     %s", record.IP)
		if i < len(activeRecords)-1 {
			logger.Info("")
		}
	}

	logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Log all available record sets
	logger.Info("📦 Available Record Sets: %d", len(cfg.Records))
	for name := range cfg.Records {
		if name == cfg.ActiveRecord {
			logger.Info("  • %s (active)", name)
		} else {
			logger.Info("  • %s", name)
		}
	}
	logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
