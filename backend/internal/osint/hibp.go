package osint

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type hibpBreach struct {
	Name string `json:"Name"`
}

// CheckBreaches queries HaveIBeenPwned's breach API for email. Requires
// an API key (HIBP has been a paid API for a while) — set HIBP_API_KEY.
// This mirrors HIBP's own intended use: checking exposure for an email
// you have a legitimate reason to check (typically your own), not a way
// to retrieve anyone's actual breached data — HIBP never returns
// passwords, just which named breaches an address appeared in.
func CheckBreaches(apiKey, email string) ([]string, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("breach checking is not configured (HIBP_API_KEY is empty)")
	}
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	endpoint := "https://haveibeenpwned.com/api/v3/breachedaccount/" + url.PathEscape(email)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("hibp-api-key", apiKey)
	req.Header.Set("User-Agent", "StickStock-OSINT")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return []string{}, nil // HIBP's documented "no breaches found" response
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HIBP returned status %d: %s", resp.StatusCode, string(body))
	}

	var breaches []hibpBreach
	if err := json.NewDecoder(resp.Body).Decode(&breaches); err != nil {
		return nil, err
	}

	names := make([]string, len(breaches))
	for i, b := range breaches {
		names[i] = b.Name
	}
	return names, nil
}
