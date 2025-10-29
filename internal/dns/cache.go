package dns

import (
	"sync"

	"github.com/bilgehannal/reghost/pkg/reghost"
)

// Cache holds the in-memory DNS cache
type Cache struct {
	mu       sync.RWMutex
	resolver *reghost.Resolver
}

// NewCache creates a new DNS cache
func NewCache(records []reghost.Record) *Cache {
	return &Cache{
		resolver: reghost.NewResolver(records),
	}
}

// Lookup performs a DNS lookup in the cache
func (c *Cache) Lookup(domain string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.resolver.Resolve(domain)
}

// Update updates the cache with new records
func (c *Cache) Update(records []reghost.Record) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.resolver.UpdateRecords(records)
}

// GetDomains returns all domain patterns from the cache
func (c *Cache) GetDomains() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.resolver.GetDomains()
}

// GetRecords returns all records from the cache
func (c *Cache) GetRecords() []reghost.Record {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.resolver.GetRecords()
}
