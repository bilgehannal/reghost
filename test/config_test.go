package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bilgehannal/reghost/internal/config"
	"github.com/bilgehannal/reghost/pkg/reghost"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test.yml")

	// Write test config
	testConfig := `activeRecord: test1
records:
  test1:
    - domain: 'test.local'
      ip: 127.0.0.1
  test2:
    - domain: '^[a-z]+\.example\.$'
      ip: 10.0.0.1
`
	if err := os.WriteFile(configPath, []byte(testConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify active record
	if cfg.ActiveRecord != "test1" {
		t.Errorf("Expected activeRecord 'test1', got '%s'", cfg.ActiveRecord)
	}

	// Verify records
	if len(cfg.Records) != 2 {
		t.Errorf("Expected 2 record sets, got %d", len(cfg.Records))
	}

	// Verify test1 records
	test1Records, ok := cfg.Records["test1"]
	if !ok {
		t.Fatal("test1 record set not found")
	}
	if len(test1Records) != 1 {
		t.Errorf("Expected 1 record in test1, got %d", len(test1Records))
	}
	if test1Records[0].Domain != "test.local" {
		t.Errorf("Expected domain 'test.local', got '%s'", test1Records[0].Domain)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *reghost.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &reghost.Config{
				ActiveRecord: "default",
				Records: map[string][]reghost.Record{
					"default": {
						{Domain: "test.local", IP: "127.0.0.1"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing active record",
			config: &reghost.Config{
				ActiveRecord: "",
				Records: map[string][]reghost.Record{
					"default": {{Domain: "test.local", IP: "127.0.0.1"}},
				},
			},
			wantErr: true,
		},
		{
			name: "active record not found",
			config: &reghost.Config{
				ActiveRecord: "nonexistent",
				Records: map[string][]reghost.Record{
					"default": {{Domain: "test.local", IP: "127.0.0.1"}},
				},
			},
			wantErr: true,
		},
		{
			name: "empty domain",
			config: &reghost.Config{
				ActiveRecord: "default",
				Records: map[string][]reghost.Record{
					"default": {{Domain: "", IP: "127.0.0.1"}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "reghost.yml")

	// Load config (should create default)
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load/create default config: %v", err)
	}

	// Verify default config
	if cfg.ActiveRecord != "default" {
		t.Errorf("Expected activeRecord 'default', got '%s'", cfg.ActiveRecord)
	}

	defaultRecords, ok := cfg.Records["default"]
	if !ok {
		t.Fatal("default record set not found")
	}

	if len(defaultRecords) != 1 {
		t.Errorf("Expected 1 default record, got %d", len(defaultRecords))
	}

	if defaultRecords[0].Domain != "reghost.local" {
		t.Errorf("Expected domain 'reghost.local', got '%s'", defaultRecords[0].Domain)
	}
}
