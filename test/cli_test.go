package test

import (
	"path/filepath"
	"testing"

	"github.com/bilgehannal/reghost/internal/config"
	"github.com/bilgehannal/reghost/pkg/reghost"
)

func TestWriter(t *testing.T) {
	// Create temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test.yml")

	// Create initial config
	initialConfig := &reghost.Config{
		ActiveRecord: "set1",
		Records: map[string][]reghost.Record{
			"set1": {
				{Domain: "test.local", IP: "127.0.0.1"},
			},
			"set2": {
				{Domain: "example.local", IP: "10.0.0.1"},
			},
		},
	}

	writer := config.NewWriter(configPath)
	if err := writer.Write(initialConfig); err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	// Test SetActiveRecord
	t.Run("SetActiveRecord", func(t *testing.T) {
		if err := writer.SetActiveRecord("set2"); err != nil {
			t.Fatalf("SetActiveRecord failed: %v", err)
		}

		cfg, err := config.Load(configPath)
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if cfg.ActiveRecord != "set2" {
			t.Errorf("Expected activeRecord 'set2', got '%s'", cfg.ActiveRecord)
		}
	})

	// Test AddRecord
	t.Run("AddRecord", func(t *testing.T) {
		newRecord := reghost.Record{
			Domain: "new.local",
			IP:     "192.168.1.1",
		}

		if err := writer.AddRecord("set1", newRecord); err != nil {
			t.Fatalf("AddRecord failed: %v", err)
		}

		cfg, err := config.Load(configPath)
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		set1 := cfg.Records["set1"]
		if len(set1) != 2 {
			t.Errorf("Expected 2 records in set1, got %d", len(set1))
		}

		found := false
		for _, r := range set1 {
			if r.Domain == "new.local" && r.IP == "192.168.1.1" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Added record not found in config")
		}
	})

	// Test RemoveRecord
	t.Run("RemoveRecord", func(t *testing.T) {
		if err := writer.RemoveRecord("set1", 0); err != nil {
			t.Fatalf("RemoveRecord failed: %v", err)
		}

		cfg, err := config.Load(configPath)
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		set1 := cfg.Records["set1"]
		if len(set1) != 1 {
			t.Errorf("Expected 1 record in set1 after removal, got %d", len(set1))
		}
	})

	// Test CreateRecordSet
	t.Run("CreateRecordSet", func(t *testing.T) {
		// Note: We can't create an empty set because validation requires at least one record
		// So we use AddRecord to a new set instead
		newRecord := reghost.Record{
			Domain: "set3.local",
			IP:     "192.168.1.3",
		}

		if err := writer.AddRecord("set3", newRecord); err != nil {
			t.Fatalf("AddRecord to new set failed: %v", err)
		}

		cfg, err := config.Load(configPath)
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if _, exists := cfg.Records["set3"]; !exists {
			t.Error("Created record set not found")
		}
	})

	// Test DeleteRecordSet
	t.Run("DeleteRecordSet", func(t *testing.T) {
		// First create a non-active record set
		writer.CreateRecordSet("to_delete")

		if err := writer.DeleteRecordSet("to_delete"); err != nil {
			t.Fatalf("DeleteRecordSet failed: %v", err)
		}

		cfg, err := config.Load(configPath)
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if _, exists := cfg.Records["to_delete"]; exists {
			t.Error("Deleted record set still exists")
		}
	})

	// Test deleting active record set (should fail)
	t.Run("DeleteActiveRecordSet", func(t *testing.T) {
		cfg, _ := config.Load(configPath)
		activeSet := cfg.ActiveRecord

		err := writer.DeleteRecordSet(activeSet)
		if err == nil {
			t.Error("Expected error when deleting active record set, got nil")
		}
	})
}
