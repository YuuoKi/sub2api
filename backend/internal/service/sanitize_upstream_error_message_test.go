package service

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizeUpstreamErrorMessage_RedactsPercentEncodedProxyCredentials(t *testing.T) {
	t.Parallel()

	username := "relay-user"
	password := "s3cret-pass"
	proxyURL := "http://" + username + ":" + password + "@proxy.example:8080"
	encodedProxy := url.QueryEscape(proxyURL)

	msg := `Get "https://custom.relay.example/v1/chat?proxy=` + encodedProxy + `": dial tcp: i/o timeout`

	got := sanitizeUpstreamErrorMessage(msg)

	require.NotContains(t, got, username, "sanitized message must not contain proxy username")
	require.NotContains(t, got, password, "sanitized message must not contain proxy password")
	require.NotContains(t, strings.ToLower(got), strings.ToLower(url.QueryEscape(username)))
	require.NotContains(t, strings.ToLower(got), strings.ToLower(url.QueryEscape(password)))
	require.Contains(t, got, "proxy=")
	require.Contains(t, got, "***")
}

func TestSanitizeUpstreamErrorMessage_StillRedactsExistingSensitiveQueryParams(t *testing.T) {
	t.Parallel()

	msg := `upstream 401: https://api.example/oauth?access_token=tok_abc&refresh_token=ref_xyz&key=sk-123`
	got := sanitizeUpstreamErrorMessage(msg)

	require.NotContains(t, got, "tok_abc")
	require.NotContains(t, got, "ref_xyz")
	require.NotContains(t, got, "sk-123")
	require.Contains(t, got, "access_token=***")
	require.Contains(t, got, "refresh_token=***")
	require.Contains(t, got, "key=***")
}
