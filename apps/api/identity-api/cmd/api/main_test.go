package main

import (
	"regexp"
	"testing"
)

var uuidV4 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// TestUUIDGeneratorFormat verifies the generator returns distinct, well-formed
// version 4 UUIDs.
func TestUUIDGeneratorFormat(t *testing.T) {
	g := newUUIDGenerator()
	seen := make(map[string]struct{})
	for i := 0; i < 100; i++ {
		id := g.New()
		if !uuidV4.MatchString(id) {
			t.Fatalf("id %q is not a v4 UUID", id)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate id %q", id)
		}
		seen[id] = struct{}{}
	}
}

// TestLoadConfigRequiresDBPath verifies DB_PATH is mandatory and has no
// hardcoded fallback.
func TestLoadConfigRequiresDBPath(t *testing.T) {
	t.Setenv("DB_PATH", "")
	if _, err := loadConfig(); err == nil {
		t.Fatal("loadConfig() with empty DB_PATH succeeded, want error")
	}
}

// TestLoadConfigDefaults verifies non-secret defaults for port and env.
func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("DB_PATH", "./data/identity.db")
	t.Setenv("HTTP_PORT", "")
	t.Setenv("APP_ENV", "")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if cfg.httpPort != "8081" {
		t.Errorf("httpPort = %q, want 8081", cfg.httpPort)
	}
	if cfg.env != "local" {
		t.Errorf("env = %q, want local", cfg.env)
	}
}
