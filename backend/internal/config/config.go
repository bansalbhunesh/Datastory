package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port            string
	OMBaseURL       string
	OMToken         string
	OMEmail         string
	OMPassword      string
	LLMAPIKey       string
	LLMModel        string
	LLMEnabled      bool
	RequestTimeout  time.Duration
	CacheTTL        time.Duration
	AllowedOrigins  []string
	LogLevel        string
	MaxBodyBytes    int64
	APIKey          string
	RateLimitRPS    float64
	RateLimitBurst  int
	IncidentLogPath string
}

func Load() (Config, error) {
	_ = godotenv.Load("../.env")
	_ = godotenv.Load(".env")

	cfg := Config{
		Port:            get("BACKEND_PORT", "8080"),
		OMBaseURL:       strings.TrimRight(get("OM_BASE_URL", get("OM_URL", "http://localhost:8585")), "/"),
		OMToken:         strings.TrimSpace(os.Getenv("OM_TOKEN")),
		OMEmail:         strings.TrimSpace(os.Getenv("OM_EMAIL")),
		OMPassword:      os.Getenv("OM_PASSWORD"),
		LLMAPIKey:       strings.TrimSpace(os.Getenv("CLAUDE_API_KEY")),
		LLMModel:        get("CLAUDE_MODEL", "claude-sonnet-4-20250514"),
		RequestTimeout:  duration("REQUEST_TIMEOUT", 45*time.Second),
		CacheTTL:        duration("CACHE_TTL", 60*time.Second),
		AllowedOrigins:  splitCSV(get("ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")),
		LogLevel:        get("LOG_LEVEL", "info"),
		MaxBodyBytes:    int64Int("MAX_BODY_BYTES", 1<<20),
		APIKey:          strings.TrimSpace(os.Getenv("API_KEY")),
		RateLimitRPS:    float64Val("RATE_LIMIT_RPS", 5),
		RateLimitBurst:  intVal("RATE_LIMIT_BURST", 20),
		IncidentLogPath: get("INCIDENT_LOG_PATH", "data/incidents.jsonl"),
	}
	cfg.LLMEnabled = cfg.LLMAPIKey != ""

	if cfg.OMBaseURL == "" {
		return cfg, errors.New("OM_BASE_URL is required")
	}
	if cfg.OMToken == "" && (cfg.OMEmail == "" || cfg.OMPassword == "") {
		return cfg, errors.New("provide OM_TOKEN or OM_EMAIL+OM_PASSWORD")
	}
	if cfg.RateLimitRPS <= 0 {
		return cfg, errors.New("RATE_LIMIT_RPS must be > 0")
	}
	if cfg.RateLimitBurst <= 0 {
		return cfg, errors.New("RATE_LIMIT_BURST must be > 0")
	}
	for _, origin := range cfg.AllowedOrigins {
		if origin == "*" {
			return cfg, errors.New("ALLOWED_ORIGINS cannot contain '*' when credentials are enabled")
		}
	}
	return cfg, nil
}

func get(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func duration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func int64Int(key string, def int64) int64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err == nil && n > 0 {
		return n
	}
	return def
}

func intVal(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err == nil && n > 0 {
		return n
	}
	return def
}

func float64Val(key string, def float64) float64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.ParseFloat(v, 64)
	if err == nil && n > 0 {
		return n
	}
	return def
}
