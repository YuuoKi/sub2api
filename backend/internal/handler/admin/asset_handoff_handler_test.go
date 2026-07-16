//go:build unit

package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeAssetHandoffManager struct {
	issuedIssuerID int64
	issuedTaskID   int64
	issuedKind     service.AssetHandoffKind
	consumeTicket  string
	consumeCalls   int
}

func (f *fakeAssetHandoffManager) Issue(_ context.Context, issuerID, taskID int64, kind service.AssetHandoffKind) (*service.IssuedAssetHandoff, error) {
	f.issuedIssuerID, f.issuedTaskID, f.issuedKind = issuerID, taskID, kind
	return &service.IssuedAssetHandoff{
		Ticket:       "opaque-ticket-only",
		SourceTaskID: taskID,
		AssetKind:    kind,
		ExpiresAt:    time.Date(2026, 7, 16, 12, 5, 0, 0, time.UTC),
	}, nil
}

func (f *fakeAssetHandoffManager) Consume(_ context.Context, ticket string) (*service.ConsumedAssetHandoff, error) {
	f.consumeCalls++
	f.consumeTicket = ticket
	return &service.ConsumedAssetHandoff{
		SourceTaskID: 501,
		AssetKind:    service.AssetHandoffVideo,
		URL:          "https://assets.example.test/result.mp4",
		MIME:         "video/mp4",
		SizeBytes:    2048,
		ExpiresAt:    time.Date(2026, 7, 16, 12, 5, 0, 0, time.UTC),
	}, nil
}

func TestVideoHandlerCreatesAssetHandoffForAuthenticatedAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := &fakeAssetHandoffManager{}
	handler := &VideoHandler{handoff: manager}
	router := gin.New()
	router.POST("/admin/video/tasks/:id/asset-handoffs", func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 9})
		c.Set(string(middleware.ContextKeyUserRole), service.RoleAdmin)
		handler.CreateAssetHandoff(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/video/tasks/501/asset-handoffs", bytes.NewBufferString(`{"asset_kind":"video"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, int64(9), manager.issuedIssuerID)
	require.Equal(t, int64(501), manager.issuedTaskID)
	require.Equal(t, service.AssetHandoffVideo, manager.issuedKind)
	require.NotContains(t, w.Body.String(), "assets.example.test")
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "opaque-ticket-only", body["data"].(map[string]any)["ticket"])
}

func TestVideoHandlerRejectsAssetHandoffWithoutIssuerIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := &fakeAssetHandoffManager{}
	handler := &VideoHandler{handoff: manager}
	router := gin.New()
	router.POST("/admin/video/tasks/:id/asset-handoffs", handler.CreateAssetHandoff)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/video/tasks/501/asset-handoffs", bytes.NewBufferString(`{"asset_kind":"video"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Zero(t, manager.issuedIssuerID)
}

func TestVideoHandlerConsumesTicketOnlyFromTrustedLoopbackBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct {
		name        string
		remoteAddr  string
		host        string
		trustBridge bool
		wantStatus  int
		wantCalls   int
	}{
		{name: "IPv4 loopback range", remoteAddr: "127.8.9.10:41000", host: "127.0.0.1:8080", wantStatus: http.StatusOK, wantCalls: 1},
		{name: "IPv6 loopback", remoteAddr: "[::1]:41000", host: "[::1]:8080", wantStatus: http.StatusOK, wantCalls: 1},
		{name: "explicit Docker bridge for canonical loopback host", remoteAddr: "172.18.0.1:41000", host: "127.0.0.1:8080", trustBridge: true, wantStatus: http.StatusOK, wantCalls: 1},
		{name: "Docker bridge disabled by default", remoteAddr: "172.18.0.1:41000", host: "127.0.0.1:8080", wantStatus: http.StatusForbidden, wantCalls: 0},
		{name: "Docker bridge cannot target a non-loopback host", remoteAddr: "172.18.0.1:41000", host: "sub2api:8080", trustBridge: true, wantStatus: http.StatusForbidden, wantCalls: 0},
		{name: "public remote remains forbidden", remoteAddr: "198.51.100.20:41000", host: "127.0.0.1:8080", trustBridge: true, wantStatus: http.StatusForbidden, wantCalls: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			manager := &fakeAssetHandoffManager{}
			handler := &VideoHandler{handoff: manager, trustDockerLoopbackBridge: test.trustBridge}
			router := gin.New()
			router.POST("/public/asset-handoffs/consume", handler.ConsumeAssetHandoff)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/public/asset-handoffs/consume", bytes.NewBufferString(`{"ticket":"opaque-ticket-only"}`))
			req.Header.Set("Content-Type", "application/json")
			req.RemoteAddr = test.remoteAddr
			req.Host = test.host
			req.Header.Set("X-Forwarded-For", "127.0.0.1")
			router.ServeHTTP(w, req)

			require.Equal(t, test.wantStatus, w.Code)
			require.Equal(t, test.wantCalls, manager.consumeCalls)
			if test.wantCalls == 1 {
				require.Equal(t, "opaque-ticket-only", manager.consumeTicket)
				var body map[string]any
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
				data := body["data"].(map[string]any)
				require.Equal(t, "https://assets.example.test/result.mp4", data["asset_url"])
				require.Equal(t, "video/mp4", data["content_type"])
				require.NotEmpty(t, data["expires_at"])
				require.NotContains(t, data, "url")
				require.NotContains(t, data, "mime")
			}
		})
	}
}
