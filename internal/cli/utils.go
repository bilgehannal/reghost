package cli

import (
	"fmt"

	"github.com/bilgehannal/reghost/pkg/reghost"
)

// PrintConfig prints the configuration in a human-readable format
func PrintConfig(cfg *reghost.Config) {
	fmt.Printf("\n=== reghost Configuration ===\n\n")
	fmt.Printf("Active Record Set: %s\n\n", cfg.ActiveRecord)

	fmt.Println("Record Sets:")
	for name, records := range cfg.Records {
		marker := " "
		if name == cfg.ActiveRecord {
			marker = "*"
		}
		fmt.Printf("  %s %s (%d records)\n", marker, name, len(records))

		for i, record := range records {
			fmt.Printf("    [%d] %s -> %s\n", i, record.Domain, record.IP)
		}
		fmt.Println()
	}
}

// PrintActiveRecord prints only the active record set
func PrintActiveRecord(cfg *reghost.Config) {
	fmt.Printf("\n=== Active Record Set: %s ===\n\n", cfg.ActiveRecord)

	activeRecords := cfg.GetActiveRecords()
	if len(activeRecords) == 0 {
		fmt.Println("No records found in active record set")
		return
	}

	fmt.Printf("Records (%d total):\n", len(activeRecords))
	for i, record := range activeRecords {
		fmt.Printf("  [%d] %s -> %s\n", i, record.Domain, record.IP)
	}
	fmt.Println()
}

// PrintError prints an error message
func PrintError(format string, args ...interface{}) {
	fmt.Printf("✗ Error: "+format+"\n", args...)
}

// PrintSuccess prints a success message
func PrintSuccess(format string, args ...interface{}) {
	fmt.Printf("✓ "+format+"\n", args...)
}
