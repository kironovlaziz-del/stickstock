// Package i18n loads locale JSON files (shared with the frontend, see
// /locales at the repo root) and resolves translated strings for
// server-generated messages (errors, emails, alert notifications).
//
// Supported locales: en.
package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var SupportedLocales = []string{"en"}

type Bundle struct {
	mu       sync.RWMutex
	messages map[string]map[string]string // locale -> key -> value
	fallback string
}

func Load(dir, fallback string) (*Bundle, error) {
	b := &Bundle{
		messages: make(map[string]map[string]string),
		fallback: fallback,
	}

	for _, locale := range SupportedLocales {
		path := filepath.Join(dir, locale+".json")
		data, err := os.ReadFile(path)
		if err != nil {
			if locale == fallback {
				return nil, fmt.Errorf("required fallback locale %q missing: %w", locale, err)
			}
			// Non-fallback locales are optional at boot time so the
			// service still starts if a translation file is incomplete.
			continue
		}

		var m map[string]string
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("parsing locale %q: %w", locale, err)
		}
		b.messages[locale] = m
	}

	return b, nil
}

// T resolves a message key for the given locale, falling back to the
// bundle's default locale, and finally to the key itself if nothing matches.
func (b *Bundle) T(locale, key string) string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if m, ok := b.messages[locale]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	if m, ok := b.messages[b.fallback]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	return key
}

// IsSupported reports whether locale is one Claude has translations for.
func IsSupported(locale string) bool {
	for _, l := range SupportedLocales {
		if l == locale {
			return true
		}
	}
	return false
}
