package dns

import (
	"net"
	"strings"

	"github.com/bilgehannal/reghost/internal/utils"
	"github.com/miekg/dns"
)

// Handler handles DNS requests
type Handler struct {
	cache  *Cache
	logger *utils.Logger
}

// NewHandler creates a new DNS handler
func NewHandler(cache *Cache, logger *utils.Logger) *Handler {
	return &Handler{
		cache:  cache,
		logger: logger,
	}
}

// ServeDNS handles a DNS request
func (h *Handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)

	// Process each question
	for _, q := range r.Question {
		qname := strings.ToLower(q.Name)

		h.logger.Info("DNS Query: %s (type: %s)", qname, dns.TypeToString[q.Qtype])

		// Only handle A record queries
		if q.Qtype != dns.TypeA {
			h.logger.Info("Skipping non-A record query for: %s", qname)
			continue
		}

		// Lookup in cache
		if ip, found := h.cache.Lookup(qname); found {
			h.logger.Info("Match found: %s -> %s", qname, ip)

			// Create A record response
			rr := &dns.A{
				Hdr: dns.RR_Header{
					Name:   q.Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    300,
				},
				A: net.ParseIP(ip),
			}
			m.Answer = append(m.Answer, rr)
		} else {
			h.logger.Info("No match for: %s - returning NXDOMAIN", qname)
			m.SetRcode(r, dns.RcodeNameError)
		}
	}

	// Send response
	if err := w.WriteMsg(m); err != nil {
		h.logger.Error("Error writing DNS response: %v", err)
	}
}
