package test

import (
	"testing"

	"github.com/bilgehannal/reghost/pkg/reghost"
)

func TestResolver(t *testing.T) {
	records := []reghost.Record{
		{Domain: "test.local", IP: "127.0.0.1"},
		{Domain: "^[a-z]+\\.example\\.$", IP: "10.0.0.1"},
		{Domain: "exact.match", IP: "192.168.1.1"},
	}

	resolver := reghost.NewResolver(records)

	tests := []struct {
		name     string
		domain   string
		wantIP   string
		wantFound bool
	}{
		{
			name:      "exact match with dot",
			domain:    "test.local.",
			wantIP:    "127.0.0.1",
			wantFound: true,
		},
		{
			name:      "exact match without dot",
			domain:    "test.local",
			wantIP:    "127.0.0.1",
			wantFound: true,
		},
		{
			name:      "regex match",
			domain:    "hello.example.",
			wantIP:    "10.0.0.1",
			wantFound: true,
		},
		{
			name:      "regex match without dot",
			domain:    "world.example",
			wantIP:    "10.0.0.1",
			wantFound: true,
		},
		{
			name:      "no match",
			domain:    "notfound.local",
			wantIP:    "",
			wantFound: false,
		},
		{
			name:      "case insensitive",
			domain:    "TEST.LOCAL",
			wantIP:    "127.0.0.1",
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip, found := resolver.Resolve(tt.domain)
			if found != tt.wantFound {
				t.Errorf("Resolve(%s) found = %v, want %v", tt.domain, found, tt.wantFound)
			}
			if ip != tt.wantIP {
				t.Errorf("Resolve(%s) IP = %s, want %s", tt.domain, ip, tt.wantIP)
			}
		})
	}
}

func TestMatcher(t *testing.T) {
	records := []reghost.Record{
		{Domain: "simple.test", IP: "1.1.1.1"},
		{Domain: "^prefix-.*\\.test\\.$", IP: "2.2.2.2"},
	}

	matcher := reghost.NewMatcher(records)

	// Test simple match
	ip, found := matcher.Match("simple.test")
	if !found || ip != "1.1.1.1" {
		t.Errorf("Expected match for 'simple.test', got found=%v, ip=%s", found, ip)
	}

	// Test regex match
	ip, found = matcher.Match("prefix-something.test.")
	if !found || ip != "2.2.2.2" {
		t.Errorf("Expected match for 'prefix-something.test.', got found=%v, ip=%s", found, ip)
	}

	// Test no match
	_, found = matcher.Match("nomatch.test")
	if found {
		t.Error("Expected no match for 'nomatch.test'")
	}
}

func TestMatcherUpdate(t *testing.T) {
	initialRecords := []reghost.Record{
		{Domain: "old.test", IP: "1.1.1.1"},
	}

	matcher := reghost.NewMatcher(initialRecords)

	// Verify initial record
	ip, found := matcher.Match("old.test")
	if !found || ip != "1.1.1.1" {
		t.Error("Initial record not found")
	}

	// Update with new records
	newRecords := []reghost.Record{
		{Domain: "new.test", IP: "2.2.2.2"},
	}
	matcher.Update(newRecords)

	// Old record should not be found
	_, found = matcher.Match("old.test")
	if found {
		t.Error("Old record should not be found after update")
	}

	// New record should be found
	ip, found = matcher.Match("new.test")
	if !found || ip != "2.2.2.2" {
		t.Error("New record not found after update")
	}
}
