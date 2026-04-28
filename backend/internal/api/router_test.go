package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/bansalbhunesh/Datastory/backend/internal/logging"
)

// SPA handler must serve real files within distDir, fall back to index.html
// for SPA routes, and refuse traversal escapes (../).
func TestSPAHandler(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<!doctype html>"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "app.js"), []byte("console.log(1)"), 0o644); err != nil {
		t.Fatalf("write js: %v", err)
	}
	// Sensitive sibling file we should never reach via traversal.
	parent := filepath.Dir(dir)
	secretPath := filepath.Join(parent, "secret.txt")
	if err := os.WriteFile(secretPath, []byte("super-secret"), 0o644); err != nil {
		t.Fatalf("write secret: %v", err)
	}
	defer os.Remove(secretPath)

	router := NewRouter(&Handlers{log: logging.New("error")}, logging.New("error"), RouterConfig{
		AllowedOrigins: []string{"http://localhost:5173"},
		MaxBodyBytes:   1 << 16,
		RateLimitRPS:   100,
		RateLimitBurst: 100,
		FrontendDist:   dir,
	})

	cases := []struct {
		name        string
		path        string
		wantStatus  int
		wantBody    string // substring; "" = don't check body
		mustNotLeak bool   // if true, test only that body != "super-secret"
	}{
		{"serves index", "/", http.StatusOK, "<!doctype html>", false},
		{"serves js asset", "/app.js", http.StatusOK, "console.log", false},
		{"spa fallback for unknown route", "/some/spa/route", http.StatusOK, "<!doctype html>", false},
		// Gin normalizes ".." segments at the router level before our handler sees
		// them — a 400 here is correct/safe. We only assert that the secret never
		// makes it into the response body.
		{"traversal attempt does not leak secret", "/../secret.txt", -1, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if tc.wantStatus != -1 && rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantBody != "" {
				if got := rec.Body.String(); !contains(got, tc.wantBody) {
					t.Fatalf("body missing %q: got %q", tc.wantBody, got)
				}
			}
			if tc.mustNotLeak && contains(rec.Body.String(), "super-secret") {
				t.Fatalf("path traversal leaked sibling file: %q", rec.Body.String())
			}
		})
	}
}

// /healthz should return 200 with no auth, no rate-limit interaction.
func TestRouter_HealthZ(t *testing.T) {
	router := NewRouter(&Handlers{log: logging.New("error")}, logging.New("error"), RouterConfig{
		AllowedOrigins: []string{"http://localhost:5173"},
		MaxBodyBytes:   1 << 16,
		APIKey:         "should-not-block-healthz",
		RateLimitRPS:   100,
		RateLimitBurst: 100,
	})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("/healthz expected 200, got %d", rec.Code)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (indexOf(s, sub) >= 0))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
