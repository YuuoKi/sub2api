package dto

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestAccountFromServiceShallow_RedactsSensitiveCredentials(t *testing.T) {
	src := &service.Account{
		ID:       42,
		Name:     "demo",
		Platform: "anthropic",
		Type:     "oauth",
		Credentials: map[string]any{
			"access_token":                      "at-secret",
			"refresh_token":                     "rt-secret",
			"id_token":                          "id-secret",
			"api_key":                           "sk-secret",
			service.HCAtomAPIKeyCiphertextField: "hc1:sentinel-ciphertext",
			service.HCAtomAPIKeyMaskedField:     "********7x9Q",
			service.HCAtomAPIKeyConfiguredField: true,
			"base_url":                          "https://api.example.com",
			"model_mapping":                     map[string]any{"foo": "bar"},
		},
	}

	got := AccountFromServiceShallow(src)
	require.NotNil(t, got)

	// 敏感键不在 Credentials 里
	require.NotContains(t, got.Credentials, "access_token")
	require.NotContains(t, got.Credentials, "refresh_token")
	require.NotContains(t, got.Credentials, "id_token")
	require.NotContains(t, got.Credentials, "api_key")
	require.NotContains(t, got.Credentials, service.HCAtomAPIKeyCiphertextField)
	require.Equal(t, "********7x9Q", got.Credentials[service.HCAtomAPIKeyMaskedField])
	require.Equal(t, true, got.Credentials[service.HCAtomAPIKeyConfiguredField])
	// 非敏感键保留
	require.Equal(t, "https://api.example.com", got.Credentials["base_url"])
	require.Equal(t, map[string]any{"foo": "bar"}, got.Credentials["model_mapping"])

	// 状态 map 标记敏感键存在
	require.True(t, got.CredentialsStatus["has_access_token"])
	require.True(t, got.CredentialsStatus["has_refresh_token"])
	require.True(t, got.CredentialsStatus["has_id_token"])
	require.True(t, got.CredentialsStatus["has_api_key"])
	require.True(t, got.CredentialsStatus["has_"+service.HCAtomAPIKeyCiphertextField])

	// JSON 序列化校验：响应体里不会出现敏感子串
	raw, err := json.Marshal(got)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "rt-secret")
	require.NotContains(t, string(raw), "at-secret")
	require.NotContains(t, string(raw), "sk-secret")
	require.NotContains(t, string(raw), "id-secret")
	require.NotContains(t, string(raw), "hc1:sentinel-ciphertext")
	// 状态标识应序列化进 JSON
	require.Contains(t, string(raw), "credentials_status")
	require.Contains(t, string(raw), "has_refresh_token")

	// 原始 service.Account 不应被改动
	require.Equal(t, "rt-secret", src.Credentials["refresh_token"])
}

func TestAccountFromServiceShallow_NilCredentialsOmitsStatus(t *testing.T) {
	src := &service.Account{ID: 1, Name: "n", Platform: "anthropic", Type: "oauth"}
	got := AccountFromServiceShallow(src)
	require.NotNil(t, got)
	require.Nil(t, got.Credentials)
	require.Nil(t, got.CredentialsStatus)
}

func TestAccountFromServiceShallow_HCAtomUsesStrictPublicAllowlistRecursively(t *testing.T) {
	sentinel := strings.Join([]string{"hc-review", "dto", "secret"}, "-")
	src := &service.Account{
		ID:       43,
		Name:     "hc-atom",
		Platform: service.PlatformHCAtom,
		Type:     service.AccountTypeAPIKey,
		Credentials: map[string]any{
			"protocol":                          "hc-atom",
			service.HCAtomAPIKeyCiphertextField: "hc1:synthetic-ciphertext",
			service.HCAtomAPIKeyMaskedField:     "********7x9Q",
			service.HCAtomAPIKeyConfiguredField: true,
			"Authorization":                     "Bearer " + sentinel,
			"metadata":                          []any{map[string]any{"token": sentinel}},
			"model_mapping": map[string]any{
				"seedream-5.0": map[string]any{"api_key": sentinel},
			},
		},
	}

	got := AccountFromServiceShallow(src)
	raw, err := json.Marshal(got)
	require.NoError(t, err)
	require.NotContains(t, string(raw), sentinel)
	require.Equal(t, "hc-atom", got.Credentials["protocol"])
	require.NotContains(t, got.Credentials, "Authorization")
	require.NotContains(t, got.Credentials, "metadata")
	require.NotContains(t, got.Credentials, "model_mapping")
	require.NotContains(t, got.Credentials, service.HCAtomAPIKeyCiphertextField)
	require.True(t, got.CredentialsStatus["has_"+service.HCAtomAPIKeyCiphertextField])
}
