package api

import (
	"log/slog"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/bansalbhunesh/Datastory/backend/internal/api/middleware"
)

type RouterConfig struct {
	AllowedOrigins []string
	MaxBodyBytes   int64
	APIKey         string
	RateLimitRPS   float64
	RateLimitBurst int
}

// NewRouter assembles all middleware + routes.
func NewRouter(h *Handlers, log *slog.Logger, cfg RouterConfig) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.Use(middleware.RequestID())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "X-Request-ID", "X-API-Key"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	r.Use(middleware.BodyLimit(cfg.MaxBodyBytes))
	r.Use(middleware.AccessLog(log))
	r.Use(middleware.Recovery(log))

	r.GET("/healthz", func(c *gin.Context) { c.Status(200) })
	r.GET("/api/health", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	api := r.Group("/api")
	api.Use(middleware.APIKey(cfg.APIKey))
	api.Use(middleware.RateLimit(cfg.RateLimitRPS, cfg.RateLimitBurst))
	{
		api.GET("/ready", h.Ready)
		api.GET("/search/tables", h.SearchTables)
		api.POST("/generate-report", h.GenerateReport)
		api.GET("/debug/lineage", h.DebugLineage)
		api.GET("/incidents", h.ListIncidents)
	}
	return r
}
