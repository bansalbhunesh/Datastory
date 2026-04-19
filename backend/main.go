package main

import (
	"context"
	"log"
	"time"

	"github.com/bansalbhunesh/Datastory/backend/config"
	"github.com/bansalbhunesh/Datastory/backend/handlers"
	"github.com/bansalbhunesh/Datastory/backend/services"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	om := services.NewOpenMetadataClient(cfg.OMBaseURL, cfg.OMToken)
	if cfg.OMToken == "" && cfg.OMEmail != "" && cfg.OMPassword != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		if _, err := om.Login(ctx, cfg.OMEmail, cfg.OMPassword); err != nil {
			log.Printf("OpenMetadata login failed (will retry on first request if needed): %v", err)
		}
		cancel()
	}

	claude := services.NewClaudeClient(cfg.ClaudeAPIKey, cfg.ClaudeModel)
	report := handlers.NewReportHandler(cfg, om, claude)

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/healthz", func(c *gin.Context) { c.Status(200) })
	r.GET("/api/health", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	r.POST("/api/generate-report", report.GenerateReport)
	r.GET("/api/debug/lineage", report.DebugLineage)

	addr := ":" + cfg.Port
	log.Printf("listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
