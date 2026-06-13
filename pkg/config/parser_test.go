package config

import (
	"strings"
	"testing"
)

func TestLexerBasic(t *testing.T) {
	input := `server {
	listen 80;
	server_name example.com;
}`
	tokens, err := LexAll(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []TokenType{TokenIdent, TokenLBrace, TokenIdent, TokenNumber, TokenSemicolon, TokenIdent, TokenIdent, TokenSemicolon, TokenRBrace, TokenEOF}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, exp := range expected {
		if tokens[i].Type != exp {
			t.Fatalf("token %d: expected %s, got %s (%s)", i, exp, tokens[i].Type, tokens[i].Value)
		}
	}
}

func TestLexerString(t *testing.T) {
	input := `root "/var/www/html";`
	tokens, err := LexAll(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 4 {
		t.Fatalf("expected 4 tokens, got %d", len(tokens))
	}
	if tokens[1].Type != TokenString || tokens[1].Value != "/var/www/html" {
		t.Fatalf("expected string /var/www/html, got %s (%s)", tokens[1].Type, tokens[1].Value)
	}
}

func TestLexerComment(t *testing.T) {
	input := `# this is a comment
server {}`
	tokens, err := LexAll(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Comment is preserved by LexAll; parser skips it
	if len(tokens) != 5 {
		t.Fatalf("expected 5 tokens, got %d", len(tokens))
	}
	if tokens[0].Type != TokenComment {
		t.Fatalf("expected first token COMMENT, got %s", tokens[0].Type)
	}
}

func TestParserDirective(t *testing.T) {
	input := `worker_processes 4;`
	p := NewNginxParser()
	cfg, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(cfg.Blocks))
	}
	if cfg.Blocks[0].Type != "worker_processes" {
		t.Fatalf("unexpected type: %s", cfg.Blocks[0].Type)
	}
}

func TestParserBlock(t *testing.T) {
	input := `http {
	server {
		listen 80;
	}
}`
	p := NewNginxParser()
	cfg, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Blocks) != 1 {
		t.Fatalf("expected 1 top-level block, got %d", len(cfg.Blocks))
	}
	if cfg.Blocks[0].Type != "http" {
		t.Fatalf("unexpected type: %s", cfg.Blocks[0].Type)
	}
}

func TestParserNestedBlock(t *testing.T) {
	input := `server {
	location / {
		root /var/www;
		index index.html;
	}
}`
	p := NewNginxParser()
	cfg, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(cfg.Blocks))
	}
	b := cfg.Blocks[0]
	if b.Type != "server" {
		t.Fatalf("unexpected type: %s", b.Type)
	}
	// Nested block directives are flattened into parent params
	if b.Params["root"] != "/var/www" {
		t.Fatalf("unexpected root param: %s", b.Params["root"])
	}
	if b.Params["index"] != "index.html" {
		t.Fatalf("unexpected index param: %s", b.Params["index"])
	}
}

func TestParserMultiDirective(t *testing.T) {
	input := `server_name example.com www.example.com;`
	p := NewNginxParser()
	cfg, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(cfg.Blocks))
	}
	b := cfg.Blocks[0]
	if b.Type != "server_name" {
		t.Fatalf("unexpected type: %s", b.Type)
	}
	if b.Params["example.com"] != "www.example.com" {
		t.Fatalf("unexpected params: %v", b.Params)
	}
}

func TestParserEmpty(t *testing.T) {
	input := ``
	p := NewNginxParser()
	cfg, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Blocks) != 0 {
		t.Fatalf("expected 0 blocks, got %d", len(cfg.Blocks))
	}
}

func TestParserWithComments(t *testing.T) {
	input := `# main config
worker_processes 4; # cpu count`
	p := NewNginxParser()
	cfg, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(cfg.Blocks))
	}
}

func TestParserCompleteConfig(t *testing.T) {
	input := `worker_processes 1;

events {
	worker_connections 1024;
}

http {
	server {
		listen 8080;
		server_name localhost;
		location / {
			root /var/www;
			index index.html;
		}
	}
}`
	p := NewNginxParser()
	cfg, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(cfg.Blocks))
	}
	if cfg.Blocks[0].Type != "worker_processes" {
		t.Fatalf("unexpected first block: %s", cfg.Blocks[0].Type)
	}
	if cfg.Blocks[1].Type != "events" {
		t.Fatalf("unexpected second block: %s", cfg.Blocks[1].Type)
	}
	if cfg.Blocks[2].Type != "http" {
		t.Fatalf("unexpected third block: %s", cfg.Blocks[2].Type)
	}
}

func TestParserInterface(t *testing.T) {
	var p ConfigParser = NewNginxParser()
	cfg, err := p.Parse(strings.NewReader(`daemon off;`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestParserMissingSemicolon(t *testing.T) {
	input := `worker_processes 4`
	p := NewNginxParser()
	_, err := p.Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for missing semicolon")
	}
}

func TestParserBlockMissingBrace(t *testing.T) {
	input := `http {`
	p := NewNginxParser()
	_, err := p.Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for missing closing brace")
	}
}
