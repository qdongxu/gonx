// Package config defines the configuration parsing interface.
// Phase 0: skeleton only; nginx.conf parser deferred to Phase 1.
package config

import "io"

// ConfigParser reads a configuration source and produces a structured
// representation. Implementations may parse nginx.conf, JSON, or other
// formats.
type ConfigParser interface {
	Parse(r io.Reader) (*Config, error)
}

// Config is the top-level server configuration.
// Fields are placeholders; full structure defined in later phases.
type Config struct {
	Version string   // e.g., "1.27.0"
	Blocks  []Block  // server, location, upstream, etc.
}

// Block represents a configuration context (server, location, http, etc.).
// Phase 0: minimal placeholder.
type Block struct {
	Type   string            // "server", "location", "upstream"
	Params map[string]string // directive parameters
}

// PlaceholderParser implements ConfigParser with a no-op parse.
// Used for interface wiring tests in Phase 0.
type PlaceholderParser struct{}

// Parse returns an empty Config. Real parsing deferred to Phase 1.
func (p *PlaceholderParser) Parse(r io.Reader) (*Config, error) {
	return &Config{Version: "0.0.0-placeholder"}, nil
}

// NewPlaceholderParser creates a no-op parser for testing.
func NewPlaceholderParser() ConfigParser {
	return &PlaceholderParser{}
}

// NewParser creates a real nginx config parser.
func NewParser() ConfigParser {
	return NewNginxParser()
}
