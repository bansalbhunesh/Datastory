package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	OMBaseURL    string
	OMToken      string
	OMEmail      string
	OMPassword   string
	ClaudeAPIKey string
	ClaudeModel  string
	Port         string
}

func Load() Config {
	_ = godotenv.Load("../.env")
	_ = godotenv.Load(".env")

	cfg := Config{
		OMBaseURL:    firstNonEmpty(os.Getenv("OM_BASE_URL"), os.Getenv("OM_URL"), "http://localhost:8585"),
		OMToken:      strings.TrimSpace(os.Getenv("OM_TOKEN")),
		OMEmail:      strings.TrimSpace(os.Getenv("OM_EMAIL")),
		OMPassword:   os.Getenv("OM_PASSWORD"),
		ClaudeAPIKey: strings.TrimSpace(os.Getenv("CLAUDE_API_KEY")),
		ClaudeModel:  firstNonEmpty(os.Getenv("CLAUDE_MODEL"), "claude-sonnet-4-20250514"),
		Port:         firstNonEmpty(os.Getenv("BACKEND_PORT"), "8080"),
	}
	return cfg
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
