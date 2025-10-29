package reghost

import (
	"regexp"
	"strings"
	"sync"
)

// Matcher handles domain matching against records
type Matcher struct {
	mu      sync.RWMutex
	records []Record
	// Cache compiled regex patterns
	regexCache map[string]*regexp.Regexp
}

// NewMatcher creates a new domain matcher
func NewMatcher(records []Record) *Matcher {
	m := &Matcher{
		records:    records,
		regexCache: make(map[string]*regexp.Regexp),
	}
	// Pre-compile regex patterns
	m.compilePatterns()
	return m
}

// compilePatterns pre-compiles all regex patterns
func (m *Matcher) compilePatterns() {
	for _, record := range m.records {
		// Check if domain looks like a regex (starts with ^)
		if strings.HasPrefix(record.Domain, "^") {
			if re, err := regexp.Compile(record.Domain); err == nil {
				m.regexCache[record.Domain] = re
			}
		}
	}
}

// Match finds the IP address for a given domain
func (m *Matcher) Match(domain string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Normalize domain to lowercase
	domain = strings.ToLower(domain)

	// Ensure domain ends with a dot (FQDN)
	if !strings.HasSuffix(domain, ".") {
		domain = domain + "."
	}

	for _, record := range m.records {
		// Try exact match first (case-insensitive)
		recordDomain := strings.ToLower(record.Domain)
		if !strings.HasSuffix(recordDomain, ".") {
			recordDomain = recordDomain + "."
		}

		if recordDomain == domain {
			return record.IP, true
		}

		// Try regex match if it's a regex pattern
		if re, ok := m.regexCache[record.Domain]; ok {
			if re.MatchString(domain) {
				return record.IP, true
			}
		}
	}

	return "", false
}

// Update replaces the current records with new ones
func (m *Matcher) Update(records []Record) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.records = records
	m.regexCache = make(map[string]*regexp.Regexp)
	m.compilePatterns()
}

// GetDomains returns all domain patterns from records
func (m *Matcher) GetDomains() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	domains := make([]string, len(m.records))
	for i, record := range m.records {
		domains[i] = record.Domain
	}
	return domains
}

// GetRecords returns all records
func (m *Matcher) GetRecords() []Record {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modifications
	records := make([]Record, len(m.records))
	copy(records, m.records)
	return records
}
