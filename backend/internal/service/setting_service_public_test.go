//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type settingPublicRepoStub struct {
	values map[string]string
	err    error
	calls  int
}

func (s *settingPublicRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *settingPublicRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}

func (s *settingPublicRepoStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *settingPublicRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func TestSettingService_GetPublicSettings_LANAdminProfileOverridesDatabaseFalse(t *testing.T) {
	repo := &settingPublicRepoStub{values: map[string]string{
		SettingKeyBackendModeEnabled: "false",
	}}
	svc := NewSettingService(repo, &config.Config{DeploymentProfile: config.DeploymentProfileLANAdmin})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.BackendModeEnabled)
	require.True(t, settings.LANAdminModeEnabled)
}

func TestSettingService_GetPublicSettings_StandardBackendModeIsNotLANAdminMode(t *testing.T) {
	repo := &settingPublicRepoStub{values: map[string]string{
		SettingKeyBackendModeEnabled: "true",
	}}
	svc := NewSettingService(repo, &config.Config{DeploymentProfile: config.DeploymentProfileStandard})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.BackendModeEnabled)
	require.False(t, settings.LANAdminModeEnabled)
}

func TestSettingService_GetPublicSettingsForInjection_LANAdminProfileFailsClosedOnRepositoryError(t *testing.T) {
	repo := &settingPublicRepoStub{err: errors.New("database unavailable")}
	svc := NewSettingService(repo, &config.Config{DeploymentProfile: config.DeploymentProfileLANAdmin})
	svc.SetVersion("test-version")

	payload, err := svc.GetPublicSettingsForInjection(context.Background())
	require.NoError(t, err)
	injection, ok := payload.(*PublicSettingsInjectionPayload)
	require.True(t, ok)
	require.True(t, injection.BackendModeEnabled)
	require.True(t, injection.LANAdminModeEnabled)
	require.NotEmpty(t, injection.SiteName)
	require.Equal(t, "test-version", injection.Version)
	require.NotEmpty(t, injection.ServerTimezone)
	require.NotEmpty(t, injection.ServerUTCOffset)
	require.Equal(t, 1, repo.calls)
}

func TestSettingService_GetPublicSettingsForInjection_StandardProfilePreservesRepositoryError(t *testing.T) {
	wantErr := errors.New("database unavailable")
	repo := &settingPublicRepoStub{err: wantErr}
	svc := NewSettingService(repo, &config.Config{DeploymentProfile: config.DeploymentProfileStandard})

	payload, err := svc.GetPublicSettingsForInjection(context.Background())
	require.Nil(t, payload)
	require.ErrorIs(t, err, wantErr)
}

func (s *settingPublicRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *settingPublicRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *settingPublicRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

func TestSettingService_GetPublicSettings_ExposesRegistrationEmailSuffixWhitelist(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyRegistrationEnabled:              "true",
			SettingKeyEmailVerifyEnabled:               "true",
			SettingKeyRegistrationEmailSuffixWhitelist: `["@EXAMPLE.com"," @foo.bar ","*.EDU.CN","@invalid_domain",""]`,
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"@example.com", "@foo.bar", "*.edu.cn"}, settings.RegistrationEmailSuffixWhitelist)
}

func TestSettingService_GetPublicSettings_ExposesTablePreferences(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyTableDefaultPageSize: "50",
			SettingKeyTablePageSizeOptions: "[20,50,100]",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, 50, settings.TableDefaultPageSize)
	require.Equal(t, []int{20, 50, 100}, settings.TablePageSizeOptions)
}

func TestSettingService_GetPublicSettings_NormalizesSiteName(t *testing.T) {
	tests := []struct {
		name     string
		siteName string
		want     string
	}{
		{name: "missing", want: "无界 · 企业 AI 管理中台"},
		{name: "blank", siteName: "   ", want: "无界 · 企业 AI 管理中台"},
		{name: "upstream default", siteName: "Sub2API", want: "无界 · 企业 AI 管理中台"},
		{name: "custom", siteName: "  My Custom Console  ", want: "My Custom Console"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &settingPublicRepoStub{values: map[string]string{}}
			if tt.siteName != "" {
				repo.values[SettingKeySiteName] = tt.siteName
			}
			svc := NewSettingService(repo, &config.Config{})

			settings, err := svc.GetPublicSettings(context.Background())
			require.NoError(t, err)
			require.Equal(t, tt.want, settings.SiteName)
		})
	}
}

