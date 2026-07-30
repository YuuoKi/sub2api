package middleware

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// BackendModeUserGuard blocks non-admin users from accessing user routes when backend mode is enabled.
// Must be placed AFTER JWT auth middleware so that the user role is available in context.
func BackendModeUserGuard(settingService *service.SettingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if settingService == nil || !settingService.IsBackendModeEnabled(c.Request.Context()) {
			c.Next()
			return
		}
		role, _ := GetUserRoleFromContext(c)
		if role == "admin" {
			c.Next()
			return
		}
		response.Forbidden(c, "Backend mode is active. User self-service is disabled.")
		c.Abort()
	}
}

func backendModeAllowsAuthPath(path string) bool {
	path = strings.ToLower(strings.TrimSpace(path))
	for _, suffix := range []string{"/auth/login", "/auth/login/2fa", "/auth/logout", "/auth/refresh"} {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return false
}

// BackendModeAuthGuard selectively blocks auth endpoints when backend mode is enabled.
// Only the password/TOTP login session endpoints required by administrators remain reachable.
func BackendModeAuthGuard(settingService *service.SettingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if settingService == nil || !settingService.IsBackendModeEnabled(c.Request.Context()) {
			c.Next()
			return
		}
		if backendModeAllowsAuthPath(c.Request.URL.Path) {
			c.Next()
			return
		}
		response.Forbidden(c, "Backend mode is active. Registration and self-service auth flows are disabled.")
		c.Abort()
	}
}

var backendModeAllowedAuthRequests = map[string]string{
	"/api/v1/auth/login":               "POST",
	"/api/v1/auth/login/2fa":           "POST",
	"/api/v1/auth/logout":              "POST",
	"/api/v1/auth/refresh":             "POST",
	"/api/v1/auth/me":                  "GET",
	"/api/v1/auth/revoke-all-sessions": "POST",
}

type backendModeRequestRule struct {
	method  string
	pattern string
}

