package resolver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bilgehannal/reghost/internal/utils"
	"github.com/bilgehannal/reghost/pkg/reghost"
)

const (
	macOSResolverDir = "/etc/resolver"
)

// Manager handles dynamic creation and cleanup of /etc/resolver files
type Manager struct {
	logger           *utils.Logger
	bindIP           string
	managedDomains   map[string]bool // Track which domains we've created files for
	resolverFilesDir string
}

// NewManager creates a new resolver manager
func NewManager(bindIP string, logger *utils.Logger) *Manager {
	return &Manager{
		logger:           logger,
		bindIP:           bindIP,
		managedDomains:   make(map[string]bool),
		resolverFilesDir: macOSResolverDir,
	}
}

// UpdateResolverFiles creates/updates resolver files based on active records
func (m *Manager) UpdateResolverFiles(records []reghost.Record) error {
	// Extract unique domain suffixes from records
	suffixes := m.extractDomainSuffixes(records)

	if len(suffixes) == 0 {
		m.logger.Warn("No domain suffixes found in active records")
		return nil
	}

	// Create /etc/resolver directory if it doesn't exist
	if err := m.ensureResolverDir(); err != nil {
		return fmt.Errorf("failed to ensure resolver directory: %w", err)
	}

	// Track new domains for this update
	newDomains := make(map[string]bool)
	changesMade := false

	// Create/update resolver files for each suffix
	for _, suffix := range suffixes {
		// Check if this is a new domain
		isNew := !m.managedDomains[suffix]

		if err := m.createResolverFile(suffix); err != nil {
			m.logger.Error("Failed to create resolver file for %s: %v", suffix, err)
			continue
		}

		if isNew {
			changesMade = true
		}
		newDomains[suffix] = true
	}

	// Remove resolver files for domains that are no longer in the config
	for domain := range m.managedDomains {
		if !newDomains[domain] {
			if err := m.removeResolverFile(domain); err != nil {
				m.logger.Error("Failed to remove resolver file for %s: %v", domain, err)
			} else {
				changesMade = true
			}
		}
	}

	// Update managed domains
	m.managedDomains = newDomains

	// Only flush DNS cache if changes were made
	if changesMade {
		m.logger.Info("Resolver configuration updated for domains: %v", suffixes)
		m.flushDNSCache()
	}

	return nil
}

// extractDomainSuffixes extracts domain suffixes from record patterns
func (m *Manager) extractDomainSuffixes(records []reghost.Record) []string {
	suffixMap := make(map[string]bool)

	for _, record := range records {
		domain := record.Domain

		// Skip empty domains
		if domain == "" {
			continue
		}

		// Extract suffix from different pattern types
		suffix := m.extractSuffix(domain)
		if suffix != "" {
			suffixMap[suffix] = true
		}
	}

	// Convert map to slice
	suffixes := make([]string, 0, len(suffixMap))
	for suffix := range suffixMap {
		suffixes = append(suffixes, suffix)
	}

	return suffixes
}

