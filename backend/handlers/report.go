package handlers

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/bansalbhunesh/Datastory/backend/config"
	"github.com/bansalbhunesh/Datastory/backend/models"
	"github.com/bansalbhunesh/Datastory/backend/services"
	"github.com/gin-gonic/gin"
)

type ReportHandler struct {
	Cfg    config.Config
	OM     *services.OpenMetadataClient
	Claude *services.ClaudeClient
}

func NewReportHandler(cfg config.Config, om *services.OpenMetadataClient, claude *services.ClaudeClient) *ReportHandler {
	return &ReportHandler{Cfg: cfg, OM: om, Claude: claude}
}

// GenerateReport orchestrates OpenMetadata + Claude. Keep this thin: no business rules beyond sequencing.
func (h *ReportHandler) GenerateReport(c *gin.Context) {
	var req models.GenerateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json: " + err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := h.ensureOMSession(ctx); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "openmetadata auth failed: " + err.Error()})
		return
	}

	tableFQN, _, err := h.OM.ResolveTableFQN(ctx, req.TableFQN, req.Query)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lineageRaw, err := h.OM.GetTableLineageByFQN(ctx, tableFQN, 3, 3)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "lineage fetch failed: " + err.Error()})
		return
	}
	lineage, err := services.SummarizeLineage(lineageRaw)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "lineage parse failed: " + err.Error()})
		return
	}

	failed, err := h.OM.ListFailedTests(ctx, tableFQN, 50)
	if err != nil {
		// Quality is best-effort: OM versions/permissions vary.
		log.Printf("quality: non-fatal: %v", err)
		failed = nil
	}

	warnings := make([]string, 0)
	source := "deterministic"
	md := services.DraftIncidentReportMarkdown(tableFQN, lineage, failed)

	if strings.TrimSpace(h.Cfg.ClaudeAPIKey) != "" {
		mdLLM, err := h.Claude.GenerateIncidentReportMarkdown(ctx, tableFQN, lineage, failed)
		if err != nil {
			warnings = append(warnings, "Claude generation failed; returned deterministic draft instead: "+err.Error())
		} else {
			md = mdLLM
			source = "claude"
		}
	} else {
		warnings = append(warnings, "CLAUDE_API_KEY not set; returning deterministic draft (still grounded in OpenMetadata facts).")
	}

	c.JSON(http.StatusOK, models.GenerateReportResponse{
		TableFQN:    tableFQN,
		Markdown:    md,
		Lineage:     lineage,
		FailedTests: failed,
		Source:      source,
		Warnings:    warnings,
	})
}

// SearchTables exposes OpenMetadata table search for the UI autocomplete.
func (h *ReportHandler) SearchTables(c *gin.Context) {
	ctx := c.Request.Context()
	if err := h.ensureOMSession(ctx); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "openmetadata auth failed: " + err.Error()})
		return
	}

	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		c.JSON(http.StatusOK, gin.H{"hits": []services.TableSearchHit{}})
		return
	}

	raw, err := h.OM.SearchTables(ctx, q, 15)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	hits, err := services.ParseTableSearchHits(raw)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"hits": hits})
}

// Ready reports whether dependencies appear configured/reachable for the UI.
func (h *ReportHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	omReachable := h.OM.Ping(ctx) == nil
	omAuthOK := strings.TrimSpace(h.OM.Token) != "" ||
		(strings.TrimSpace(h.Cfg.OMEmail) != "" && h.Cfg.OMPassword != "")

	claudeConfigured := strings.TrimSpace(h.Cfg.ClaudeAPIKey) != ""

	c.JSON(http.StatusOK, gin.H{
		"openmetadata": gin.H{
			"reachable": omReachable,
			"auth":      omAuthOK,
		},
		"claude": gin.H{
			"configured": claudeConfigured,
		},
	})
}

// DebugLineage returns parsed lineage + raw JSON string for quick Postman-less verification.
func (h *ReportHandler) DebugLineage(c *gin.Context) {
	ctx := c.Request.Context()
	if err := h.ensureOMSession(ctx); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	q := c.Query("q")
	fqn := c.Query("tableFQN")
	tableFQN, _, err := h.OM.ResolveTableFQN(ctx, fqn, q)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	raw, err := h.OM.GetTableLineageByFQN(ctx, tableFQN, 3, 3)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	sum, err := services.SummarizeLineage(raw)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"tableFQN": tableFQN,
		"summary":  sum,
		"raw":      string(raw),
	})
}

func (h *ReportHandler) ensureOMSession(ctx context.Context) error {
	if strings.TrimSpace(h.OM.Token) != "" {
		return nil
	}
	if strings.TrimSpace(h.Cfg.OMEmail) == "" || h.Cfg.OMPassword == "" {
		return errNeedOMCreds{}
	}
	_, err := h.OM.Login(ctx, h.Cfg.OMEmail, h.Cfg.OMPassword)
	return err
}

type errNeedOMCreds struct{}

func (errNeedOMCreds) Error() string {
	return "set OM_TOKEN or OM_EMAIL + OM_PASSWORD"
}
