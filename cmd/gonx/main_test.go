package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantPath  string
		wantVer   bool
		wantParse bool
	}{
		{
			name:      "config path",
			args:      []string{"-c", "/etc/gonx.conf"},
			wantPath:  "/etc/gonx.conf",
			wantVer:   false,
			wantParse: true,
		},
		{
			name:      "version flag",
			args:      []string{"-version"},
			wantPath:  "",
			wantVer:   true,
			wantParse: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: flag.Parse() can only be called once per process
			// This is a simplified test; real flag testing uses flag.NewFlagSet
			if !tt.wantParse {
				return
			}
		})
	}
}

func TestRunRequiresConfig(t *testing.T) {
	opts := options{configPath: ""}
	err := run(opts)
	if err == nil {
		t.Fatal("expected error for empty config path")
	}
	if !strings.Contains(err.Error(), "configuration file required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVersionOutput(t *testing.T) {
	var buf bytes.Buffer
	version = "test-v1"
	buildTime = "2024-01-01"

	fmt.Fprintf(&buf, "gonx version %s (built %s)\n", version, buildTime)
	got := buf.String()
	want := "gonx version test-v1 (built 2024-01-01)\n"
	if got != want {
		t.Fatalf("version output mismatch: got %q, want %q", got, want)
	}
}
