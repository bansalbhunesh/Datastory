package api

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/bansalbhunesh/Datastory/backend/internal/clients/llm"
	"github.com/bansalbhunesh/Datastory/backend/internal/clients/openmetadata"
	"github.com/bansalbhunesh/Datastory/backend/internal/errs"
	"github.com/bansalbhunesh/Datastory/backend/internal/logging"
	"github.com/bansalbhunesh/Datastory/backend/internal/services"
)

const (
	maxQueryLen = 256
	maxFQNLen   = 512
)

// Handlers is a thin HTTP layer. No business logic lives here.
type Handlers struct {
	report     *services.ReportService
	om         *openmetadata.Client
	llm        llm.Client
	log        *slog.Logger
	genTimeout time.Duration
}

// HandlersConfig configures handler-level behaviour. Zero values fall back
// to sensible defaults so existing call sites keep working.
type HandlersConfig struct {
	GenerateTimeout time.Duration
}

func NewHandlers(r *services.ReportService, om *openmetadata.Client, l llm.Client, log *slog.Logger, cfg ...HandlersConfig) *Handlers {
	h := &Handlers{report: r, om: om, llm: l, log: log, genTimeout: 45 * time.Second}
	if len(cfg) > 0 && cfg[0].GenerateTimeout > 0 {
		h.genTimeout = cfg[0].GenerateTimeout
	}
	return h
}

// POST /api/generate-report
func (h *Handlers) GenerateReport(c *gin.Context) {
	var req GenerateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Avoid leaking decoder internals; the precise cause is in server logs.
		h.log.Warn("generate-report: invalid json", "error", err.Error())
		h.respondError(c, errs.BadRequest("invalid JSON body"))
		return
	}
	if strings.TrimSpace(req.Query) == "" && strings.TrimSpace(req.TableFQN) == "" {
		h.respondError(c, errs.BadRequest("provide 'query' or 'tableFQN'"))
		return
	}
	req.Query = strings.TrimSpace(req.Query)
	req.TableFQN = strings.TrimSpace(req.TableFQN)
	if len(req.Query) > maxQueryLen {
		h.respondError(c, errs.BadRequest("query too long"))
		return
	}
	if len(req.TableFQN) > maxFQNLen {
		h.respondError(c, errs.BadRequest("tableFQN too long"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.genTimeout)
	defer cancel()

	report, err := h.report.Generate(ctx, services.GenerateInput{
		Query: req.Query, TableFQN: req.TableFQN,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, toResponse(report))
}

// GET /api/search/tables?q=...
func (h *Handlers) SearchTables(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if len(q) > maxQueryLen {
		h.respondError(c, errs.BadRequest("q too long"))
		return
	}
	hits, err := h.report.SearchTables(c.Request.Context(), q)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"hits": hits})
}

// GET /api/ready
//
// Always returns 200 with a structured payload. The frontend renders the
// individual booleans, and a 200 here just means "the API process is up";
// it does not imply OpenMetadata is reachable. We avoid 503 so the SPA can
// still render a clear "OpenMetadata unreachable" banner instead of a
// browser-side network error.
func (h *Handlers) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	omReachable := h.om.Ping(ctx) == nil
	omAuthOK := h.om.HasStaticToken() || h.om.HasCreds()
	c.JSON(http.StatusOK, gin.H{
		"openmetadata": gin.H{"reachable": omReachable, "auth": omAuthOK},
		"claude":       gin.H{"configured": h.llm != nil && h.llm.Enabled()},
	})
}

// GET /api/debug/lineage?tableFQN=...&q=...
func (h *Handlers) DebugLineage(c *gin.Context) {
	lineage, fqn, err := h.report.Lineage(c.Request.Context(), services.GenerateInput{
		Query: c.Query("q"), TableFQN: c.Query("tableFQN"),
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"tableFQN": fqn,
		"lineage":  lineage,
	})
}

// GET /api/incidents?tableFQN=...&limit=...
func (h *Handlers) ListIncidents(c *gin.Context) {
	tableFQN := strings.TrimSpace(c.Query("tableFQN"))
	if tableFQN == "" {
		h.respondError(c, errs.BadRequest("tableFQN required"))
		return
	}
	limit := 20
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	entries, err := h.report.ListIncidents(tableFQN, limit)
	if err != nil {
		h.respondError(c, errs.Internal("failed to list incidents", err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"incidents": entries})
}

func (h *Handlers) respondError(c *gin.Context, err error) {
	status := errs.HTTPStatus(err)
	// Log at appropriate level.
	log := logging.FromCtx(c.Request.Context(), h.log)
	requestID := logging.RequestID(c.Request.Context())
	if status >= 500 {
		log.Error("request failed", "error", err.Error(), "status", status, "request_id", requestID)
	} else {
		log.Warn("request rejected", "error", err.Error(), "status", status, "request_id", requestID)
	}
	if status >= 500 {
		c.JSON(status, gin.H{
			"error":      "internal server error",
			"request_id": requestID,
		})
		return
	}
	c.JSON(status, gin.H{"error": err.Error()})
}
