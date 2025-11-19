package config

import (
	"errors"
	"strings"
)

// AppConfig represents runtime-configurable settings shared between master and client.
type AppConfig struct {
	URLs []string `json:"urls"`
}

var defaultURLs = []string{
	"http://158.108.30.225:5678/form/07e92300-17f4-4265-be50-c42dae953ffb",
	"http://158.108.16.32",
}

// DefaultAppConfig returns a copy of the built-in defaults.
func DefaultAppConfig() AppConfig {
	urls := make([]string, len(defaultURLs))
	copy(urls, defaultURLs)
	return AppConfig{
		URLs: urls,
	}
}

// Normalize removes duplicates/empty values.
func Normalize(cfg AppConfig) AppConfig {
	cleaned := AppConfig{
		URLs: make([]string, 0, len(cfg.URLs)),
	}

	seen := make(map[string]struct{})
	for _, raw := range cfg.URLs {
		url := strings.TrimSpace(raw)
		if url == "" {
			continue
		}
		if _, exists := seen[url]; exists {
			continue
		}
		cleaned.URLs = append(cleaned.URLs, url)
		seen[url] = struct{}{}
	}

	return cleaned
}

// MergeWithDefaults falls back to defaults when nothing is configured.
func MergeWithDefaults(cfg AppConfig) AppConfig {
	cleaned := Normalize(cfg)
	if len(cleaned.URLs) == 0 {
		return DefaultAppConfig()
	}
	return cleaned
}

// Validate ensures at least one URL is configured.
func (cfg AppConfig) Validate() error {
	if len(cfg.URLs) == 0 {
		return errors.New("at least one URL is required")
	}
	return nil
}
