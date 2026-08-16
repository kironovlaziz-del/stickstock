package osint

import (
	"context"
	"net"
)

type DNSResult struct {
	A     []string `json:"a,omitempty"`
	AAAA  []string `json:"aaaa,omitempty"`
	MX    []string `json:"mx,omitempty"`
	TXT   []string `json:"txt,omitempty"`
	NS    []string `json:"ns,omitempty"`
	CNAME string   `json:"cname,omitempty"`
}

// DNSLookup gathers the common record types for domain. Each record type
// is looked up independently and a failure on one (e.g. no MX records)
// doesn't fail the others — a domain with no TXT records isn't an error,
// it's just empty.
func DNSLookup(ctx context.Context, domain string) (*DNSResult, error) {
	result := &DNSResult{}
	resolver := net.DefaultResolver

	if ips, err := resolver.LookupIP(ctx, "ip4", domain); err == nil {
		for _, ip := range ips {
			result.A = append(result.A, ip.String())
		}
	}
	if ips, err := resolver.LookupIP(ctx, "ip6", domain); err == nil {
		for _, ip := range ips {
			result.AAAA = append(result.AAAA, ip.String())
		}
	}
	if mxRecords, err := resolver.LookupMX(ctx, domain); err == nil {
		for _, mx := range mxRecords {
			result.MX = append(result.MX, mx.Host)
		}
	}
	if txtRecords, err := resolver.LookupTXT(ctx, domain); err == nil {
		result.TXT = txtRecords
	}
	if nsRecords, err := resolver.LookupNS(ctx, domain); err == nil {
		for _, ns := range nsRecords {
			result.NS = append(result.NS, ns.Host)
		}
	}
	if cname, err := resolver.LookupCNAME(ctx, domain); err == nil {
		result.CNAME = cname
	}

	return result, nil
}
