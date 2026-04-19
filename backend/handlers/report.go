package handlers

import (
	"context"
	"log"
	"net/http"
	"strings"

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

	lineageRaw, err := h.OM.GetTableLineageByFQN(ctx, tableFQN, 2, 2)
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

	md, err := h.Claude.GenerateIncidentReportMarkdown(ctx, tableFQN, lineage, failed)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "claude failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.GenerateReportResponse{
		TableFQN:    tableFQN,
		Markdown:    md,
		Lineage:     lineage,
		FailedTests: failed,
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
	raw, err := h.OM.GetTableLineageByFQN(ctx, tableFQN, 2, 2)
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
