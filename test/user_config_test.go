package test

import (
	"testing"

	"github.com/bilgehannal/reghost/pkg/reghost"
)

func TestUserConfiguration(t *testing.T) {
	// This is the user's configuration
	config := &reghost.Config{
		ActiveRecord: "record1",
		Records: map[string][]reghost.Record{
			"record1": {
				{
					Domain: "^[a-zA-Z0-9-]+\\.myhost\\.$",
					IP:     "10.113.241.216",
				},
				{
					Domain: "myhost",
					IP:     "10.113.241.216",
				},
			},
			"record2": {
				{
					Domain: "^[a-zA-Z0-9-]+\\.myhost\\.$",
					IP:     "10.113.241.217%", // INVALID IP with trailing %
				},
			},
		},
	}

	// Test validation
	err := config.Validate()
	if err != nil {
		t.Logf("Config validation error: %v", err)
	}

	// Get active records
	activeRecords := config.GetActiveRecords()
	if len(activeRecords) == 0 {
		t.Fatal("No active records found")
	}

	// Create matcher with active records
	matcher := reghost.NewMatcher(activeRecords)

	// Test cases
	testCases := []struct {
		domain   string
		expected string
		found    bool
	}{
		{"x.myhost", "10.113.241.216", true},
		{"test.myhost", "10.113.241.216", true},
		{"abc-123.myhost", "10.113.241.216", true},
		{"myhost", "10.113.241.216", true},
		{"something.else", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.domain, func(t *testing.T) {
			ip, found := matcher.Match(tc.domain)
			if found != tc.found {
				t.Errorf("For domain %s: expected found=%v, got found=%v", tc.domain, tc.found, found)
			}
			if found && ip != tc.expected {
				t.Errorf("For domain %s: expected IP=%s, got IP=%s", tc.domain, tc.expected, ip)
			}
			t.Logf("Domain: %s -> IP: %s (found: %v)", tc.domain, ip, found)
		})
	}
}
