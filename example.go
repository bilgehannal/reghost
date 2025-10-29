package main

import (
	"log"
	"net"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/miekg/dns"
)

const (
	targetIP          = "10.113.241.216"
	dnsBindAddr       = "127.0.0.2:53"
	macOSResolverFile = "/etc/resolver/testbox"
	resolvConfFile    = "/etc/resolv.conf"
	checkInterval     = 30 * time.Second
)

var testboxRegex = regexp.MustCompile(`^[a-zA-Z0-9-]+\.testbox\.$`)

// handleDNSRequest processes incoming DNS requests
func handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)

	// Process each question in the request
	for _, q := range r.Question {
		qname := strings.ToLower(q.Name)

		log.Printf("Query received: %s (type: %s)", qname, dns.TypeToString[q.Qtype])

		// Check if it matches *.testbox pattern and is an A record query
		if testboxRegex.MatchString(qname) && q.Qtype == dns.TypeA {
			log.Printf("✓ Matched *.testbox pattern: %s -> %s", qname, targetIP)

			// Create A record response
			rr := &dns.A{
				Hdr: dns.RR_Header{
					Name:   q.Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    300,
				},
				A: net.ParseIP(targetIP),
			}
			m.Answer = append(m.Answer, rr)
		} else {
			// Don't respond - return NXDOMAIN, system will try next DNS server
			log.Printf("✗ No match for: %s - returning NXDOMAIN", qname)
			m.SetRcode(r, dns.RcodeNameError)
		}
	}

	// Send response
	if err := w.WriteMsg(m); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

// setupLoopbackAlias creates 127.0.0.2 loopback alias if it doesn't exist
func setupLoopbackAlias() error {
	switch runtime.GOOS {
	case "darwin":
		// Check if 127.0.0.2 already exists
		cmd := exec.Command("ifconfig", "lo0")
		output, err := cmd.Output()
		if err != nil {
			return err
		}

		if strings.Contains(string(output), "127.0.0.2") {
			log.Printf("✓ Loopback alias 127.0.0.2 already exists")
			return nil
		}

		// Create the alias
		log.Printf("Creating loopback alias 127.0.0.2...")
		cmd = exec.Command("sudo", "ifconfig", "lo0", "alias", "127.0.0.2", "up")
		if err := cmd.Run(); err != nil {
			log.Printf("✗ Failed to create loopback alias: %v", err)
			log.Printf("  Please run manually: sudo ifconfig lo0 alias 127.0.0.2 up")
			return err
		}
		log.Printf("✓ Created loopback alias 127.0.0.2")
		return nil

	case "linux":
		// Check if 127.0.0.2 already exists
		cmd := exec.Command("ip", "addr", "show", "lo")
		output, err := cmd.Output()
		if err != nil {
			return err
		}

		if strings.Contains(string(output), "127.0.0.2") {
			log.Printf("✓ Loopback alias 127.0.0.2 already exists")
			return nil
		}

		// Create the alias
		log.Printf("Creating loopback alias 127.0.0.2...")
		cmd = exec.Command("sudo", "ip", "addr", "add", "127.0.0.2/8", "dev", "lo")
		if err := cmd.Run(); err != nil {
			log.Printf("✗ Failed to create loopback alias: %v", err)
			log.Printf("  Please run manually: sudo ip addr add 127.0.0.2/8 dev lo")
			return err
		}
		log.Printf("✓ Created loopback alias 127.0.0.2")
		return nil

	default:
		return nil
	}
}

