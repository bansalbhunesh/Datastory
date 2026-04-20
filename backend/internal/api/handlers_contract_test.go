package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/bansalbhunesh/Datastory/backend/internal/errs"
	"github.com/bansalbhunesh/Datastory/backend/internal/logging"
)

func testHandlers() *Handlers {
	return &Handlers{log: logging.New("error")}
}

func TestGenerateReport_RejectsEmptyPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := testHandlers()

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/generate-report", bytes.NewBufferString(`{"query":"","tableFQN":""}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.GenerateReport(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	msg, _ := body["error"].(string)
	if !strings.Contains(msg, "provide 'query' or 'tableFQN'") {
		t.Fatalf("unexpected error message: %q", msg)
	}
}

func TestSearchTables_RejectsOverlongQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := testHandlers()

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/search/tables?q="+strings.Repeat("a", maxQueryLen+1), nil)

	h.SearchTables(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestRespondError_SanitizesInternalErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := testHandlers()

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/api/anything", nil)
	req = req.WithContext(logging.WithRequestID(req.Context(), "req-test-1"))
	c.Request = req

	h.respondError(c, errs.Internal("sensitive internal details", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if body["error"] != "internal server error" {
		t.Fatalf("unexpected error body: %v", body["error"])
	}
	if body["request_id"] != "req-test-1" {
		t.Fatalf("missing/invalid request_id: %v", body["request_id"])
	}
}
