// Package osint implements a few well-defined, legitimate lookups:
// WHOIS, DNS, and breach-checking via HaveIBeenPwned. Deliberately not
// attempting anything that requires scraping, unofficial APIs, or
// touches data people haven't consented to have checked — HIBP's own
// design (an email owner checking their own exposure) is the model here.
package osint

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

// WHOIS queries whois.iana.org for domain, and if the response contains a
// referral to a more specific registry server (the standard "whois:" or
// "refer:" line), follows that once. Good enough for most gTLDs/ccTLDs
// without hardcoding a per-TLD server table.
func WHOIS(domain string) (string, error) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return "", fmt.Errorf("domain is required")
	}

	response, err := queryWHOIS("whois.iana.org", domain)
	if err != nil {
		return "", err
	}

	if referral := extractReferral(response); referral != "" && referral != "whois.iana.org" {
		if more, err := queryWHOIS(referral, domain); err == nil {
			return more, nil
		}
		// Referral failed — the IANA response is still useful, return it.
	}

	return response, nil
}

func queryWHOIS(server, domain string) (string, error) {
	conn, err := net.DialTimeout("tcp", server+":43", 10*time.Second)
	if err != nil {
		return "", fmt.Errorf("could not reach %s: %w", server, err)
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))

	if _, err := conn.Write([]byte(domain + "\r\n")); err != nil {
		return "", err
	}

	var sb strings.Builder
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 64*1024)
	for scanner.Scan() {
		sb.WriteString(scanner.Text())
		sb.WriteString("\n")
	}
	if err := scanner.Err(); err != nil && sb.Len() == 0 {
		return "", err
	}

	return sb.String(), nil
}

func extractReferral(whoisResponse string) string {
	for _, line := range strings.Split(whoisResponse, "\n") {
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "whois:") || strings.HasPrefix(lower, "refer:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}
