package api

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/bansalbhunesh/Datastory/backend/internal/api/middleware"
)

type RouterConfig struct {
	AllowedOrigins  []string
	MaxBodyBytes    int64
	APIKey          string
	RateLimitRPS    float64
	RateLimitBurst  int
	AllowCredentials bool
	FrontendDist    string // path to built frontend; empty = disabled
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
		AllowCredentials: cfg.AllowCredentials,
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
	// Serve the built React frontend (SPA) when FRONTEND_DIST is configured.
	// All API routes are already registered above, so only unmatched paths reach here.
	if cfg.FrontendDist != "" {
		r.NoRoute(spaHandler(cfg.FrontendDist))
	}

	return r
}

// spaHandler serves static files from distDir. If a file doesn't exist it
// falls back to index.html so React Router can handle client-side routes.
// Path traversal is mitigated by both filepath.Clean *and* a final
// containment check against the absolute distDir.
func spaHandler(distDir string) gin.HandlerFunc {
	absRoot, err := filepath.Abs(distDir)
	if err != nil {
		absRoot = distDir
	}
	return func(c *gin.Context) {
		clean := filepath.Join(absRoot, filepath.Clean("/"+c.Request.URL.Path))
		// Belt-and-braces: refuse anything that escapes the dist root after Clean+Join.
		if !strings.HasPrefix(clean+string(os.PathSeparator), absRoot+string(os.PathSeparator)) && clean != absRoot {
			c.Status(http.StatusNotFound)
			return
		}
		if info, err := os.Stat(clean); err == nil && !info.IsDir() {
			c.File(clean)
			return
		}
		index := filepath.Join(absRoot, "index.html")
		if _, err := os.Stat(index); err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.File(index)
	}
}
