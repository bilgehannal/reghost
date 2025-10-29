package dns

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/bilgehannal/reghost/internal/resolver"
	"github.com/bilgehannal/reghost/internal/utils"
	"github.com/bilgehannal/reghost/pkg/reghost"
	"github.com/miekg/dns"
)

// Server represents a DNS server
type Server struct {
	cache              *Cache
	handler            *Handler
	logger             *utils.Logger
	udpServer          *dns.Server
	tcpServer          *dns.Server
	bindIP             string
	resolverConfigured bool
	originalResolvConf []byte            // Linux: backup of original resolv.conf
	resolverManager    *resolver.Manager // Dynamic resolver file manager
}

// NewServer creates a new DNS server
func NewServer(cache *Cache, logger *utils.Logger) *Server {
	handler := NewHandler(cache, logger)

	return &Server{
		cache:   cache,
		handler: handler,
		logger:  logger,
	}
}

// Start starts the DNS server
func (s *Server) Start() error {
	// Find and bind to a random loopback IP
	ip, err := s.bindLoopbackIP()
	if err != nil {
		return fmt.Errorf("failed to bind loopback IP: %w", err)
	}
	s.bindIP = ip

	s.logger.Info("DNS server bound to: %s:53", s.bindIP)

	// Configure system DNS resolver
	if err := s.configureSystemResolver(); err != nil {
		s.logger.Warn("Failed to configure system resolver: %v", err)
		s.logger.Info("You can manually configure DNS or use the setup-resolver tool")
	}

	// Start resolver configuration monitor
	go s.monitorResolverConfig()

	// Create UDP server
	s.udpServer = &dns.Server{
		Addr:    net.JoinHostPort(s.bindIP, "53"),
		Net:     "udp",
		Handler: s.handler,
	}

	// Create TCP server
	s.tcpServer = &dns.Server{
		Addr:    net.JoinHostPort(s.bindIP, "53"),
		Net:     "tcp",
		Handler: s.handler,
	}

	// Start servers in goroutines
	errChan := make(chan error, 2)

	go func() {
		s.logger.Info("Starting UDP DNS server on %s:53", s.bindIP)
		if err := s.udpServer.ListenAndServe(); err != nil {
			errChan <- fmt.Errorf("UDP server error: %w", err)
		}
	}()

	go func() {
		s.logger.Info("Starting TCP DNS server on %s:53", s.bindIP)
		if err := s.tcpServer.ListenAndServe(); err != nil {
			errChan <- fmt.Errorf("TCP server error: %w", err)
		}
	}()

	// Wait a bit to see if servers start successfully
	select {
	case err := <-errChan:
		return err
	case <-time.After(100 * time.Millisecond):
		return nil
	}
}

// Shutdown gracefully shuts down the DNS server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down DNS server...")

	// Cleanup system resolver configuration
	if s.resolverConfigured {
		if err := s.cleanupSystemResolver(); err != nil {
			s.logger.Error("Failed to cleanup system resolver: %v", err)
		}
	}

	// Shutdown servers
	var err error
	if s.udpServer != nil {
		if e := s.udpServer.ShutdownContext(ctx); e != nil {
			err = e
		}
	}
	if s.tcpServer != nil {
		if e := s.tcpServer.ShutdownContext(ctx); e != nil {
			err = e
		}
	}

	// Release loopback IP
	if s.bindIP != "" {
		if e := s.releaseLoopbackIP(s.bindIP); e != nil {
			s.logger.Error("Failed to release loopback IP: %v", e)
		}
	}

	return err
}

// bindLoopbackIP finds and binds a random IP in 127.0.0.0/8 range
func (s *Server) bindLoopbackIP() (string, error) {
	rand.Seed(time.Now().UnixNano())

	maxAttempts := 100
	for i := 0; i < maxAttempts; i++ {
		// Generate random IP in 127.0.0.0/8 range (avoid 127.0.0.1)
		ip := fmt.Sprintf("127.%d.%d.%d",
			rand.Intn(256),
			rand.Intn(256),
			rand.Intn(256))

		// Skip 127.0.0.1 (reserved)
		if ip == "127.0.0.1" {
			continue
		}

		// Check if IP is already in use
		if s.isIPInUse(ip) {
			s.logger.Info("IP %s is already in use, trying another...", ip)
			continue
		}

		// Try to add the IP alias
		if err := s.addLoopbackAlias(ip); err != nil {
			s.logger.Warn("Failed to add IP %s: %v, trying another...", ip, err)
			continue
		}

		return ip, nil
	}

	return "", fmt.Errorf("failed to find available loopback IP after %d attempts", maxAttempts)
}

