package config

import (
	"strings"
	"testing"
)

// Helper: snapshot relevant env vars and restore on cleanup.
func setEnv(t *testing.T, kv map[string]string) {
	t.Helper()
	for k, v := range kv {
		t.Setenv(k, v)
	}
}

func TestLoad_DefaultsOMBaseURLWhenEmpty(t *testing.T) {
	setEnv(t, map[string]string{
		"OM_BASE_URL": "",
		"OM_URL":      "",
		"OM_TOKEN":    "tok",
	})
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(cfg.OMBaseURL, "http://") {
		t.Fatalf("expected default http URL, got %q", cfg.OMBaseURL)
	}
}

func TestLoad_RequiresAuth(t *testing.T) {
	setEnv(t, map[string]string{
		"OM_BASE_URL": "http://localhost:8585",
		"OM_TOKEN":    "",
		"OM_EMAIL":    "",
		"OM_PASSWORD": "",
	})
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "OM_TOKEN") {
		t.Fatalf("expected auth error, got %v", err)
	}
}

func TestLoad_RejectsNonHTTPBaseURL(t *testing.T) {
	setEnv(t, map[string]string{
		"OM_BASE_URL": "ftp://bad",
		"OM_TOKEN":    "tok",
	})
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "http://") {
		t.Fatalf("expected scheme error, got %v", err)
	}
}

func TestLoad_RejectsWildcardOriginWithAPIKey(t *testing.T) {
	setEnv(t, map[string]string{
		"OM_BASE_URL":     "http://localhost:8585",
		"OM_TOKEN":        "tok",
		"API_KEY":         "k",
		"ALLOWED_ORIGINS": "*,http://x",
	})
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "ALLOWED_ORIGINS") {
		t.Fatalf("expected origins error, got %v", err)
	}
}

func TestLoad_HappyPath(t *testing.T) {
	setEnv(t, map[string]string{
		"OM_BASE_URL":     "http://localhost:8585",
		"OM_TOKEN":        "tok",
		"ALLOWED_ORIGINS": "http://localhost:5173",
		"BACKEND_PORT":    "1234",
		"PORT":            "",
	})
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "1234" {
		t.Fatalf("port = %q, want 1234", cfg.Port)
	}
	if cfg.OMBaseURL != "http://localhost:8585" {
		t.Fatalf("OM base url = %q", cfg.OMBaseURL)
	}
	if !cfg.LLMEnabled && cfg.LLMAPIKey != "" {
		t.Fatal("LLMEnabled inconsistent with key")
	}
	if cfg.RateLimitRPS <= 0 || cfg.RateLimitBurst <= 0 {
		t.Fatalf("rate limit defaults wrong: rps=%v burst=%v", cfg.RateLimitRPS, cfg.RateLimitBurst)
	}
}

func TestLoad_PortFallbackToPORTEnv(t *testing.T) {
	setEnv(t, map[string]string{
		"OM_BASE_URL":  "http://localhost:8585",
		"OM_TOKEN":     "tok",
		"BACKEND_PORT": "",
		"PORT":         "9090",
	})
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "9090" {
		t.Fatalf("expected PORT fallback, got %q", cfg.Port)
	}
}

func TestLoad_FrontendDistMadeAbsolute(t *testing.T) {
	setEnv(t, map[string]string{
		"OM_BASE_URL":   "http://localhost:8585",
		"OM_TOKEN":      "tok",
		"FRONTEND_DIST": "./dist",
	})
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(cfg.FrontendDist, "/") {
		t.Fatalf("expected absolute path, got %q", cfg.FrontendDist)
	}
}
