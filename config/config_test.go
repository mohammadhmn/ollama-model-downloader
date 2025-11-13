package config

import (
	"flag"
	"testing"
)

func TestParse(t *testing.T) {
	// Test default values
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)

	cfg, err := Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if cfg.Registry != "https://registry.ollama.ai" {
		t.Errorf("Expected registry 'https://registry.ollama.ai', got '%s'", cfg.Registry)
	}

	if cfg.Concurrency != 4 {
		t.Errorf("Expected concurrency 4, got %d", cfg.Concurrency)
	}

	if cfg.Retries != 3 {
		t.Errorf("Expected retries 3, got %d", cfg.Retries)
	}

	if cfg.Timeout != 0 {
		t.Errorf("Expected timeout 0, got %v", cfg.Timeout)
	}
}

func TestArchFromGo(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"amd64", "amd64"},
		{"arm64", "arm64"},
		{"386", "386"},
		{"arm", "arm"},
	}

	for _, test := range tests {
		if got := archFromGo(test.input); got != test.expected {
			t.Errorf("archFromGo(%s) = %s, expected %s", test.input, got, test.expected)
		}
	}
}
