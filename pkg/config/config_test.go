package config

import (
	"strings"
	"testing"
)

func TestPlaceholderParserParse(t *testing.T) {
	p := NewPlaceholderParser()
	cfg, err := p.Parse(strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil Config")
	}
	if cfg.Version != "0.0.0-placeholder" {
		t.Fatalf("unexpected version: %s", cfg.Version)
	}
	if len(cfg.Blocks) != 0 {
		t.Fatalf("expected empty blocks, got %d", len(cfg.Blocks))
	}
}

func TestConfigStructure(t *testing.T) {
	cfg := Config{
		Version: "1.0.0",
		Blocks: []Block{
			{Type: "server", Params: map[string]string{"listen": "80"}},
			{Type: "location", Params: map[string]string{"path": "/"}},
		},
	}
	if len(cfg.Blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(cfg.Blocks))
	}
	if cfg.Blocks[0].Type != "server" {
		t.Fatalf("unexpected block type: %s", cfg.Blocks[0].Type)
	}
}

func TestBlockParams(t *testing.T) {
	b := Block{
		Type:   "upstream",
		Params: map[string]string{"backend": "127.0.0.1:8080"},
	}
	if b.Params["backend"] != "127.0.0.1:8080" {
		t.Fatalf("unexpected param value: %s", b.Params["backend"])
	}
}
