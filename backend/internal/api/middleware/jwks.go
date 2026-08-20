package middleware

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

type jwksResponse struct {
	Keys []jwk `json:"keys"`
}

// jwksCache fetches and caches Supabase's JWKS discovery endpoint (public
// keys for ES256-signed session tokens, once a project has rotated off
// the legacy HS256 shared secret — see
// https://supabase.com/docs/guides/auth/signing-keys). Refetches at most
// every 10 minutes, matching Supabase's own edge cache TTL for this
// endpoint.
type jwksCache struct {
	url string

	mu        sync.Mutex
	keys      map[string]*ecdsa.PublicKey
	fetchedAt time.Time
}

func newJWKSCache(supabaseURL string) *jwksCache {
	return &jwksCache{url: strings.TrimRight(supabaseURL, "/") + "/auth/v1/.well-known/jwks.json"}
}

func (c *jwksCache) publicKey(kid string) (*ecdsa.PublicKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if key, ok := c.keys[kid]; ok && time.Since(c.fetchedAt) < 10*time.Minute {
		return key, nil
	}

	if err := c.refresh(); err != nil {
		return nil, err
	}

	key, ok := c.keys[kid]
	if !ok {
		return nil, fmt.Errorf("no signing key found for kid %q", kid)
	}
	return key, nil
}

func (c *jwksCache) refresh() error {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(c.url)
	if err != nil {
		return fmt.Errorf("could not fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	var parsed jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return fmt.Errorf("could not parse JWKS: %w", err)
	}

	keys := make(map[string]*ecdsa.PublicKey, len(parsed.Keys))
	for _, k := range parsed.Keys {
		if k.Kty != "EC" || k.Crv != "P-256" {
			continue // only P-256/ES256 supported — Supabase's default asymmetric algorithm
		}
		pub, err := ecPublicKeyFromJWK(k)
		if err != nil {
			continue // skip a malformed key rather than failing the whole refresh
		}
		keys[k.Kid] = pub
	}

	c.keys = keys
	c.fetchedAt = time.Now()
	return nil
}

func ecPublicKeyFromJWK(k jwk) (*ecdsa.PublicKey, error) {
	xBytes, err := base64.RawURLEncoding.DecodeString(k.X)
	if err != nil {
		return nil, fmt.Errorf("invalid x: %w", err)
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(k.Y)
	if err != nil {
		return nil, fmt.Errorf("invalid y: %w", err)
	}

	return &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}, nil
}
