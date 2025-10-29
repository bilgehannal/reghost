package reghost

// Resolver provides domain to IP resolution
type Resolver struct {
	matcher *Matcher
}

// NewResolver creates a new DNS resolver
func NewResolver(records []Record) *Resolver {
	return &Resolver{
		matcher: NewMatcher(records),
	}
}

// Resolve looks up the IP address for a given domain
func (r *Resolver) Resolve(domain string) (string, bool) {
	return r.matcher.Match(domain)
}

// UpdateRecords updates the resolver with new records
func (r *Resolver) UpdateRecords(records []Record) {
	r.matcher.Update(records)
}

// GetDomains returns all domain patterns
func (r *Resolver) GetDomains() []string {
	return r.matcher.GetDomains()
}

// GetRecords returns all records
func (r *Resolver) GetRecords() []Record {
	return r.matcher.GetRecords()
}
