package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

func TestRealAccessPolicyHandlerAuthAndKillSwitch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	policyRepo := service.NewMemoryProviderRealAccessPolicyRepo(&service.ProviderRealAccessPolicy{
		ID: 1, Name: "default", Enabled: false, GlobalKillSwitch: false, AllowMember: true,
		ImageDailyCNY: decimal.NewFromInt(10), VideoDailyCNY: decimal.NewFromInt(10), MonthlyCNY: decimal.NewFromInt(100),
	})
	// Video repo is unused for kill-switch-on path (SaveRealAccessPolicy skips formal check when kill switch is on).
	svc := service.NewVideoGatewayService(nil, nil, &config.Config{})
	svc.SetRealAccessPolicyRepository(policyRepo)
	h := NewRealReviewSessionHandler(svc)

	r := gin.New()
	r.GET("/admin/real-access-policy", func(c *gin.Context) {
		h.GetRealAccessPolicy(c)
	})
	r.GET("/admin/real-access-policy/authed", func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1})
		h.GetRealAccessPolicy(c)
	})
	r.POST("/admin/real-access-policy/kill-switch", func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1})
		h.PutKillSwitch(c)
	})

	unauth := httptest.NewRequest(http.MethodGet, "/admin/real-access-policy", nil)
	unauthRec := httptest.NewRecorder()
	r.ServeHTTP(unauthRec, unauth)
	if unauthRec.Code == http.StatusOK {
		t.Fatal("expected unauthorized without auth subject")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/admin/real-access-policy/authed", nil)
	getRec := httptest.NewRecorder()
	r.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET policy status=%d body=%s", getRec.Code, getRec.Body.String())
	}

	ksBody := `{"enabled":true}`
	ksReq := httptest.NewRequest(http.MethodPost, "/admin/real-access-policy/kill-switch", bytes.NewBufferString(ksBody))
	ksReq.Header.Set("Content-Type", "application/json")
	ksRec := httptest.NewRecorder()
	r.ServeHTTP(ksRec, ksReq)
	if ksRec.Code != http.StatusOK {
		t.Fatalf("kill-switch status=%d body=%s", ksRec.Code, ksRec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(ksRec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	data, _ := payload["data"].(map[string]any)
	if data == nil || data["global_kill_switch"] != true {
		t.Fatalf("expected kill switch true, got %#v", payload)
	}
}

func TestRealAccessPolicyHandlerClearBootstrapUnauth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewRealReviewSessionHandler(nil)
	r := gin.New()
	r.POST("/admin/real-review-session/clear-bootstrap", h.ClearReviewOnlyAccounts)
	req := httptest.NewRequest(http.MethodPost, "/admin/real-review-session/clear-bootstrap", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatal("expected unauthorized")
	}
}