// isIPInUse checks if an IP is already bound to loopback interface
func (s *Server) isIPInUse(ip string) bool {
	switch runtime.GOOS {
	case "darwin":
		cmd := exec.Command("ifconfig", "lo0")
		output, err := cmd.Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(output), ip)

	case "linux":
		cmd := exec.Command("ip", "addr", "show", "lo")
		output, err := cmd.Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(output), ip)

	default:
		return false
	}
}

// addLoopbackAlias adds an IP alias to the loopback interface
func (s *Server) addLoopbackAlias(ip string) error {
	switch runtime.GOOS {
	case "darwin":
		cmd := exec.Command("ifconfig", "lo0", "alias", ip, "up")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("ifconfig failed: %w (output: %s)", err, string(output))
		}
		s.logger.Info("Added loopback alias: %s", ip)
		return nil

	case "linux":
		cmd := exec.Command("ip", "addr", "add", ip+"/8", "dev", "lo")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("ip addr add failed: %w (output: %s)", err, string(output))
		}
		s.logger.Info("Added loopback alias: %s", ip)
		return nil

	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// releaseLoopbackIP removes the IP alias from the loopback interface
func (s *Server) releaseLoopbackIP(ip string) error {
	switch runtime.GOOS {
	case "darwin":
		cmd := exec.Command("ifconfig", "lo0", "-alias", ip)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("ifconfig failed: %w (output: %s)", err, string(output))
		}
		s.logger.Info("Released loopback alias: %s", ip)
		return nil

	case "linux":
		cmd := exec.Command("ip", "addr", "del", ip+"/8", "dev", "lo")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("ip addr del failed: %w (output: %s)", err, string(output))
		}
		s.logger.Info("Released loopback alias: %s", ip)
		return nil

	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// GetBindIP returns the IP address the server is bound to
func (s *Server) GetBindIP() string {
	return s.bindIP
}

// UpdateResolverFiles updates the resolver files based on new records
func (s *Server) UpdateResolverFiles(records []reghost.Record) error {
	if runtime.GOOS != "darwin" {
		// Only supported on macOS
		return nil
	}

	if s.resolverManager == nil {
		s.resolverManager = resolver.NewManager(s.bindIP, s.logger)
	}

	return s.resolverManager.UpdateResolverFiles(records)
}

// configureSystemResolver configures the system DNS resolver
func (s *Server) configureSystemResolver() error {
	switch runtime.GOOS {
	case "darwin":
		return s.configureMacOSResolver()
	case "linux":
		return s.configureLinuxResolver()
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// configureMacOSResolver creates dynamic /etc/resolver files based on active records
func (s *Server) configureMacOSResolver() error {
	s.logger.Info("Configuring macOS resolver with dynamic domain-based files...")

	// Create resolver manager if not exists
	if s.resolverManager == nil {
		s.resolverManager = resolver.NewManager(s.bindIP, s.logger)
	}

	// Get active records from cache
	records := s.cache.GetRecords()

	// Update resolver files based on active records
	if err := s.resolverManager.UpdateResolverFiles(records); err != nil {
		return fmt.Errorf("failed to update resolver files: %w", err)
	}

	s.resolverConfigured = true
	s.logger.Info("✓ DNS resolver configured - queries will be matched against your regex patterns")
	s.logger.Info("Managed domains: %v", s.resolverManager.GetManagedDomains())

	return nil
} // configureLinuxResolver adds nameserver to /etc/resolv.conf on Linux
func (s *Server) configureLinuxResolver() error {
	resolvConfFile := "/etc/resolv.conf"

	// Read and backup original resolv.conf
	content, err := exec.Command("cat", resolvConfFile).Output()
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", resolvConfFile, err)
	}
	s.originalResolvConf = content

	s.logger.Info("Configuring Linux resolver to use %s", s.bindIP)

	lines := strings.Split(string(content), "\n")
	nameserverEntry := fmt.Sprintf("nameserver %s", s.bindIP)

	// Check if already configured
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == nameserverEntry {
			s.logger.Info("✓ %s already configured in %s", s.bindIP, resolvConfFile)
			s.resolverConfigured = true
			return nil
		}
	}

	// Build new content with our nameserver first
	var newLines []string
	newLines = append(newLines, nameserverEntry)
	newLines = append(newLines, lines...)

	newContent := strings.Join(newLines, "\n")

	// Write new config
	cmd := exec.Command("tee", resolvConfFile)
	cmd.Stdin = strings.NewReader(newContent)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to write %s: %w (output: %s)", resolvConfFile, err, string(output))
	}

	s.logger.Info("✓ Updated %s - %s is now first nameserver", resolvConfFile, s.bindIP)
	s.resolverConfigured = true

	return nil
}

