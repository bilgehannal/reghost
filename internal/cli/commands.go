package cli

import (
	"fmt"
	"os"

	"github.com/bilgehannal/reghost/internal/config"
	"github.com/spf13/cobra"
)

var (
	configPath string
)

// NewRootCommand creates the root command for reghostctl
func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reghostctl",
		Short: "reghost configuration management tool",
		Long:  `reghostctl is a CLI tool for managing reghost DNS server configuration.`,
	}

	cmd.PersistentFlags().StringVarP(&configPath, "config", "c", "/etc/reghost.yml", "Path to config file")

	// Add subcommands
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newSetActiveCommand())
	cmd.AddCommand(newAddRecordCommand())
	cmd.AddCommand(newRemoveRecordCommand())
	cmd.AddCommand(newCreateSetCommand())
	cmd.AddCommand(newDeleteSetCommand())
	cmd.AddCommand(newShowCommand())

	return cmd
}

// newListCommand creates the list command
func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all record sets",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			PrintConfig(cfg)
			return nil
		},
	}
}

// newSetActiveCommand creates the set-active command
func newSetActiveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set-active <record-set>",
		Short: "Set the active record set",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			writer := config.NewWriter(configPath)
			if err := writer.SetActiveRecord(args[0]); err != nil {
				return err
			}

			fmt.Printf("✓ Active record set changed to: %s\n", args[0])
			return nil
		},
	}
}

// newAddRecordCommand creates the add-record command
func newAddRecordCommand() *cobra.Command {
	var (
		domain string
		ip     string
	)

	cmd := &cobra.Command{
		Use:   "add-record <record-set>",
		Short: "Add a record to a record set",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			record := config.Record{
				Domain: domain,
				IP:     ip,
			}

			writer := config.NewWriter(configPath)
			if err := writer.AddRecord(args[0], record); err != nil {
				return err
			}

			fmt.Printf("✓ Record added to '%s': %s -> %s\n", args[0], domain, ip)
			return nil
		},
	}

	cmd.Flags().StringVarP(&domain, "domain", "d", "", "Domain pattern (required)")
	cmd.Flags().StringVarP(&ip, "ip", "i", "", "IP address (required)")
	cmd.MarkFlagRequired("domain")
	cmd.MarkFlagRequired("ip")

	return cmd
}

// newRemoveRecordCommand creates the remove-record command
func newRemoveRecordCommand() *cobra.Command {
	var index int

	cmd := &cobra.Command{
		Use:   "remove-record <record-set>",
		Short: "Remove a record from a record set",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			writer := config.NewWriter(configPath)
			if err := writer.RemoveRecord(args[0], index); err != nil {
				return err
			}

			fmt.Printf("✓ Record removed from '%s' at index %d\n", args[0], index)
			return nil
		},
	}

	cmd.Flags().IntVarP(&index, "index", "i", -1, "Index of the record to remove (required)")
	cmd.MarkFlagRequired("index")

	return cmd
}

// newCreateSetCommand creates the create-set command
func newCreateSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "create-set <record-set>",
		Short: "Create a new record set",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			writer := config.NewWriter(configPath)
			if err := writer.CreateRecordSet(args[0]); err != nil {
				return err
			}

			fmt.Printf("✓ Record set '%s' created\n", args[0])
			return nil
		},
	}
}

// newDeleteSetCommand creates the delete-set command
func newDeleteSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete-set <record-set>",
		Short: "Delete a record set",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			writer := config.NewWriter(configPath)
			if err := writer.DeleteRecordSet(args[0]); err != nil {
				return err
			}

			fmt.Printf("✓ Record set '%s' deleted\n", args[0])
			return nil
		},
	}
}

// newShowCommand creates the show command
func newShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show active record set",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			PrintActiveRecord(cfg)
			return nil
		},
	}
}

// Execute runs the CLI
func Execute() {
	if err := NewRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
