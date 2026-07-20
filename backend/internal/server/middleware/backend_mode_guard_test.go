//go:build unit

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type bmSettingRepo struct {
	values map[string]string
}

func (r *bmSettingRepo) Get(_ context.Context, _ string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (r *bmSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	v, ok := r.values[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return v, nil
}

func (r *bmSettingRepo) Set(_ context.Context, _, _ string) error {
	panic("unexpected Set call")
}

func (r *bmSettingRepo) GetMultiple(_ context.Context, _ []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (r *bmSettingRepo) SetMultiple(_ context.Context, settings map[string]string) error {
	if r.values == nil {
		r.values = make(map[string]string, len(settings))
	}
	for key, value := range settings {
		r.values[key] = value
	}
	return nil
}

func (r *bmSettingRepo) GetAll(_ context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (r *bmSettingRepo) Delete(_ context.Context, _ string) error {
	panic("unexpected Delete call")
}

func newBackendModeSettingService(t *testing.T, enabled string) *service.SettingService {
	t.Helper()

	repo := &bmSettingRepo{
		values: map[string]string{
			service.SettingKeyBackendModeEnabled: enabled,
		},
	}
	svc := service.NewSettingService(repo, &config.Config{})
	require.NoError(t, svc.UpdateSettings(context.Background(), &service.SystemSettings{
		BackendModeEnabled: enabled == "true",
	}))

	return svc
}

func stringPtr(v string) *string {
	return &v
}

func TestBackendModeUserGuard(t *testing.T) {
	tests := []struct {
		name       string
		nilService bool
		enabled    string
		role       *string
		wantStatus int
	}{
		{
			name:       "disabled_allows_all",
			enabled:    "false",
			role:       stringPtr("user"),
			wantStatus: http.StatusOK,
		},
		{
			name:       "nil_service_allows_all",
			nilService: true,
			role:       stringPtr("user"),
			wantStatus: http.StatusOK,
		},
		{
			name:       "enabled_admin_allowed",
			enabled:    "true",
			role:       stringPtr("admin"),
			wantStatus: http.StatusOK,
		},
		{
			name:       "enabled_user_blocked",
			enabled:    "true",
			role:       stringPtr("user"),
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_no_role_blocked",
			enabled:    "true",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_empty_role_blocked",
			enabled:    "true",
			role:       stringPtr(""),
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			r := gin.New()
			if tc.role != nil {
				role := *tc.role
				r.Use(func(c *gin.Context) {
					c.Set(string(ContextKeyUserRole), role)
					c.Next()
				})
			}

			var svc *service.SettingService
			if !tc.nilService {
				svc = newBackendModeSettingService(t, tc.enabled)
			}

			r.Use(BackendModeUserGuard(svc))
			r.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			r.ServeHTTP(w, req)

			require.Equal(t, tc.wantStatus, w.Code)
		})
	}
}

func TestBackendModeAuthGuard(t *testing.T) {
	tests := []struct {
		name       string
		nilService bool
		enabled    string
		path       string
		wantStatus int
	}{
		{
			name:       "disabled_allows_all",
			enabled:    "false",
			path:       "/api/v1/auth/register",
			wantStatus: http.StatusOK,
		},
		{
			name:       "nil_service_allows_all",
			nilService: true,
			path:       "/api/v1/auth/register",
			wantStatus: http.StatusOK,
		},
		{
			name:       "enabled_allows_login",
			enabled:    "true",
			path:       "/api/v1/auth/login",
			wantStatus: http.StatusOK,
		},
		{
			name:       "enabled_allows_login_2fa",
			enabled:    "true",
			path:       "/api/v1/auth/login/2fa",
			wantStatus: http.StatusOK,
		},
		{
			name:       "enabled_allows_logout",
			enabled:    "true",
			path:       "/api/v1/auth/logout",
			wantStatus: http.StatusOK,
		},
		{
			name:       "enabled_allows_refresh",
			enabled:    "true",
			path:       "/api/v1/auth/refresh",
			wantStatus: http.StatusOK,
		},
		{
			name:       "enabled_blocks_linuxdo_oauth_start",
			enabled:    "true",
			path:       "/api/v1/auth/oauth/linuxdo/start",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_linuxdo_oauth_callback",
			enabled:    "true",
			path:       "/api/v1/auth/oauth/linuxdo/callback",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_wechat_oauth_start",
			enabled:    "true",
			path:       "/api/v1/auth/oauth/wechat/start",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_wechat_oauth_callback",
			enabled:    "true",
			path:       "/api/v1/auth/oauth/wechat/callback",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_wechat_payment_oauth_start",
			enabled:    "true",
			path:       "/api/v1/auth/oauth/wechat/payment/start",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_wechat_payment_oauth_callback",
			enabled:    "true",
			path:       "/api/v1/auth/oauth/wechat/payment/callback",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_oidc_oauth_start",
			enabled:    "true",
			path:       "/api/v1/auth/oauth/oidc/start",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_oidc_oauth_callback",
			enabled:    "true",
			path:       "/api/v1/auth/oauth/oidc/callback",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_github_oauth_start",
			enabled:    "true",
			path:       "/api/v1/auth/oauth/github/start",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_github_oauth_callback",
			enabled:    "true",
			path:       "/api/v1/auth/oauth/github/callback",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_github_complete_registration",
			enabled:    "true",
			path:       "/api/v1/auth/oauth/github/complete-registration",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_google_oauth_start",
			enabled:    "true",
			path:       "/api/v1/auth/oauth/google/start",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_google_oauth_callback",
			enabled:    "true",
			path:       "/api/v1/auth/oauth/google/callback",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_google_complete_registration",
			enabled:    "true",
			path:       "/api/v1/auth/oauth/google/complete-registration",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_dingtalk_oauth_start",
			enabled:    "true",
			path:       "/api/v1/auth/oauth/dingtalk/start",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_dingtalk_oauth_callback",
			enabled:    "true",
			path:       "/api/v1/auth/oauth/dingtalk/callback",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_dingtalk_complete_registration",
			enabled:    "true",
			path:       "/api/v1/auth/oauth/dingtalk/complete-registration",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_dingtalk_create_account",
			enabled:    "true",
			path:       "/api/v1/auth/oauth/dingtalk/create-account",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_dingtalk_bind_login",
			enabled:    "true",
			path:       "/api/v1/auth/oauth/dingtalk/bind-login",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_oauth_pending_exchange",
			enabled:    "true",
			path:       "/api/v1/auth/oauth/pending/exchange",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_oauth_pending_send_verify_code",
			enabled:    "true",
			path:       "/api/v1/auth/oauth/pending/send-verify-code",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_oauth_pending_create_account",
			enabled:    "true",
			path:       "/api/v1/auth/oauth/pending/create-account",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_oauth_pending_bind_login",
			enabled:    "true",
			path:       "/api/v1/auth/oauth/pending/bind-login",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_provider_bind_login",
			enabled:    "true",
			path:       "/api/v1/auth/oauth/oidc/bind-login",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_provider_create_account",
			enabled:    "true",
			path:       "/api/v1/auth/oauth/wechat/create-account",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_legacy_complete_registration",
			enabled:    "true",
			path:       "/api/v1/auth/oauth/linuxdo/complete-registration",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_register",
			enabled:    "true",
			path:       "/api/v1/auth/register",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_forgot_password",
			enabled:    "true",
			path:       "/api/v1/auth/forgot-password",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_reset_password",
			enabled:    "true",
			path:       "/api/v1/auth/reset-password",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_promo_validation",
			enabled:    "true",
			path:       "/api/v1/auth/validate-promo-code",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "enabled_blocks_invite_flow",
			enabled:    "true",
			path:       "/api/v1/auth/invite/accept",
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			r := gin.New()

			var svc *service.SettingService
			if !tc.nilService {
				svc = newBackendModeSettingService(t, tc.enabled)
			}

			r.Use(BackendModeAuthGuard(svc))
			r.Any("/*path", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			r.ServeHTTP(w, req)

			require.Equal(t, tc.wantStatus, w.Code)
		})
	}
}

func TestBackendModeProductSurfacePolicy(t *testing.T) {
	blocked := []string{
		"/api/v1/pages",
		"/api/v1/pages/terms",
		"/api/v1/keys",
		"/api/v1/keys/key-1",
		"/api/v1/groups/available",
		"/api/v1/channels/available",
		"/api/v1/usage",
		"/api/v1/channel-monitors",
		"/api/v1/video/tasks/task-1/local-asset",
		"/api/v1/user/aff",
		"/api/v1/user/account-bindings/email",
		"/api/v1/user/auth-identities/bind/start",
		"/api/v1/user/api-keys/key-1/usage/daily",
		"/api/v1/user/platform-quotas",
		"/api/v1/user/notify-email/send-code",
		"/api/v1/payment/config",
		"/api/v1/payment/public/orders/verify",
		"/api/v1/payment/webhook/stripe",
		"/api/v1/admin/payment/config",
		"/api/v1/public/asset-handoffs/consume",
		"/api/v1/user/video/simulation/tasks",
		"/api/v1/admin/video/simulation/tasks/task-1/result",
		"/api/v1/subscriptions",
		"/api/v1/redeem",
		"/api/v1/announcements",
		"/api/v1/admin/subscriptions",
		"/api/v1/admin/users/42/subscriptions",
		"/api/v1/admin/groups/7/subscriptions",
		"/api/v1/admin/redeem-codes",
		"/api/v1/admin/promo-codes",
		"/api/v1/admin/affiliates/invites",
		"/api/v1/admin/announcements",
		"/api/v1/admin/risk-control/config",
		"/api/v1/admin/data-management/backups",
		"/api/v1/admin/scheduled-test-plans",
		"/api/v1/admin/video/providers/9/tiny-real-authorization",
		"/api/v1/admin/video/providers/9/production-smoke",
		"/api/v1/admin/dashboard/monthly-budget",
		"/api/v1/admin/users/42/balance",
		"/api/v1/admin/users/42/platform-quotas",
		"/api/v1/admin/review-only/tasks",
		"/api/v1/admin/mock/tasks",
		"/api/v1/admin/not-a-real-release-resource",
	}
	for _, path := range blocked {
		require.False(t, backendModeAllowsProductPath(path), path)
	}

	allowed := []string{
		"/api/v1/auth/login",
		"/api/v1/auth/me",
		"/api/v1/auth/revoke-all-sessions",
		"/api/v1/user/profile",
		"/api/v1/user/totp/status",
		"/api/v1/admin/users",
		"/api/v1/admin/users/42/api-keys",
		"/api/v1/admin/accounts",
		"/api/v1/admin/settings",
		"/v1/responses",
		"/v1/chat/completions",
		"/v1/messages",
		"/v1/video/tasks",
		"/v1beta/models/gemini:generateContent",
	}
	for _, path := range allowed {
		require.True(t, backendModeAllowsProductPath(path), path)
	}

	// Unknown administrator resources fail closed even when their names merely
	// resemble a retained capability.
	require.False(t, backendModeAllowsProductPath("/api/v1/admin/payment-processor"))
}

func TestBackendModeProductSurfaceMethodPolicy(t *testing.T) {
	allowed := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v1/user/profile"},
		{method: http.MethodPut, path: "/api/v1/user"},
		{method: http.MethodPut, path: "/api/v1/user/password"},
		{method: http.MethodGet, path: "/api/v1/user/totp/status"},
		{method: http.MethodPost, path: "/api/v1/user/totp/setup"},
		{method: http.MethodGet, path: "/api/v1/settings/public"},
		{method: http.MethodGet, path: "/api/v1/admin/users"},
		{method: http.MethodPost, path: "/api/v1/admin/users"},
		{method: http.MethodPost, path: "/api/v1/admin/users/42/api-keys"},
		{method: http.MethodPut, path: "/api/v1/admin/api-keys/87"},
		{method: http.MethodDelete, path: "/api/v1/admin/api-keys/87"},
		{method: http.MethodPost, path: "/api/v1/admin/dashboard/users-usage"},
		{method: http.MethodPost, path: "/api/v1/admin/dashboard/api-keys-usage"},
		{method: http.MethodGet, path: "/api/v1/admin/accounts"},
		{method: http.MethodPost, path: "/api/v1/admin/accounts"},
		{method: http.MethodPut, path: "/api/v1/admin/accounts/9"},
		{method: http.MethodPost, path: "/api/v1/admin/accounts/9/apply-oauth-credentials"},
		{method: http.MethodPost, path: "/api/v1/admin/accounts/generate-auth-url"},
		{method: http.MethodPost, path: "/api/v1/admin/accounts/import/codex-session"},
		{method: http.MethodPost, path: "/api/v1/admin/accounts/batch-update-credentials"},
		{method: http.MethodPost, path: "/api/v1/admin/openai/generate-auth-url"},
		{method: http.MethodPost, path: "/api/v1/admin/openai/exchange-code"},
		{method: http.MethodGet, path: "/api/v1/admin/openai/accounts/9/quota"},
		{method: http.MethodPost, path: "/api/v1/admin/gemini/oauth/auth-url"},
		{method: http.MethodGet, path: "/api/v1/admin/gemini/oauth/capabilities"},
		{method: http.MethodPost, path: "/api/v1/admin/antigravity/oauth/auth-url"},
		{method: http.MethodPost, path: "/api/v1/admin/grok/oauth/auth-url"},
		{method: http.MethodPost, path: "/api/v1/admin/grok/oauth/create-from-oauth"},
		{method: http.MethodGet, path: "/api/v1/admin/grok/runtime-sanity"},
		{method: http.MethodGet, path: "/api/v1/admin/video/providers"},
		{method: http.MethodPut, path: "/api/v1/admin/video/providers/9"},
		{method: http.MethodGet, path: "/api/v1/admin/video/tasks/501/local-asset"},
		{method: http.MethodGet, path: "/api/v1/admin/generation-content/samples"},
		{method: http.MethodGet, path: "/api/v1/admin/usage"},
		{method: http.MethodPost, path: "/api/v1/admin/backups"},
		{method: http.MethodPost, path: "/api/v1/admin/backups/12/restore"},
		{method: http.MethodGet, path: "/api/v1/admin/ops/dashboard/overview"},
		{method: http.MethodPost, path: "/api/v1/admin/ops/alert-rules"},
		{method: http.MethodPut, path: "/api/v1/admin/ops/runtime/alert"},
		{method: http.MethodPost, path: "/api/v1/admin/ops/system-logs/cleanup"},
		{method: http.MethodGet, path: "/api/v1/admin/settings"},
	}
	for _, tc := range allowed {
		require.True(t, backendModeAllowsProductRequest(tc.method, tc.path), "%s %s", tc.method, tc.path)
	}

	blocked := []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/user/profile"},
		{method: http.MethodGet, path: "/api/v1/user"},
		{method: http.MethodDelete, path: "/api/v1/user/totp/status"},
		{method: http.MethodPost, path: "/api/v1/settings/public"},
		{method: http.MethodPut, path: "/api/v1/settings/public"},
		{method: http.MethodGet, path: "/api/v1/settings/email-unsubscribe"},
		{method: http.MethodPost, path: "/api/v1/admin/video/tasks/501/asset-handoffs"},
		{method: http.MethodPost, path: "/api/v1/admin/video/providers/9/tiny-real-authorization"},
		{method: http.MethodPost, path: "/api/v1/admin/video/providers/9/production-smoke"},
		{method: http.MethodPut, path: "/api/v1/admin/dashboard/monthly-budget"},
		{method: http.MethodPost, path: "/api/v1/admin/users/42/balance"},
		{method: http.MethodGet, path: "/api/v1/admin/users/42/platform-quotas"},
		{method: http.MethodPut, path: "/api/v1/admin/groups/7/rpm-overrides"},
		{method: http.MethodDelete, path: "/api/v1/admin/groups/7/rpm-overrides"},
		{method: http.MethodPost, path: "/api/v1/admin/scheduled-test-plans"},
		{method: http.MethodGet, path: "/api/v1/admin/accounts/9/scheduled-test-plans"},
		{method: http.MethodGet, path: "/api/v1/admin/data-management/backups"},
		{method: http.MethodGet, path: "/api/v1/admin/mock/tasks"},
		{method: http.MethodGet, path: "/api/v1/admin/review-only/tasks"},
		{method: http.MethodGet, path: "/api/v1/admin/ops/review-only"},
		{method: http.MethodPost, path: "/api/v1/admin/accounts/9/reset-quota"},
		{method: http.MethodGet, path: "/api/v1/admin/unknown"},
		{method: http.MethodPut, path: "/api/v1/admin/settings"},
		{method: http.MethodGet, path: "/api/v1/admin/settings/payment"},
		{method: http.MethodPut, path: "/api/v1/admin/settings/payment"},
		{method: http.MethodPost, path: "/api/v1/admin/settings/admin-api-key"},
		{method: http.MethodDelete, path: "/api/v1/admin/settings/runtime/config"},
	}
	for _, tc := range blocked {
		require.False(t, backendModeAllowsProductRequest(tc.method, tc.path), "%s %s", tc.method, tc.path)
	}
}

func TestBackendModeProductSurfaceGuard(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	lanAdminSettings := service.NewSettingService(&bmSettingRepo{values: map[string]string{
		service.SettingKeyBackendModeEnabled: "true",
	}}, &config.Config{DeploymentProfile: config.DeploymentProfileLANAdmin})
	r.Use(BackendModeProductSurfaceGuard(lanAdminSettings))
	r.Any("/*path", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	for _, tc := range []struct {
		method     string
		path       string
		wantStatus int
	}{
		{method: http.MethodGet, path: "/api/v1/payment/webhook/stripe", wantStatus: http.StatusForbidden},
		{method: http.MethodGet, path: "/api/v1/keys", wantStatus: http.StatusForbidden},
		{method: http.MethodPost, path: "/api/v1/admin/video/tasks/501/asset-handoffs", wantStatus: http.StatusForbidden},
		{method: http.MethodGet, path: "/api/v1/admin/users", wantStatus: http.StatusOK},
		{method: http.MethodGet, path: "/v1/chat/completions", wantStatus: http.StatusOK},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		r.ServeHTTP(w, req)
		require.Equal(t, tc.wantStatus, w.Code, tc.path)
	}
}

func TestBackendModeProductSurfaceGuard_DoesNotApplyLANAllowlistToStandardProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(BackendModeProductSurfaceGuard(newBackendModeSettingService(t, "true")))
	r.Any("/*path", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/payment/webhook/stripe", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}