// cleanupSystemResolver removes DNS resolver configuration
func (s *Server) cleanupSystemResolver() error {
	switch runtime.GOOS {
	case "darwin":
		return s.cleanupMacOSResolver()
	case "linux":
		return s.cleanupLinuxResolver()
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// cleanupMacOSResolver removes all managed resolver files on macOS
func (s *Server) cleanupMacOSResolver() error {
	if s.resolverManager == nil {
		return nil
	}

	s.logger.Info("Removing all managed macOS resolver configs...")
	return s.resolverManager.CleanupAll()
}

// cleanupLinuxResolver restores original /etc/resolv.conf on Linux
func (s *Server) cleanupLinuxResolver() error {
	if s.originalResolvConf == nil {
		return nil
	}

	resolvConfFile := "/etc/resolv.conf"
	s.logger.Info("Restoring original %s", resolvConfFile)

	// Write back original content
	cmd := exec.Command("tee", resolvConfFile)
	cmd.Stdin = strings.NewReader(string(s.originalResolvConf))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to restore %s: %w (output: %s)", resolvConfFile, err, string(output))
	}

	s.logger.Info("✓ Restored original %s", resolvConfFile)

	return nil
}

// monitorResolverConfig periodically checks if resolver configuration is intact
// and restores it if it gets changed or deleted (e.g., by VPN changes)
func (s *Server) monitorResolverConfig() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for range ticker.C {
		if !s.resolverConfigured {
			continue
		}

		switch runtime.GOOS {
		case "darwin":
			s.checkAndRestoreMacOSResolver()
		case "linux":
			s.checkAndRestoreLinuxResolver()
		}
	}
}

// checkAndRestoreMacOSResolver checks if resolver files exist and restores them if needed
func (s *Server) checkAndRestoreMacOSResolver() {
	if s.resolverManager == nil {
		return
	}

	// Get current active records and update resolver files
	records := s.cache.GetRecords()
	if err := s.resolverManager.UpdateResolverFiles(records); err != nil {
		s.logger.Error("Failed to restore macOS resolver: %v", err)
	}
}

// checkAndRestoreLinuxResolver checks if /etc/resolv.conf still has our nameserver
func (s *Server) checkAndRestoreLinuxResolver() {
	resolvConfFile := "/etc/resolv.conf"

	content, err := exec.Command("cat", resolvConfFile).Output()
	if err != nil {
		s.logger.Error("Failed to read %s: %v", resolvConfFile, err)
		return
	}

	nameserverEntry := fmt.Sprintf("nameserver %s", s.bindIP)

	// Check if our nameserver is still present
	lines := strings.Split(string(content), "\n")
	found := false
	for _, line := range lines {
		if strings.TrimSpace(line) == nameserverEntry {
			found = true
			break
		}
	}

	if !found {
		s.logger.Warn("⚠ Nameserver %s removed from %s, restoring...", s.bindIP, resolvConfFile)

		// Re-add our nameserver at the top
		var newLines []string
		newLines = append(newLines, nameserverEntry)
		newLines = append(newLines, lines...)
		newContent := strings.Join(newLines, "\n")

		cmd := exec.Command("tee", resolvConfFile)
		cmd.Stdin = strings.NewReader(newContent)
		if output, err := cmd.CombinedOutput(); err != nil {
			s.logger.Error("Failed to restore %s: %v (output: %s)", resolvConfFile, err, string(output))
		} else {
			s.logger.Info("✓ Restored nameserver in %s", resolvConfFile)
		}
	}
}