// backendModeAllowedAdminRequests is the release contract for the five LAN
// administrator capabilities. New /api/v1/admin routes are denied until they
// are deliberately added here with an HTTP method and a bounded path pattern.
var backendModeAllowedAdminRequests = []backendModeRequestRule{
	// Administrator session/compliance and read-only deployment settings.
	{http.MethodGet, "/api/v1/admin/compliance"},
	{http.MethodPost, "/api/v1/admin/compliance/accept"},
	{http.MethodGet, "/api/v1/admin/settings"},

	// Overview and cost. Budget/quota mutation is intentionally absent.
	{http.MethodGet, "/api/v1/admin/dashboard/stats"},
	{http.MethodGet, "/api/v1/admin/dashboard/snapshot-v2"},
	{http.MethodGet, "/api/v1/admin/dashboard/realtime"},
	{http.MethodGet, "/api/v1/admin/dashboard/trend"},
	{http.MethodGet, "/api/v1/admin/dashboard/models"},
	{http.MethodGet, "/api/v1/admin/dashboard/groups"},
	{http.MethodGet, "/api/v1/admin/dashboard/api-keys-trend"},
	{http.MethodGet, "/api/v1/admin/dashboard/users-trend"},
	{http.MethodGet, "/api/v1/admin/dashboard/users-ranking"},
	{http.MethodGet, "/api/v1/admin/dashboard/user-breakdown"},
	{http.MethodPost, "/api/v1/admin/dashboard/users-usage"},
	{http.MethodPost, "/api/v1/admin/dashboard/api-keys-usage"},

	// Service identities, independent administrators and API cards. Human
	// employee creation and local quota are enforced in the service layer.
	{http.MethodGet, "/api/v1/admin/users"},
	{http.MethodPost, "/api/v1/admin/users"},
	{http.MethodGet, "/api/v1/admin/users/:id"},
	{http.MethodPut, "/api/v1/admin/users/:id"},
	{http.MethodDelete, "/api/v1/admin/users/:id"},
	{http.MethodGet, "/api/v1/admin/users/:id/api-keys"},
	{http.MethodPost, "/api/v1/admin/users/:id/api-keys"},
	{http.MethodPost, "/api/v1/admin/users/:id/qcanvas-key-pair"},
	{http.MethodPost, "/api/v1/admin/users/:id/balance"},
	{http.MethodGet, "/api/v1/admin/api-keys/:id/reveal"},
	{http.MethodPut, "/api/v1/admin/api-keys/:id"},
	{http.MethodDelete, "/api/v1/admin/api-keys/:id"},

	// Routing groups and pricing.
	{http.MethodGet, "/api/v1/admin/groups"},
	{http.MethodPost, "/api/v1/admin/groups"},
	{http.MethodGet, "/api/v1/admin/groups/all"},
	{http.MethodGet, "/api/v1/admin/groups/usage-summary"},
	{http.MethodGet, "/api/v1/admin/groups/capacity-summary"},
	{http.MethodPut, "/api/v1/admin/groups/sort-order"},
	{http.MethodGet, "/api/v1/admin/groups/:id"},
	{http.MethodPut, "/api/v1/admin/groups/:id"},
	{http.MethodDelete, "/api/v1/admin/groups/:id"},
	{http.MethodGet, "/api/v1/admin/groups/:id/stats"},
	{http.MethodGet, "/api/v1/admin/groups/:id/api-keys"},
	{http.MethodGet, "/api/v1/admin/groups/:id/models-list-candidates"},
	{http.MethodGet, "/api/v1/admin/groups/:id/rate-multipliers"},
	{http.MethodPut, "/api/v1/admin/groups/:id/rate-multipliers"},
	{http.MethodDelete, "/api/v1/admin/groups/:id/rate-multipliers"},

	// Upstream accounts. Scheduled tests, review/mock and smoke routes are not
	// included. Data import/export is retained for operator-managed credentials.
	{http.MethodGet, "/api/v1/admin/accounts"},
	{http.MethodPost, "/api/v1/admin/accounts"},
	{http.MethodGet, "/api/v1/admin/accounts/data"},
	{http.MethodPost, "/api/v1/admin/accounts/data"},
	{http.MethodPost, "/api/v1/admin/accounts/batch"},
	{http.MethodPost, "/api/v1/admin/accounts/batch-update-credentials"},
	{http.MethodPost, "/api/v1/admin/accounts/batch-refresh-tier"},
	{http.MethodPost, "/api/v1/admin/accounts/batch-clear-error"},
	{http.MethodPost, "/api/v1/admin/accounts/batch-refresh"},
	{http.MethodPost, "/api/v1/admin/accounts/bulk-update"},
	{http.MethodPost, "/api/v1/admin/accounts/check-mixed-channel"},
	{http.MethodPost, "/api/v1/admin/accounts/import/codex-session"},
	{http.MethodPost, "/api/v1/admin/accounts/sync/crs"},
	{http.MethodPost, "/api/v1/admin/accounts/sync/crs/preview"},
	{http.MethodPost, "/api/v1/admin/accounts/today-stats/batch"},
	{http.MethodPost, "/api/v1/admin/accounts/models/sync-upstream-preview"},
	{http.MethodGet, "/api/v1/admin/accounts/antigravity/default-model-mapping"},
	{http.MethodPost, "/api/v1/admin/accounts/generate-auth-url"},
	{http.MethodPost, "/api/v1/admin/accounts/generate-setup-token-url"},
	{http.MethodPost, "/api/v1/admin/accounts/exchange-code"},
	{http.MethodPost, "/api/v1/admin/accounts/exchange-setup-token-code"},
	{http.MethodPost, "/api/v1/admin/accounts/cookie-auth"},
	{http.MethodPost, "/api/v1/admin/accounts/setup-token-cookie-auth"},
	{http.MethodGet, "/api/v1/admin/accounts/:id"},
	{http.MethodPut, "/api/v1/admin/accounts/:id"},
	{http.MethodDelete, "/api/v1/admin/accounts/:id"},
	{http.MethodPost, "/api/v1/admin/accounts/:id/connectivity-check"},
	{http.MethodGet, "/api/v1/admin/accounts/:id/stats"},
	{http.MethodGet, "/api/v1/admin/accounts/:id/usage"},
	{http.MethodGet, "/api/v1/admin/accounts/:id/today-stats"},
	{http.MethodGet, "/api/v1/admin/accounts/:id/models"},
	{http.MethodGet, "/api/v1/admin/accounts/:id/temp-unschedulable"},
	{http.MethodDelete, "/api/v1/admin/accounts/:id/temp-unschedulable"},
	{http.MethodPost, "/api/v1/admin/accounts/:id/test"},
	{http.MethodPost, "/api/v1/admin/accounts/:id/refresh"},
	{http.MethodPost, "/api/v1/admin/accounts/:id/recover-state"},
	{http.MethodPost, "/api/v1/admin/accounts/:id/clear-error"},
	{http.MethodPost, "/api/v1/admin/accounts/:id/clear-rate-limit"},
	{http.MethodPost, "/api/v1/admin/accounts/:id/schedulable"},
	{http.MethodPost, "/api/v1/admin/accounts/:id/set-privacy"},
	{http.MethodPost, "/api/v1/admin/accounts/:id/refresh-tier"},
	{http.MethodPost, "/api/v1/admin/accounts/:id/apply-oauth-credentials"},
	{http.MethodPost, "/api/v1/admin/accounts/:id/revert-proxy-fallback"},
	{http.MethodPost, "/api/v1/admin/accounts/:id/shadow"},
	{http.MethodPost, "/api/v1/admin/accounts/:id/models/sync-upstream"},

	// Provider OAuth is part of upstream-account administration. Keep every
	// operation explicit so new commercial or test endpoints stay denied by
	// default.
	{http.MethodPost, "/api/v1/admin/openai/generate-auth-url"},
	{http.MethodPost, "/api/v1/admin/openai/exchange-code"},
	{http.MethodPost, "/api/v1/admin/openai/refresh-token"},
	{http.MethodPost, "/api/v1/admin/openai/accounts/:id/refresh"},
	{http.MethodPost, "/api/v1/admin/openai/create-from-oauth"},
	{http.MethodPost, "/api/v1/admin/openai/create-from-codex-pat"},
	{http.MethodGet, "/api/v1/admin/openai/accounts/:id/quota"},
	{http.MethodPost, "/api/v1/admin/openai/accounts/:id/reset-quota"},
	{http.MethodPost, "/api/v1/admin/gemini/oauth/auth-url"},
	{http.MethodPost, "/api/v1/admin/gemini/oauth/exchange-code"},
	{http.MethodGet, "/api/v1/admin/gemini/oauth/capabilities"},
	{http.MethodPost, "/api/v1/admin/antigravity/oauth/auth-url"},
	{http.MethodPost, "/api/v1/admin/antigravity/oauth/exchange-code"},
	{http.MethodPost, "/api/v1/admin/antigravity/oauth/refresh-token"},
	{http.MethodPost, "/api/v1/admin/grok/oauth/auth-url"},
	{http.MethodPost, "/api/v1/admin/grok/oauth/exchange-code"},
	{http.MethodPost, "/api/v1/admin/grok/oauth/refresh-token"},
	{http.MethodPost, "/api/v1/admin/grok/oauth/create-from-oauth"},
	{http.MethodPost, "/api/v1/admin/grok/accounts/:id/refresh"},
	{http.MethodGet, "/api/v1/admin/grok/accounts/:id/quota"},
	{http.MethodPost, "/api/v1/admin/grok/accounts/:id/reset-quota"},
	{http.MethodGet, "/api/v1/admin/grok/runtime-sanity"},

	// Channels, monitors and proxies.
	{http.MethodGet, "/api/v1/admin/channels"},
	{http.MethodPost, "/api/v1/admin/channels"},
	{http.MethodGet, "/api/v1/admin/channels/model-pricing"},
	{http.MethodGet, "/api/v1/admin/channels/pricing/sync-models"},
	{http.MethodGet, "/api/v1/admin/channels/:id"},
	{http.MethodPut, "/api/v1/admin/channels/:id"},
	{http.MethodDelete, "/api/v1/admin/channels/:id"},
	{http.MethodGet, "/api/v1/admin/channel-monitors"},
	{http.MethodPost, "/api/v1/admin/channel-monitors"},
	{http.MethodGet, "/api/v1/admin/channel-monitors/:id"},
	{http.MethodPut, "/api/v1/admin/channel-monitors/:id"},
	{http.MethodDelete, "/api/v1/admin/channel-monitors/:id"},
	{http.MethodPost, "/api/v1/admin/channel-monitors/:id/run"},
	{http.MethodGet, "/api/v1/admin/channel-monitors/:id/history"},
	{http.MethodGet, "/api/v1/admin/channel-monitor-templates"},
	{http.MethodPost, "/api/v1/admin/channel-monitor-templates"},
	{http.MethodGet, "/api/v1/admin/channel-monitor-templates/:id"},
	{http.MethodPut, "/api/v1/admin/channel-monitor-templates/:id"},
	{http.MethodDelete, "/api/v1/admin/channel-monitor-templates/:id"},
	{http.MethodGet, "/api/v1/admin/channel-monitor-templates/:id/monitors"},
	{http.MethodPost, "/api/v1/admin/channel-monitor-templates/:id/apply"},
	{http.MethodGet, "/api/v1/admin/proxies"},
	{http.MethodPost, "/api/v1/admin/proxies"},
	{http.MethodGet, "/api/v1/admin/proxies/all"},
	{http.MethodGet, "/api/v1/admin/proxies/data"},
	{http.MethodPost, "/api/v1/admin/proxies/data"},
	{http.MethodPost, "/api/v1/admin/proxies/batch"},
	{http.MethodPost, "/api/v1/admin/proxies/batch-delete"},
	{http.MethodGet, "/api/v1/admin/proxies/:id"},
	{http.MethodPut, "/api/v1/admin/proxies/:id"},
	{http.MethodDelete, "/api/v1/admin/proxies/:id"},
	{http.MethodPost, "/api/v1/admin/proxies/:id/test"},
	{http.MethodPost, "/api/v1/admin/proxies/:id/quality-check"},
	{http.MethodGet, "/api/v1/admin/proxies/:id/stats"},
	{http.MethodGet, "/api/v1/admin/proxies/:id/accounts"},

	// Video provider configuration, task evidence and persistent assets. The
	// legacy tiny-real authorization and asset-handoff paths are absent.
	{http.MethodGet, "/api/v1/admin/video/contract"},
	{http.MethodGet, "/api/v1/admin/video/providers"},
	{http.MethodPost, "/api/v1/admin/video/providers"},
	{http.MethodPut, "/api/v1/admin/video/providers/:id"},
	{http.MethodDelete, "/api/v1/admin/video/providers/:id"},
	{http.MethodPost, "/api/v1/admin/video/providers/:id/connectivity-check"},
	{http.MethodGet, "/api/v1/admin/video/tasks"},
	{http.MethodGet, "/api/v1/admin/video/tasks/:id"},
	{http.MethodGet, "/api/v1/admin/video/tasks/:id/local-asset"},
	{http.MethodGet, "/api/v1/admin/video/system-check"},

	// Call, asset and usage evidence.
	{http.MethodGet, "/api/v1/admin/generation-content/stats"},
	{http.MethodGet, "/api/v1/admin/generation-content/samples"},
	{http.MethodGet, "/api/v1/admin/generation-content/weekly-report"},
	{http.MethodGet, "/api/v1/admin/usage"},
	{http.MethodGet, "/api/v1/admin/usage/stats"},
	{http.MethodGet, "/api/v1/admin/usage/search-users"},
	{http.MethodGet, "/api/v1/admin/usage/search-api-keys"},

	// Formal backup/restore surface.
	{http.MethodGet, "/api/v1/admin/backups"},
	{http.MethodPost, "/api/v1/admin/backups"},
	{http.MethodGet, "/api/v1/admin/backups/s3-config"},
	{http.MethodPut, "/api/v1/admin/backups/s3-config"},
	{http.MethodPost, "/api/v1/admin/backups/s3-config/test"},
	{http.MethodGet, "/api/v1/admin/backups/schedule"},
	{http.MethodPut, "/api/v1/admin/backups/schedule"},
	{http.MethodGet, "/api/v1/admin/backups/:id"},
	{http.MethodDelete, "/api/v1/admin/backups/:id"},
	{http.MethodGet, "/api/v1/admin/backups/:id/download-url"},
	{http.MethodPost, "/api/v1/admin/backups/:id/restore"},

	// Health/operations are read-only except bounded acknowledgement and runtime
	// maintenance operations that are already present in the retained UI.
	{http.MethodGet, "/api/v1/admin/ops/concurrency"},
	{http.MethodGet, "/api/v1/admin/ops/user-concurrency"},
	{http.MethodGet, "/api/v1/admin/ops/account-availability"},
	{http.MethodGet, "/api/v1/admin/ops/realtime-traffic"},
	{http.MethodGet, "/api/v1/admin/ops/alert-rules"},
	{http.MethodGet, "/api/v1/admin/ops/alert-events"},
	{http.MethodGet, "/api/v1/admin/ops/alert-events/:id"},
	{http.MethodGet, "/api/v1/admin/ops/email-notification/config"},
	{http.MethodGet, "/api/v1/admin/ops/runtime/alert"},
	{http.MethodGet, "/api/v1/admin/ops/runtime/logging"},
	{http.MethodGet, "/api/v1/admin/ops/advanced-settings"},
	{http.MethodGet, "/api/v1/admin/ops/settings/metric-thresholds"},
	{http.MethodGet, "/api/v1/admin/ops/ws/qps"},
	{http.MethodGet, "/api/v1/admin/ops/errors"},
	{http.MethodGet, "/api/v1/admin/ops/errors/:id"},
	{http.MethodGet, "/api/v1/admin/ops/request-errors"},
	{http.MethodGet, "/api/v1/admin/ops/request-errors/:id"},
	{http.MethodGet, "/api/v1/admin/ops/request-errors/:id/upstream-errors"},
	{http.MethodGet, "/api/v1/admin/ops/upstream-errors"},
	{http.MethodGet, "/api/v1/admin/ops/upstream-errors/:id"},
	{http.MethodGet, "/api/v1/admin/ops/requests"},
	{http.MethodGet, "/api/v1/admin/ops/system-logs"},
	{http.MethodGet, "/api/v1/admin/ops/system-logs/health"},
	{http.MethodGet, "/api/v1/admin/ops/dashboard/snapshot-v2"},
	{http.MethodGet, "/api/v1/admin/ops/dashboard/overview"},
	{http.MethodGet, "/api/v1/admin/ops/dashboard/throughput-trend"},
	{http.MethodGet, "/api/v1/admin/ops/dashboard/latency-histogram"},
	{http.MethodGet, "/api/v1/admin/ops/dashboard/error-trend"},
	{http.MethodGet, "/api/v1/admin/ops/dashboard/error-distribution"},
	{http.MethodGet, "/api/v1/admin/ops/dashboard/openai-token-stats"},
	{http.MethodPut, "/api/v1/admin/ops/alert-events/:id/status"},
	{http.MethodPut, "/api/v1/admin/ops/errors/:id/resolve"},
	{http.MethodPut, "/api/v1/admin/ops/request-errors/:id/resolve"},
	{http.MethodPut, "/api/v1/admin/ops/upstream-errors/:id/resolve"},
	{http.MethodPost, "/api/v1/admin/ops/alert-rules"},
	{http.MethodPut, "/api/v1/admin/ops/alert-rules/:id"},
	{http.MethodDelete, "/api/v1/admin/ops/alert-rules/:id"},
	{http.MethodPost, "/api/v1/admin/ops/alert-silences"},
	{http.MethodPut, "/api/v1/admin/ops/email-notification/config"},
	{http.MethodPut, "/api/v1/admin/ops/runtime/alert"},
	{http.MethodPut, "/api/v1/admin/ops/runtime/logging"},
	{http.MethodPost, "/api/v1/admin/ops/runtime/logging/reset"},
	{http.MethodPut, "/api/v1/admin/ops/advanced-settings"},
	{http.MethodPut, "/api/v1/admin/ops/settings/metric-thresholds"},
	{http.MethodPost, "/api/v1/admin/ops/system-logs/cleanup"},
	{http.MethodGet, "/api/v1/admin/system/version"},
}

