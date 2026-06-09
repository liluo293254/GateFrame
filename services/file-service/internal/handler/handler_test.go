package handler

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gateframe/file-service/internal/validation"
	"github.com/gin-gonic/gin"
)

func TestCreateRejectsExeUpload(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	h := New(nil, nil, validation.DefaultMaxFileBytes)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", "00000000-0000-0000-0000-000000000001")
		c.Next()
	})
	r.POST("/api/files", h.Create)

	body := `{"filename":"malware.exe","content_type":"application/octet-stream","content_base64":"` +
		base64.StdEncoding.EncodeToString([]byte("MZ")) + `"}`

	req := httptest.NewRequest(http.MethodPost, "/api/files", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "file type not allowed") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}
