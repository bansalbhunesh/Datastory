package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/bansalbhunesh/Datastory/backend/internal/logging"
)

func TestGenerateReport_RejectsOverlongQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := testHandlers()

	body := `{"query":"` + strings.Repeat("a", maxQueryLen+1) + `"}`
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/generate-report", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.GenerateReport(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGenerateReport_RejectsOverlongFQN(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := testHandlers()

	body := `{"tableFQN":"` + strings.Repeat("a", maxFQNLen+1) + `"}`
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/generate-report", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.GenerateReport(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

// Bad-JSON payload should produce a generic 400 message — not echo the
// underlying parser error string, which can leak internals.
func TestGenerateReport_SanitizesJSONErrorMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := testHandlers()

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/generate-report", strings.NewReader("{not-json"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request = c.Request.WithContext(logging.WithRequestID(c.Request.Context(), "req-bad-json"))

	h.GenerateReport(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	msg, _ := body["error"].(string)
	if msg == "" || strings.Contains(msg, "unexpected token") || strings.Contains(msg, "looking for beginning") {
		t.Fatalf("error message should be generic, got %q", msg)
	}
}

func TestListIncidents_RequiresTableFQN(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := testHandlers()

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/incidents", nil)
	h.ListIncidents(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
