package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
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
	cipher, account := hcAtomBatchTestAccount(t, "hc-test-key")
	provider := NewHCAtomBatchImageProviderWithCredentialCipher(NewHCAtomBatchHTTPClient(&http.Client{Transport: transport}), cipher)

	got, err := provider.Submit(context.Background(), &BatchImageJob{BatchID: "imgbatch_123", Model: "seedream-5.0"}, account, BatchImageInput{
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

// Regression target: completion must consume validated image bytes through the
// existing JSONL/base64 index boundary, never persist or return the provider URL.
func TestHCAtomBatchProvider_OpenResultArchivesValidatedImageAsJSONL(t *testing.T) {
	resultTransport := hcAtomRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "https", req.URL.Scheme)
		require.Equal(t, "8.8.8.8", req.URL.Hostname())
		response := hcAtomHTTPResponse(http.StatusOK, "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01")
		response.Header.Set("Content-Type", "image/png")
		return response, nil
	})
	cipher, account := hcAtomBatchTestAccount(t, "test")
	provider := NewHCAtomBatchImageProviderWithResultClientAndCredentialCipher(&fakeHCAtomBatchClient{task: &HCAtomBatchTask{
		TaskID: "hc-task-1", Status: "SUCCESS", ResultURL: "https://8.8.8.8/result.png",
	}}, &http.Client{Transport: resultTransport}, cipher)
	jobName, customID := "hc-task-1", "item_000001"
	r, contentType, err := provider.OpenResult(context.Background(), &BatchImageJob{ProviderJobName: &jobName, ProviderInputRef: &customID}, account)
	require.NoError(t, err)
	defer r.Close()
	require.Equal(t, "application/jsonl", contentType)
	line, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Contains(t, string(line), `"custom_id":"item_000001"`)
	require.Contains(t, string(line), `"mimeType":"image/png"`)
	require.NotContains(t, string(line), "https://8.8.8.8")
}

// Regression target: a confirmed upstream DELETE is allowed to return HTTP 204
// with no envelope; treating it as a JSON protocol error would strand holds.
func TestHCAtomBatchHTTPClient_DeleteAcceptsNoContent(t *testing.T) {
	client := NewHCAtomBatchHTTPClient(&http.Client{Transport: hcAtomRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodDelete, req.Method)
		require.Equal(t, "/image/generation/tasks/hc-task-1", req.URL.Path)
		return hcAtomHTTPResponse(http.StatusNoContent, ""), nil
	})})
	require.NoError(t, client.Delete(context.Background(), "synthetic-key", "hc-task-1"))
}

func TestHCAtomBatchProvider_StrictStatusAndBusinessEnvelopeMapping(t *testing.T) {
	for _, tt := range []struct {
		state string
		want  BatchProviderInternalState
	}{
		{"PENDING", BatchProviderStateQueued}, {"RUNNING", BatchProviderStateRunning},
		{"FAILED", BatchProviderStateFailed}, {"CANCELLED", BatchProviderStateCancelled},
	} {
		t.Run(tt.state, func(t *testing.T) {
			got, err := mapHCAtomBatchState(&HCAtomBatchTask{Status: tt.state, ResultURL: "https://result.example/image.png"})
			require.NoError(t, err)
			require.Equal(t, tt.want, got.InternalState)
		})
	}
	_, err := mapHCAtomBatchState(&HCAtomBatchTask{Status: "MYSTERY"})
	require.Error(t, err)
	require.Equal(t, "HC_ATOM_PROTOCOL_ERROR", infraerrors.Reason(err))

	client := NewHCAtomBatchHTTPClient(&http.Client{Transport: hcAtomRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return hcAtomHTTPResponse(http.StatusOK, `{"code":40101,"msg":"hidden upstream detail","data":{}}`), nil
	})})
	_, err = client.Get(context.Background(), "synthetic-key", "hc-task-1")
	require.Error(t, err)
	require.Equal(t, "HC_ATOM_BUSINESS_ERROR", infraerrors.Reason(mapHCAtomBatchError(err)))
}

// Regression target: every HC result URL must stay inside the public HTTPS
// archive boundary before the client can consume even one response byte.
func TestHCAtomBatchProvider_ArchiveRejectsUnsafeURLFormsBeforeDownload(t *testing.T) {
	for _, rawURL := range []string{
		"https://user:pass@8.8.8.8/result.png",
		"https://8.8.8.8/result.png#fragment",
		"https://8.8.8.8:8443/result.png",
		"https://127.0.0.1/result.png",
		"https://0177.0.0.1/result.png",
	} {
		t.Run(rawURL, func(t *testing.T) {
			calls := 0
			provider := NewHCAtomBatchImageProviderWithResultClient(&fakeHCAtomBatchClient{}, &http.Client{Transport: hcAtomRoundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				response := hcAtomHTTPResponse(http.StatusOK, "\x89PNG\r\n\x1a\nvalid-test-image")
				response.Header.Set("Content-Type", "image/png")
				return response, nil
			})})
			_, _, err := provider.archiveResultURL(context.Background(), rawURL)
			require.Error(t, err)
			require.Zero(t, calls)
		})
	}
}

func TestHCAtomBatchProvider_ArchiveRejectsOversizedPNGDimensions(t *testing.T) {
	// PNG signature + IHDR declares a 100001 x 1 image. It is deliberately not
	// a full PNG: dimensions must be rejected before any decoder allocation.
	oversized := append([]byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"), []byte{0x00, 0x01, 0x86, 0xa1, 0x00, 0x00, 0x00, 0x01}...)
	provider := NewHCAtomBatchImageProviderWithResultClient(&fakeHCAtomBatchClient{}, &http.Client{Transport: hcAtomRoundTripFunc(func(*http.Request) (*http.Response, error) {
		response := hcAtomHTTPResponse(http.StatusOK, string(oversized))
		response.Header.Set("Content-Type", "image/png")
		return response, nil
	})})
	_, _, err := provider.archiveResultURL(context.Background(), "https://8.8.8.8/oversized.png")
	require.Error(t, err)
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

type fakeHCAtomBatchClient struct{ task *HCAtomBatchTask }

func hcAtomBatchTestAccount(t *testing.T, apiKey string) (HCAtomCredentialCipher, *Account) {
	t.Helper()
	cipher, err := NewHCAtomCredentialCipher(strings.Repeat("77", 32))
	require.NoError(t, err)
	credentials, err := ProtectHCAtomAccountCredentials(nil, map[string]any{"api_key": apiKey}, cipher)
	require.NoError(t, err)
	return cipher, &Account{Platform: PlatformHCAtom, Type: AccountTypeAPIKey, Credentials: credentials}
}

func (f *fakeHCAtomBatchClient) Create(context.Context, string, string, HCAtomBatchCreateRequest) (*HCAtomBatchTask, error) {
	return f.task, nil
}
func (f *fakeHCAtomBatchClient) Get(context.Context, string, string) (*HCAtomBatchTask, error) {
	return f.task, nil
}
func (f *fakeHCAtomBatchClient) Delete(context.Context, string, string) error { return nil }
