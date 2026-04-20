package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bansalbhunesh/Datastory/backend/internal/api"
	"github.com/bansalbhunesh/Datastory/backend/internal/clients/llm"
	"github.com/bansalbhunesh/Datastory/backend/internal/clients/openmetadata"
	"github.com/bansalbhunesh/Datastory/backend/internal/config"
	"github.com/bansalbhunesh/Datastory/backend/internal/logging"
	"github.com/bansalbhunesh/Datastory/backend/internal/services"
)

func main() {
	cfg, err := config.Load()
	log := logging.New(cfg.LogLevel)
	if err != nil {
		log.Error("config load failed", "error", err.Error())
		os.Exit(1)
	}

	om := openmetadata.New(openmetadata.Options{
		BaseURL:  cfg.OMBaseURL,
		Token:    cfg.OMToken,
		Email:    cfg.OMEmail,
		Password: cfg.OMPassword,
		Timeout:  cfg.RequestTimeout,
	})

	var llmClient llm.Client
	if cfg.LLMEnabled {
		llmClient = llm.NewAnthropic(cfg.LLMAPIKey, cfg.LLMModel)
	} else {
		llmClient = disabledLLM{}
	}

	incidentStore := services.NewFileIncidentStore(cfg.IncidentLogPath)
	sqliteStore, err := services.NewSQLiteIncidentStore(cfg.IncidentLogPath)
	if err != nil {
		log.Error("failed to initialize sqlite incident store", "error", err.Error())
		os.Exit(1)
	}
	incidentStore = sqliteStore
	reportSvc := services.NewReportService(om, llmClient, log, cfg.CacheTTL, incidentStore)
	handlers := api.NewHandlers(reportSvc, om, llmClient, log)
	router := api.NewRouter(handlers, log, api.RouterConfig{
		AllowedOrigins: cfg.AllowedOrigins,
		MaxBodyBytes:   cfg.MaxBodyBytes,
		APIKey:         cfg.APIKey,
		RateLimitRPS:   cfg.RateLimitRPS,
		RateLimitBurst: cfg.RateLimitBurst,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("server starting", "addr", srv.Addr, "llm_enabled", cfg.LLMEnabled)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "error", err.Error())
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Info("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("shutdown error", "error", err.Error())
	}
	log.Info("bye")
}

// disabledLLM lets services treat the LLM as always-off uniformly.
type disabledLLM struct{}

func (disabledLLM) Rewrite(context.Context, string, string, int) (string, error) {
	return "", errors.New("llm disabled")
}
func (disabledLLM) Enabled() bool { return false }