var backendModeAllowedUserRequests = map[string]string{
	"/api/v1/user/profile":                  "GET",
	"/api/v1/user":                          "PUT",
	"/api/v1/user/password":                 "PUT",
	"/api/v1/user/totp/status":              "GET",
	"/api/v1/user/totp/verification-method": "GET",
	"/api/v1/user/totp/send-code":           "POST",
	"/api/v1/user/totp/setup":               "POST",
	"/api/v1/user/totp/enable":              "POST",
	"/api/v1/user/totp/disable":             "POST",
}

func pathIsOrIsBelow(path, prefix string) bool {
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

func backendModePathMatches(pattern, path string) bool {
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	for i, part := range patternParts {
		if part == "**" {
			return i < len(pathParts)
		}
		if i >= len(pathParts) {
			return false
		}
		if strings.HasPrefix(part, ":") {
			if pathParts[i] == "" {
				return false
			}
			continue
		}
		if part != pathParts[i] {
			return false
		}
	}
	return len(patternParts) == len(pathParts)
}

func backendModeAllowsProductRequest(method, path string) bool {
	method = strings.ToUpper(strings.TrimSpace(method))
	path = strings.ToLower(strings.TrimSpace(path))

	// The administrator surface guard is mounted on /api/v1 only. API-key
	// gateways (/v1 and /v1beta) deliberately remain available to QCanvas.
	if !pathIsOrIsBelow(path, "/api/v1") {
		return true
	}

	if allowedMethod, ok := backendModeAllowedAuthRequests[path]; ok {
		return method == allowedMethod
	}
	if allowedMethod, ok := backendModeAllowedUserRequests[path]; ok {
		return method == allowedMethod
	}
	if path == "/api/v1/settings/public" {
		return method == "GET"
	}
	for _, rule := range backendModeAllowedAdminRequests {
		if method == rule.method && backendModePathMatches(rule.pattern, path) {
			return true
		}
	}
	return false
}

func backendModeAllowsProductPath(path string) bool {
	normalizedPath := strings.ToLower(strings.TrimSpace(path))
	if _, ok := backendModeAllowedAuthRequests[normalizedPath]; ok {
		return true
	}
	if _, ok := backendModeAllowedUserRequests[normalizedPath]; ok {
		return true
	}
	return backendModeAllowsProductRequest("GET", path)
}

// BackendModeProductSurfaceGuard makes legacy commercial and review-only APIs
// unreachable in the LAN administrator deployment profile. It is intentionally
// mounted only on /api/v1; API-key gateways such as /v1 and /v1beta stay outside it.
func BackendModeProductSurfaceGuard(settingService *service.SettingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if settingService == nil || !settingService.IsLANAdminProfile() {
			c.Next()
			return
		}
		if backendModeAllowsProductRequest(c.Request.Method, c.Request.URL.Path) {
			c.Next()
			return
		}
		response.Forbidden(c, "Backend mode is active. This product surface is disabled.")
		c.Abort()
	}
}