func TestSettingService_ParseSettings_NormalizesSiteName(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{}}, &config.Config{})

	require.Equal(t, "无界 · 企业 AI 管理中台", svc.parseSettings(map[string]string{}).SiteName)
	require.Equal(t, "无界 · 企业 AI 管理中台", svc.parseSettings(map[string]string{SettingKeySiteName: "Sub2API"}).SiteName)
	require.Equal(t, "My Custom Console", svc.parseSettings(map[string]string{SettingKeySiteName: "  My Custom Console  "}).SiteName)
}

func TestSettingService_GetSiteName_NormalizesSiteName(t *testing.T) {
	tests := []struct {
		name    string
		present bool
		value   string
		want    string
	}{
		{name: "missing", want: "无界 · 企业 AI 管理中台"},
		{name: "blank", present: true, value: "  ", want: "无界 · 企业 AI 管理中台"},
		{name: "upstream default", present: true, value: "Sub2API", want: "无界 · 企业 AI 管理中台"},
		{name: "custom", present: true, value: "  My Custom Console  ", want: "My Custom Console"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &settingPublicRepoStub{values: map[string]string{}}
			if tt.present {
				repo.values[SettingKeySiteName] = tt.value
			}
			svc := NewSettingService(repo, &config.Config{})

			require.Equal(t, tt.want, svc.GetSiteName(context.Background()))
		})
	}
}

func TestSettingService_GetPublicSettings_ExposesForceEmailOnThirdPartySignup(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyForceEmailOnThirdPartySignup: "true",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.ForceEmailOnThirdPartySignup)
}

func TestSettingService_GetPublicSettings_ExposesAllowUserViewErrorRequests(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyAllowUserViewErrorRequests: "true",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.AllowUserViewErrorRequests)
}

func TestSettingService_GetPublicSettings_ExposesWeChatOAuthModeCapabilities(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{
		values: map[string]string{
			SettingKeyWeChatConnectEnabled:             "true",
			SettingKeyWeChatConnectAppID:               "wx-mp-app",
			SettingKeyWeChatConnectAppSecret:           "wx-mp-secret",
			SettingKeyWeChatConnectMode:                "mp",
			SettingKeyWeChatConnectScopes:              "snsapi_base",
			SettingKeyWeChatConnectOpenEnabled:         "true",
			SettingKeyWeChatConnectMPEnabled:           "true",
			SettingKeyWeChatConnectRedirectURL:         "https://api.example.com/api/v1/auth/oauth/wechat/callback",
			SettingKeyWeChatConnectFrontendRedirectURL: "/auth/wechat/callback",
		},
	}, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.WeChatOAuthEnabled)
	require.True(t, settings.WeChatOAuthOpenEnabled)
	require.True(t, settings.WeChatOAuthMPEnabled)
}

func TestSettingService_GetPublicSettings_DoesNotExposeMobileOnlyWeChatAsWebOAuthAvailable(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{
		values: map[string]string{
			SettingKeyWeChatConnectEnabled:             "true",
			SettingKeyWeChatConnectMobileEnabled:       "true",
			SettingKeyWeChatConnectMode:                "mobile",
			SettingKeyWeChatConnectMobileAppID:         "wx-mobile-app",
			SettingKeyWeChatConnectMobileAppSecret:     "wx-mobile-secret",
			SettingKeyWeChatConnectFrontendRedirectURL: "/auth/wechat/callback",
		},
	}, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.False(t, settings.WeChatOAuthEnabled)
	require.False(t, settings.WeChatOAuthOpenEnabled)
	require.False(t, settings.WeChatOAuthMPEnabled)
	require.True(t, settings.WeChatOAuthMobileEnabled)
}

func TestSettingService_GetPublicSettings_FallsBackToConfigForWeChatOAuthCapabilities(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{}}, &config.Config{
		WeChat: config.WeChatConnectConfig{
			Enabled:             true,
			OpenEnabled:         true,
			OpenAppID:           "wx-open-config",
			OpenAppSecret:       "wx-open-secret",
			FrontendRedirectURL: "/auth/wechat/config-callback",
		},
	})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.WeChatOAuthEnabled)
	require.True(t, settings.WeChatOAuthOpenEnabled)
	require.False(t, settings.WeChatOAuthMPEnabled)
	require.False(t, settings.WeChatOAuthMobileEnabled)
}
