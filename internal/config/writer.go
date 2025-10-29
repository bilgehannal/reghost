package config

import (
	"fmt"
	"os"

	"github.com/bilgehannal/reghost/pkg/reghost"
	"gopkg.in/yaml.v3"
)

// Writer handles safe configuration file updates
type Writer struct {
	configPath string
}

// NewWriter creates a new config writer
func NewWriter(configPath string) *Writer {
	return &Writer{
		configPath: configPath,
	}
}

// Write writes the configuration to disk safely
func (w *Writer) Write(config *reghost.Config) error {
	// Validate before writing
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to temporary file first
	tempPath := w.configPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp config: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, w.configPath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to rename config file: %w", err)
	}

	return nil
}

// SetActiveRecord updates the active record set
func (w *Writer) SetActiveRecord(recordName string) error {
	// Load current config
	config, err := Load(w.configPath)
	if err != nil {
		return err
	}

	// Check if record exists
	if _, exists := config.Records[recordName]; !exists {
		return fmt.Errorf("record '%s' does not exist", recordName)
	}

	// Update active record
	config.ActiveRecord = recordName

	// Write back
	return w.Write(config)
}

// AddRecord adds a new record to a record set
func (w *Writer) AddRecord(recordSetName string, record reghost.Record) error {
	// Load current config
	config, err := Load(w.configPath)
	if err != nil {
		return err
	}

	// Add record to the set
	if _, exists := config.Records[recordSetName]; !exists {
		config.Records[recordSetName] = []reghost.Record{}
	}

	config.Records[recordSetName] = append(config.Records[recordSetName], record)

	// Write back
	return w.Write(config)
}

// RemoveRecord removes a record from a record set
func (w *Writer) RemoveRecord(recordSetName string, index int) error {
	// Load current config
	config, err := Load(w.configPath)
	if err != nil {
		return err
	}

	// Check if record set exists
	records, exists := config.Records[recordSetName]
	if !exists {
		return fmt.Errorf("record set '%s' does not exist", recordSetName)
	}

	// Check if index is valid
	if index < 0 || index >= len(records) {
		return fmt.Errorf("invalid index %d for record set '%s'", index, recordSetName)
	}

	// Remove record
	config.Records[recordSetName] = append(records[:index], records[index+1:]...)

	// Write back
	return w.Write(config)
}

// CreateRecordSet creates a new record set
func (w *Writer) CreateRecordSet(name string) error {
	// Load current config
	config, err := Load(w.configPath)
	if err != nil {
		return err
	}

	// Check if already exists
	if _, exists := config.Records[name]; exists {
		return fmt.Errorf("record set '%s' already exists", name)
	}

	// Create empty record set
	config.Records[name] = []reghost.Record{}

	// Write back
	return w.Write(config)
}

// DeleteRecordSet deletes a record set
func (w *Writer) DeleteRecordSet(name string) error {
	// Load current config
	config, err := Load(w.configPath)
	if err != nil {
		return err
	}

	// Check if it's the active record
	if config.ActiveRecord == name {
		return fmt.Errorf("cannot delete active record set '%s'", name)
	}

	// Delete record set
	delete(config.Records, name)

	// Write back
	return w.Write(config)
}
