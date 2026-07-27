package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Regression target: a provider change that redirects HC requests, flattens its
// body, loses the intent key, or uses a non-HC account must fail this test.
func TestHCAtomBatchProvider_SubmitUsesFixedAsyncContract(t *testing.T) {
	transport := hcAtomRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "POST", req.Method)
		require.Equal(t, "https", req.URL.Scheme)
		require.Equal(t, "api-aigc.fzyinghe.com", req.URL.Host)
		require.Equal(t, "/image/generation/tasks", req.URL.Path)
		require.Equal(t, "Bearer hc-test-key", req.Header.Get("Authorization"))
		require.Equal(t, "hc-image:imgbatch_123", req.Header.Get("Idempotency-Key"))

		var body map[string]any
		require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
		require.Equal(t, "seedream-5.0", body["model"])
		require.Equal(t, map[string]any{"prompt": "A clean product hero image"}, body["input"])
		require.Equal(t, map[string]any{"aspect_ratio": "1:1", "image_size": "2K", "response_mime_type": "image/png"}, body["parameters"])

		return hcAtomHTTPResponse(http.StatusOK, `{"code":0,"msg":"ok","data":{"taskId":"hc-task-1","status":"PENDING"}}`), nil
	})
	provider := NewHCAtomBatchImageProvider(NewHCAtomBatchHTTPClient(&http.Client{Transport: transport}))

	got, err := provider.Submit(context.Background(), &BatchImageJob{BatchID: "imgbatch_123", Model: "seedream-5.0"}, &Account{
		Platform: PlatformHCAtom,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"hc_atom_api_key": "hc-test-key",
		},
	}, BatchImageInput{
		BatchID:          "imgbatch_123",
		Model:            "seedream-5.0",
		ResponseMimeType: "image/png",
		AspectRatio:      "1:1",
		ImageSize:        "2K",
		Items:            []BatchImageInputItem{{CustomID: "cover_001", Prompt: "A clean product hero image"}},
	})
	require.NoError(t, err)
	require.Equal(t, "hc-task-1", got.ProviderJobName)
	require.Equal(t, "PENDING", got.RawState)
}

// Regression target: treating an HC-only media group as Gemini would either
// reject valid HC work or send account selection to the wrong scheduler pool.
func TestHCAtomBatchProvider_UsesDedicatedPlatformAndMediaGroupGate(t *testing.T) {
	groupID := int64(42)
	service := &BatchImagePublicService{GroupRepo: &hcAtomBatchGroupRepo{groups: map[int64]*Group{
		groupID: {ID: groupID, Platform: PlatformHCAtom, AllowBatchImageGeneration: true},
	}}}

	require.Equal(t, PlatformHCAtom, batchImageProviderPlatform(BatchImageProviderHCAtom))
	require.NoError(t, service.ensureGroupAllowsBatchImage(context.Background(), &groupID))
}

type hcAtomRoundTripFunc func(*http.Request) (*http.Response, error)

func (f hcAtomRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func hcAtomHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

type hcAtomBatchGroupRepo struct{ groups map[int64]*Group }

func (r *hcAtomBatchGroupRepo) GetByIDLite(_ context.Context, id int64) (*Group, error) {
	if group, ok := r.groups[id]; ok {
		return group, nil
	}
	return nil, ErrGroupNotFound
}
