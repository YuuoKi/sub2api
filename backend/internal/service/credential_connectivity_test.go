//go:build unit

package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type connectivityRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn connectivityRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func connectivityHTTPClient(fn connectivityRoundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func TestAccountConnectivityUsesNonGeneratingHCProbeAndBothAcceptedAuthHeaders(t *testing.T) {
	cipher, err := NewHCAtomCredentialCipher(strings.Repeat("42", 32))
	require.NoError(t, err)
	ciphertext, err := cipher.Encrypt("account-connectivity-secret")
	require.NoError(t, err)
	account := &Account{
		ID:       17,
		Platform: PlatformHCAtom,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			HCAtomAPIKeyCiphertextField: ciphertext,
		},
	}
	repo := &mockAccountRepoForPlatform{accountsByID: map[int64]*Account{account.ID: account}}
	var requests []*http.Request
	service := &adminServiceImpl{
		accountRepo:            repo,
		hcAtomCredentialCipher: cipher,
		connectivityClient: connectivityHTTPClient(func(req *http.Request) (*http.Response, error) {
			requests = append(requests, req)
			status := http.StatusNotFound
			if req.URL.Host == "ai-aigc.fzyinghe.com" {
				status = http.StatusMethodNotAllowed
			}
			return &http.Response{
				StatusCode: status,
				Body:       io.NopCloser(strings.NewReader(`{"code":"TASK_NOT_FOUND"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	result, err := service.CheckAccountConnectivity(context.Background(), account.ID)
	require.NoError(t, err)
	require.Equal(t, CredentialConnectivityOK, result.Status)
	require.True(t, result.Reachable)
	require.Equal(t, CredentialAuthAccepted, result.Authentication)
	require.False(t, result.GenerationStarted)
	require.Len(t, requests, 2)
	require.Equal(t, http.MethodGet, requests[0].Method)
	require.Equal(t, "/v1/images/generations", requests[0].URL.Path)
	require.Equal(t, "Bearer account-connectivity-secret", requests[0].Header.Get("Authorization"))
	require.Empty(t, requests[0].Header.Get("x-api-key"))
	require.Equal(t, http.MethodGet, requests[1].Method)
	require.Contains(t, requests[1].URL.Path, "/image/generation/tasks/sub2-connectivity-")
	require.Equal(t, "Bearer account-connectivity-secret", requests[1].Header.Get("Authorization"))
	require.Equal(t, "account-connectivity-secret", requests[1].Header.Get("x-api-key"))
}

func TestCredentialConnectivityReportsRejectedKeyWithoutStartingGeneration(t *testing.T) {
	result := checkCredentialConnectivity(
		context.Background(),
		connectivityHTTPClient(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(strings.NewReader(`{"message":"invalid api key"}`)),
				Header:     make(http.Header),
			}, nil
		}),
		credentialConnectivityTarget{URL: HCAtomMediaOrigin + "/image/generation/tasks/probe"},
		"rejected-secret",
	)

	require.Equal(t, CredentialConnectivityError, result.Status)
	require.True(t, result.Reachable)
	require.Equal(t, CredentialAuthRejected, result.Authentication)
	require.False(t, result.GenerationStarted)
	require.NotContains(t, result.Message, "rejected-secret")
}

func TestCredentialConnectivityRejectsBusinessEnvelopeAuthenticationFailure(t *testing.T) {
	result := checkCredentialConnectivity(
		context.Background(),
		connectivityHTTPClient(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"code":1001,"msg":"认证失败，密钥无效"}`)),
				Header:     make(http.Header),
			}, nil
		}),
		credentialConnectivityTarget{URL: HCAtomMediaOrigin + "/image/generation/tasks/probe"},
		"rejected-business-secret",
	)

	require.Equal(t, CredentialConnectivityError, result.Status)
	require.Equal(t, CredentialAuthRejected, result.Authentication)
	require.False(t, result.GenerationStarted)
}

func TestVideoConnectivityUsesProviderSpecificPollContractsWithoutPOST(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		expectedPath string
	}{
		{name: "official Seedance", provider: "seedance", expectedPath: "/api/v3/contents/generations/tasks/sub2-connectivity-"},
		{name: "HC video V1", provider: HCAtomVideoV1Provider, expectedPath: "/video/generation/tasks/sub2-connectivity-"},
		{name: "HC Seedance V3", provider: HCAtomSeedanceV3Provider, expectedPath: "/v3/video/tasks/sub2-connectivity-"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &fakeVideoAdminRepo{providers: []VideoProviderAccount{{
				ID:              31,
				Provider:        test.provider,
				EncryptedAPIKey: "video-connectivity-secret",
			}}}
			var request *http.Request
			service := NewVideoAdminService(repo, fakeVideoEncryptor{})
			service.connectivityClient = connectivityHTTPClient(func(req *http.Request) (*http.Response, error) {
				request = req
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"code":"TASK_NOT_FOUND"}`)),
					Header:     make(http.Header),
				}, nil
			})

			result, err := service.CheckProviderConnectivity(context.Background(), 31)
			require.NoError(t, err)
			require.Equal(t, CredentialConnectivityOK, result.Status)
			require.False(t, result.GenerationStarted)
			require.NotNil(t, request)
			require.Equal(t, http.MethodGet, request.Method)
			require.Contains(t, request.URL.Path, test.expectedPath)
			require.Equal(t, "Bearer video-connectivity-secret", request.Header.Get("Authorization"))
		})
	}
}

func TestUnsupportedAccountConnectivityNeverCallsUpstream(t *testing.T) {
	account := &Account{ID: 88, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	repo := &mockAccountRepoForPlatform{accountsByID: map[int64]*Account{account.ID: account}}
	called := false
	service := &adminServiceImpl{
		accountRepo: repo,
		connectivityClient: connectivityHTTPClient(func(*http.Request) (*http.Response, error) {
			called = true
			return nil, nil
		}),
	}

	result, err := service.CheckAccountConnectivity(context.Background(), account.ID)
	require.NoError(t, err)
	require.Equal(t, CredentialConnectivityWarning, result.Status)
	require.False(t, result.Supported)
	require.False(t, result.GenerationStarted)
	require.False(t, called)
}