// extractSuffix extracts the domain suffix from a pattern
func (m *Manager) extractSuffix(pattern string) string {
	// Remove trailing dot if present
	pattern = strings.TrimSuffix(pattern, ".")

	// Remove leading ^ and trailing $ for regex patterns
	pattern = strings.TrimPrefix(pattern, "^")
	pattern = strings.TrimSuffix(pattern, "$")

	// Pattern examples:
	// 1. Plain domain: "myhost" -> "myhost"
	// 2. Wildcard regex: "^[a-zA-Z0-9-]+.myhost.$" -> "myhost"
	// 3. Subdomain: "api.example.com" -> "example.com"
	// 4. Complex regex: "^(api|web).example.com$" -> "example.com"

	// Try to find the base domain using regex
	// Look for patterns like: .something or just something at the end

	// Remove common regex patterns
	pattern = strings.ReplaceAll(pattern, "\\.", ".")
	pattern = regexp.MustCompile(`\[.*?\]`).ReplaceAllString(pattern, "")   // Remove character classes
	pattern = regexp.MustCompile(`\(.*?\)\+`).ReplaceAllString(pattern, "") // Remove groups
	pattern = regexp.MustCompile(`\(.*?\)\*`).ReplaceAllString(pattern, "") // Remove groups
	pattern = regexp.MustCompile(`\(.*?\)\?`).ReplaceAllString(pattern, "") // Remove groups
	pattern = strings.ReplaceAll(pattern, "+", "")
	pattern = strings.ReplaceAll(pattern, "*", "")
	pattern = strings.ReplaceAll(pattern, "?", "")
	pattern = strings.ReplaceAll(pattern, "(", "")
	pattern = strings.ReplaceAll(pattern, ")", "")
	pattern = strings.ReplaceAll(pattern, "|", "")

	// Split by dots and take the last meaningful part(s)
	parts := strings.Split(pattern, ".")

	// Filter out empty parts
	var cleanParts []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			cleanParts = append(cleanParts, part)
		}
	}

	if len(cleanParts) == 0 {
		return ""
	}

	// For patterns like "subdomain.domain.tld", we want at least the last part
	// For patterns like "domain", we want just that
	if len(cleanParts) >= 1 {
		// Return the last part (main domain suffix)
		return cleanParts[len(cleanParts)-1]
	}

	return ""
}

// ensureResolverDir creates /etc/resolver directory if it doesn't exist
func (m *Manager) ensureResolverDir() error {
	if _, err := os.Stat(m.resolverFilesDir); os.IsNotExist(err) {
		cmd := exec.Command("mkdir", "-p", m.resolverFilesDir)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to create directory: %w (output: %s)", err, string(output))
		}
		m.logger.Info("Created directory: %s", m.resolverFilesDir)
	}
	return nil
}

// createResolverFile creates a resolver file for a domain suffix
func (m *Manager) createResolverFile(suffix string) error {
	filePath := filepath.Join(m.resolverFilesDir, suffix)
	expectedContent := fmt.Sprintf("nameserver %s\nport 53\n", m.bindIP)

	// Check if file already exists and has correct content
	if existingContent, err := os.ReadFile(filePath); err == nil {
		if string(existingContent) == expectedContent {
			// File exists with correct content, no need to recreate
			return nil
		}
		// File exists but content is wrong, will recreate
		m.logger.Info("Resolver file %s has incorrect content, updating...", filePath)
	}

	// Write the file
	cmd := exec.Command("tee", filePath)
	cmd.Stdin = strings.NewReader(expectedContent)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to write resolver file: %w (output: %s)", err, string(output))
	}

	m.logger.Info("✓ Created resolver file: %s", filePath)
	return nil
}

// removeResolverFile removes a resolver file for a domain suffix
func (m *Manager) removeResolverFile(suffix string) error {
	filePath := filepath.Join(m.resolverFilesDir, suffix)

	cmd := exec.Command("rm", "-f", filePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove resolver file: %w (output: %s)", err, string(output))
	}

	m.logger.Info("✓ Removed resolver file: %s", filePath)
	delete(m.managedDomains, suffix)
	return nil
}

// CleanupAll removes all managed resolver files
func (m *Manager) CleanupAll() error {
	m.logger.Info("Cleaning up all resolver files...")

	for domain := range m.managedDomains {
		if err := m.removeResolverFile(domain); err != nil {
			m.logger.Error("Failed to remove resolver file for %s: %v", domain, err)
		}
	}

	m.flushDNSCache()
	m.managedDomains = make(map[string]bool)

	return nil
}

// flushDNSCache flushes the macOS DNS cache
func (m *Manager) flushDNSCache() {
	exec.Command("dscacheutil", "-flushcache").Run()
	exec.Command("killall", "-HUP", "mDNSResponder").Run()
	m.logger.Info("✓ DNS cache flushed")
}

// GetManagedDomains returns the list of currently managed domain suffixes
func (m *Manager) GetManagedDomains() []string {
	domains := make([]string, 0, len(m.managedDomains))
	for domain := range m.managedDomains {
		domains = append(domains, domain)
	}
	return domains
}