// ensureResolverConfigMacOS creates/maintains /etc/resolver/testbox on macOS
func ensureResolverConfigMacOS() {
	expectedContent := "nameserver 127.0.0.2\nport 53\n"

	// Check if file exists and has correct content
	content, err := os.ReadFile(macOSResolverFile)
	if err == nil && string(content) == expectedContent {
		// File exists and is correct
		return
	}

	// File is missing or incorrect, recreate it
	log.Printf("⚠ macOS resolver config missing or incorrect, recreating %s", macOSResolverFile)

	// Create directory if needed
	cmd := exec.Command("sudo", "mkdir", "-p", "/etc/resolver")
	if err := cmd.Run(); err != nil {
		log.Printf("✗ Failed to create /etc/resolver directory: %v", err)
		log.Printf("  Please run manually: sudo mkdir -p /etc/resolver")
		return
	}

	// Write the config using sudo tee
	cmd = exec.Command("sudo", "tee", macOSResolverFile)
	cmd.Stdin = strings.NewReader(expectedContent)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("✗ Failed to write %s: %v", macOSResolverFile, err)
		log.Printf("  Output: %s", output)
		return
	}

	log.Printf("✓ Successfully created %s", macOSResolverFile)

	// Flush DNS cache
	exec.Command("sudo", "dscacheutil", "-flushcache").Run()
	exec.Command("sudo", "killall", "-HUP", "mDNSResponder").Run()
	log.Printf("✓ DNS cache flushed")
}

// ensureResolvConfConfig manages /etc/resolv.conf on Linux
func ensureResolvConfConfig() {
	// Read current resolv.conf
	content, err := os.ReadFile(resolvConfFile)
	if err != nil {
		log.Printf("✗ Failed to read %s: %v", resolvConfFile, err)
		return
	}

	lines := strings.Split(string(content), "\n")
	localhostEntry := "nameserver 127.0.0.2"

	// Check if 127.0.0.2 is already the first nameserver
	firstNameserver := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "nameserver") {
			firstNameserver = trimmed
			break
		}
	}

	if firstNameserver == localhostEntry {
		// Already configured correctly
		return
	}

	log.Printf("⚠ 127.0.0.2 is not the first nameserver, updating %s", resolvConfFile)

	// Build new content with 127.0.0.2 as first nameserver
	var newLines []string
	localhostAdded := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip if this line is already our nameserver
		if trimmed == localhostEntry {
			continue
		}

		// Add our nameserver as first before any other nameserver
		if !localhostAdded && strings.HasPrefix(trimmed, "nameserver") {
			newLines = append(newLines, localhostEntry)
			localhostAdded = true
		}

		newLines = append(newLines, line)
	}

	// If no nameserver was found, add at the end
	if !localhostAdded {
		newLines = append(newLines, localhostEntry)
	}

	newContent := strings.Join(newLines, "\n")

	// Write the new config using sudo tee
	cmd := exec.Command("sudo", "tee", resolvConfFile)
	cmd.Stdin = strings.NewReader(newContent)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("✗ Failed to write %s: %v", resolvConfFile, err)
		log.Printf("  Output: %s", output)
		return
	}

	log.Printf("✓ Successfully updated %s - 127.0.0.2 is now first nameserver", resolvConfFile)
}

// ensureResolverConfig detects OS and calls appropriate function
func ensureResolverConfig() {
	switch runtime.GOOS {
	case "darwin":
		ensureResolverConfigMacOS()
	case "linux":
		ensureResolvConfConfig()
	default:
		log.Printf("⚠ Unsupported OS: %s", runtime.GOOS)
		log.Printf("  Please manually configure DNS to query 127.0.0.2:53 for *.testbox domains")
	}
}

// monitorResolverConfig continuously monitors and maintains the resolver config
func monitorResolverConfig() {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	// Initial check
	ensureResolverConfig()

	for range ticker.C {
		ensureResolverConfig()
	}
}

func main() {
	log.Printf("Starting DNS server on %s", dnsBindAddr)
	log.Printf("Routing *.testbox domains to %s", targetIP)
	log.Println("Examples: xyz.testbox, app.testbox, etc.")
	log.Println()

	// Setup loopback alias 127.0.0.2
	if err := setupLoopbackAlias(); err != nil {
		log.Printf("⚠ Warning: Could not setup loopback alias automatically")
	}

	log.Printf("Test with: dig @127.0.0.2 -p 53 xyz.testbox")
	log.Println()

	// Start resolver config monitor in background
	log.Printf("Starting resolver config monitor (checks every %v)", checkInterval)
	go monitorResolverConfig()

	// Create DNS server
	dns.HandleFunc(".", handleDNSRequest)

	server := &dns.Server{
		Addr: dnsBindAddr,
		Net:  "udp",
	}

	// Start server
	log.Printf("✓ DNS server ready and listening on %s", dnsBindAddr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	defer server.Shutdown()
}
