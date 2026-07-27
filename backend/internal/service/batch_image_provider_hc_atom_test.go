package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

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

// Regression target: HC URLs are one-time archival inputs, never a durable
// download dependency. A restarted provider must read the owned JSONL file.
func TestHCAtomBatchProvider_OpenResultPersistsOwnedJSONLForRestartedDownloads(t *testing.T) {
	resultTransport := hcAtomRoundTripFunc(func(*http.Request) (*http.Response, error) {
		response := hcAtomHTTPResponse(http.StatusOK, "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01")
		response.Header.Set("Content-Type", "image/png")
		return response, nil
	})
	cipher, account := hcAtomBatchTestAccount(t, "test")
	firstClient := &fakeHCAtomBatchClient{task: &HCAtomBatchTask{TaskID: "hc-owned-1", Status: "SUCCESS", ResultURL: "https://8.8.8.8/result.png"}}
	first := NewHCAtomBatchImageProviderWithOwnedResultStore(firstClient, &http.Client{Transport: resultTransport}, cipher, t.TempDir())
	taskID, customID := "hc-owned-1", "item_000001"
	job := &BatchImageJob{BatchID: "imgbatch_owned_1", ProviderJobName: &taskID, ProviderInputRef: &customID}

	r, _, err := first.OpenResult(context.Background(), job, account)
	require.NoError(t, err)
	owned, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())
	require.NotEmpty(t, owned)
	require.NotNil(t, job.ProviderOutputRef)
	require.True(t, strings.HasPrefix(*job.ProviderOutputRef, "hc_atom_owned:"))

	restarted := NewHCAtomBatchImageProviderWithOwnedResultStore(&fakeHCAtomBatchClient{err: errors.New("provider URL expired")}, &http.Client{Transport: resultTransport}, cipher, first.OwnedResultDir())
	r, _, err = restarted.OpenResult(context.Background(), job, account)
	require.NoError(t, err)
	got, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())
	require.Equal(t, owned, got)
	require.NoError(t, restarted.Cleanup(context.Background(), job, account, CleanupTargetOutput))
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

// Regression target: a 2xx transport response is not cancellation confirmation
// when HC's business envelope rejects the DELETE.
func TestHCAtomBatchHTTPClient_DeleteRejectsBusinessFailure(t *testing.T) {
	client := NewHCAtomBatchHTTPClient(&http.Client{Transport: hcAtomRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodDelete, req.Method)
		return hcAtomHTTPResponse(http.StatusOK, `{"code":40001,"msg":"not cancelled","data":{}}`), nil
	})})

	err := client.Delete(context.Background(), "synthetic-key", "hc-task-1")
	require.Error(t, err)
	require.Equal(t, "HC_ATOM_BUSINESS_ERROR", infraerrors.Reason(mapHCAtomBatchError(err)))
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

// Regression target: an HTTP 200 response is not an HC business success unless
// the envelope explicitly carries code=0.
func TestHCAtomBatchHTTPClient_RejectsMissingBusinessCode(t *testing.T) {
	client := NewHCAtomBatchHTTPClient(&http.Client{Transport: hcAtomRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return hcAtomHTTPResponse(http.StatusOK, `{"msg":"ok","data":{"taskId":"hc-task-missing-code","status":"PENDING"}}`), nil
	})})

	_, err := client.Get(context.Background(), "synthetic-key", "hc-task-missing-code")
	require.Error(t, err)
	require.Equal(t, "HC_ATOM_BUSINESS_ERROR", infraerrors.Reason(mapHCAtomBatchError(err)))
}

// Regression target: the HC API must not inherit the streaming batch client,
// whose unbounded overall timeout can leave a worker stuck reading a response.
func TestHCAtomBatchHTTPClient_DefaultProductionClientHasOverallAndPhaseTimeouts(t *testing.T) {
	client := NewHCAtomBatchHTTPClient(nil).client
	require.NotNil(t, client)
	require.Equal(t, 30*time.Second, client.Timeout)
	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok)
	require.Equal(t, 10*time.Second, transport.TLSHandshakeTimeout)
	require.Equal(t, 15*time.Second, transport.ResponseHeaderTimeout)
	require.NotNil(t, client.CheckRedirect)
}

// Regression target: a syntactically valid envelope plus unbounded trailing
// bytes must be rejected instead of making the worker read/retain arbitrary JSON.
func TestHCAtomBatchHTTPClient_RejectsOversizedJSONBody(t *testing.T) {
	body := `{"code":0,"data":{"taskId":"hc-task-large","status":"PENDING"}}` +
		strings.Repeat(" ", 1<<20)
	client := NewHCAtomBatchHTTPClient(&http.Client{Transport: hcAtomRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return hcAtomHTTPResponse(http.StatusOK, body), nil
	})})

	_, err := client.Get(context.Background(), "synthetic-key", "hc-task-large")
	require.Error(t, err)
}

func TestHCAtomBatchHTTPClient_MapsHTTPAndReadFailureMatrix(t *testing.T) {
	for _, tt := range []struct {
		name       string
		statusCode int
		wantReason string
	}{
		{name: "unauthorized", statusCode: http.StatusUnauthorized, wantReason: "HC_ATOM_AUTH_FAILED"},
		{name: "forbidden", statusCode: http.StatusForbidden, wantReason: "HC_ATOM_AUTH_FAILED"},
		{name: "rate_limited", statusCode: http.StatusTooManyRequests, wantReason: "HC_ATOM_RATE_LIMITED"},
		{name: "server_error", statusCode: http.StatusInternalServerError, wantReason: "HC_ATOM_UPSTREAM_ERROR"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := NewHCAtomBatchHTTPClient(&http.Client{Transport: hcAtomRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return hcAtomHTTPResponse(tt.statusCode, `{"code":0,"data":{}}`), nil
			})})
			_, err := client.Get(context.Background(), "synthetic-key", "hc-task-errors")
			require.Error(t, err)
			require.Equal(t, tt.wantReason, infraerrors.Reason(mapHCAtomBatchError(err)))
		})
	}

	t.Run("body_read_failure", func(t *testing.T) {
		client := NewHCAtomBatchHTTPClient(&http.Client{Transport: hcAtomRoundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: hcAtomFailingReadCloser{}}, nil
		})})
		_, err := client.Get(context.Background(), "synthetic-key", "hc-task-errors")
		require.Error(t, err)
		require.Equal(t, "HC_ATOM_UPSTREAM_ERROR", infraerrors.Reason(mapHCAtomBatchError(err)))
	})

	t.Run("request_timeout", func(t *testing.T) {
		client := NewHCAtomBatchHTTPClient(&http.Client{Transport: hcAtomRoundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, context.DeadlineExceeded
		})})
		_, err := client.Get(context.Background(), "synthetic-key", "hc-task-errors")
		require.Error(t, err)
		require.Equal(t, "HC_ATOM_UPSTREAM_ERROR", infraerrors.Reason(mapHCAtomBatchError(err)))
	})
}

func TestHCAtomBatchProvider_RejectsDisabledDolaAndMultipleItemsBeforeCreate(t *testing.T) {
	calls := 0
	client := NewHCAtomBatchHTTPClient(&http.Client{Transport: hcAtomRoundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return hcAtomHTTPResponse(http.StatusOK, `{"code":0,"data":{"taskId":"unexpected","status":"PENDING"}}`), nil
	})})
	cipher, account := hcAtomBatchTestAccount(t, "test")
	provider := NewHCAtomBatchImageProviderWithCredentialCipher(client, cipher)
	job := &BatchImageJob{BatchID: "imgbatch_validate"}

	_, err := provider.Submit(context.Background(), job, account, BatchImageInput{
		BatchID: "imgbatch_validate", Model: "dola-seedream-5.0-pro", Items: []BatchImageInputItem{{CustomID: "one", Prompt: "hero"}},
	})
	require.Error(t, err)
	require.Equal(t, "HC_ATOM_MODEL_UNSUPPORTED", infraerrors.Reason(err))

	_, err = provider.Submit(context.Background(), job, account, BatchImageInput{
		BatchID: "imgbatch_validate", Model: "seedream-5.0", Items: []BatchImageInputItem{{CustomID: "one", Prompt: "hero"}, {CustomID: "two", Prompt: "other"}},
	})
	require.Error(t, err)
	require.Equal(t, "HC_ATOM_SINGLE_ITEM_REQUIRED", infraerrors.Reason(err))
	require.Zero(t, calls)
}

func TestHCAtomBatchHTTPClient_ResultURLsDeduplicateAndMissingUsageDoesNotInventCount(t *testing.T) {
	client := NewHCAtomBatchHTTPClient(&http.Client{Transport: hcAtomRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return hcAtomHTTPResponse(http.StatusOK, `{"code":0,"data":{"taskId":"hc-task-urls","status":"SUCCESS","resultUrls":["https://8.8.8.8/a.png","https://8.8.8.8/a.png"],"resultUrl":"https://8.8.8.8/a.png"}}`), nil
	})})

	task, err := client.Get(context.Background(), "synthetic-key", "hc-task-urls")
	require.NoError(t, err)
	require.Equal(t, 0, task.ImageCount)
	require.Equal(t, []string{"https://8.8.8.8/a.png"}, hcAtomBatchResultURLs(task))
	status, err := mapHCAtomBatchState(task)
	require.NoError(t, err)
	require.Equal(t, BatchProviderStateSucceeded, status.InternalState)
}

// Regression target: camel/snake plural and singular result aliases must all
// contribute to one deterministic result set.
func TestHCAtomBatchHTTPClient_AggregatesDistinctResultURLAliasesAndUsage(t *testing.T) {
	client := NewHCAtomBatchHTTPClient(&http.Client{Transport: hcAtomRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return hcAtomHTTPResponse(http.StatusOK, `{"code":0,"data":{"taskId":"hc-task-aliases","status":"SUCCESS","resultUrls":["https://8.8.8.8/a.png"],"result_urls":["https://8.8.8.8/b.png"],"resultUrl":"https://8.8.8.8/c.png","result_url":"https://8.8.8.8/d.png","usage":{"imageCount":4}}}`), nil
	})})

	task, err := client.Get(context.Background(), "synthetic-key", "hc-task-aliases")
	require.NoError(t, err)
	require.Equal(t, 4, task.ImageCount)
	require.Equal(t, []string{
		"https://8.8.8.8/a.png",
		"https://8.8.8.8/b.png",
		"https://8.8.8.8/c.png",
		"https://8.8.8.8/d.png",
	}, hcAtomBatchResultURLs(task))
}

// Regression target: explicit upstream usage cannot disagree with the
// deduplicated result set and still become succeeded.
func TestHCAtomBatchProvider_RejectsUsageImageCountMismatch(t *testing.T) {
	_, err := mapHCAtomBatchState(&HCAtomBatchTask{
		Status: "SUCCESS", ImageCount: 3,
		ResultURLs: []string{"https://8.8.8.8/a.png", "https://8.8.8.8/b.png"},
	})
	require.Error(t, err)
	require.Equal(t, "HC_ATOM_USAGE_MISMATCH", infraerrors.Reason(err))
}

// Regression target: multiple images for the one HC request belong to one
// custom_id line with multiple inlineData parts, not duplicate custom_id lines.
func TestHCAtomBatchProvider_OpenResultAggregatesDistinctURLsIntoOneJSONLLine(t *testing.T) {
	cipher, account := hcAtomBatchTestAccount(t, "test")
	provider := NewHCAtomBatchImageProviderWithResultClientAndCredentialCipher(&fakeHCAtomBatchClient{task: &HCAtomBatchTask{
		TaskID: "hc-task-many", Status: "SUCCESS", ImageCount: 3,
		ResultURLs: []string{"https://8.8.8.8/a.png", "https://8.8.8.8/b.png"},
		ResultURL:  "https://8.8.8.8/c.png",
	}}, &http.Client{Transport: hcAtomRoundTripFunc(func(*http.Request) (*http.Response, error) {
		response := hcAtomHTTPResponse(http.StatusOK, "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01")
		response.Header.Set("Content-Type", "image/png")
		return response, nil
	})}, cipher)
	taskID, customID := "hc-task-many", "item_000001"

	r, _, err := provider.OpenResult(context.Background(), &BatchImageJob{ProviderJobName: &taskID, ProviderInputRef: &customID}, account)
	require.NoError(t, err)
	defer r.Close()
	body, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Len(t, strings.Split(strings.TrimSpace(string(body)), "\n"), 1)
	var line struct {
		CustomID string `json:"custom_id"`
		Response struct {
			Candidates []struct {
				Content struct {
					Parts []map[string]any `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		} `json:"response"`
	}
	require.NoError(t, json.Unmarshal(body, &line))
	require.Equal(t, customID, line.CustomID)
	require.Len(t, line.Response.Candidates, 1)
	require.Len(t, line.Response.Candidates[0].Content.Parts, 3)
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

// Regression target: validation applies to every redirect target, not just the
// provider-supplied first URL.
func TestHCAtomBatchProvider_ArchiveRejectsUnsafeRedirectTargetBeforeSecondHop(t *testing.T) {
	for _, target := range []string{
		"http://8.8.8.8/result.png",
		"https://user:pass@8.8.8.8/result.png",
		"https://8.8.8.8:8443/result.png",
		"https://127.0.0.1/result.png",
		"https://0x08080808/result.png",
		"https://8.8.8.8/result.png#fragment",
	} {
		t.Run(target, func(t *testing.T) {
			calls := 0
			client := &http.Client{Transport: hcAtomRoundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					response := hcAtomHTTPResponse(http.StatusFound, "")
					response.Header.Set("Location", target)
					return response, nil
				}
				response := hcAtomHTTPResponse(http.StatusOK, "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01")
				response.Header.Set("Content-Type", "image/png")
				return response, nil
			})}
			provider := NewHCAtomBatchImageProviderWithResultClient(&fakeHCAtomBatchClient{}, client)

			_, _, err := provider.archiveResultURL(context.Background(), "https://8.8.8.8/start.png")
			require.Error(t, err)
			require.Equal(t, 1, calls, "unsafe redirect must fail before the next transport hop")
		})
	}
}

// Regression target: matching three JPEG bytes or a RIFF/WEBP prefix is not a
// complete image validation boundary.
func TestHCAtomBatchProvider_ArchiveRejectsTruncatedJPEGAndWebP(t *testing.T) {
	for _, tt := range []struct {
		name     string
		mimeType string
		body     string
	}{
		{name: "jpeg", mimeType: "image/jpeg", body: string([]byte{0xff, 0xd8, 0xff, 0xe0})},
		{name: "webp", mimeType: "image/webp", body: "RIFF\x04\x00\x00\x00WEBP"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewHCAtomBatchImageProviderWithResultClient(&fakeHCAtomBatchClient{}, &http.Client{Transport: hcAtomRoundTripFunc(func(*http.Request) (*http.Response, error) {
				response := hcAtomHTTPResponse(http.StatusOK, tt.body)
				response.Header.Set("Content-Type", tt.mimeType)
				return response, nil
			})})
			_, _, err := provider.archiveResultURL(context.Background(), "https://8.8.8.8/result")
			require.Error(t, err)
		})
	}
}

// Regression target: JPEG dimensions must be bounded even when the compressed
// payload itself is small.
func TestHCAtomBatchProvider_ArchiveRejectsOversizedJPEGDimensions(t *testing.T) {
	var encoded bytes.Buffer
	require.NoError(t, jpeg.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 1, 1)), nil))
	for _, dimensions := range []struct {
		name          string
		width, height uint16
	}{
		{name: "axis_limit", width: 10001, height: 1},
		{name: "pixel_limit", width: 8000, height: 6000},
	} {
		t.Run(dimensions.name, func(t *testing.T) {
			oversized := hcAtomTestJPEGWithDimensions(t, encoded.Bytes(), dimensions.width, dimensions.height)
			provider := NewHCAtomBatchImageProviderWithResultClient(&fakeHCAtomBatchClient{}, &http.Client{Transport: hcAtomRoundTripFunc(func(*http.Request) (*http.Response, error) {
				response := hcAtomHTTPResponse(http.StatusOK, string(oversized))
				response.Header.Set("Content-Type", "image/jpeg")
				return response, nil
			})})

			_, _, err := provider.archiveResultURL(context.Background(), "https://8.8.8.8/oversized.jpg")
			require.Error(t, err)
		})
	}
}

// Regression target: a VP8X canvas header without an actual VP8/VP8L image
// chunk is metadata, not an archivable WebP image.
func TestHCAtomBatchProvider_ArchiveRejectsHeaderOnlyWebP(t *testing.T) {
	headerOnly := hcAtomTestWebPVP8X(t, nil, 1, 1)
	provider := NewHCAtomBatchImageProviderWithResultClient(&fakeHCAtomBatchClient{}, &http.Client{Transport: hcAtomRoundTripFunc(func(*http.Request) (*http.Response, error) {
		response := hcAtomHTTPResponse(http.StatusOK, string(headerOnly))
		response.Header.Set("Content-Type", "image/webp")
		return response, nil
	})})

	_, _, err := provider.archiveResultURL(context.Background(), "https://8.8.8.8/header-only.webp")
	require.Error(t, err)
}

func TestHCAtomBatchProvider_ArchiveAcceptsCompleteBoundedJPEGAndWebP(t *testing.T) {
	var jpegBytes bytes.Buffer
	require.NoError(t, jpeg.Encode(&jpegBytes, image.NewRGBA(image.Rect(0, 0, 1, 1)), nil))
	for _, tt := range []struct {
		name     string
		mimeType string
		data     []byte
	}{
		{name: "jpeg", mimeType: "image/jpeg", data: jpegBytes.Bytes()},
		{name: "webp", mimeType: "image/webp", data: hcAtomTestValidWebP(t)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewHCAtomBatchImageProviderWithResultClient(&fakeHCAtomBatchClient{}, &http.Client{Transport: hcAtomRoundTripFunc(func(*http.Request) (*http.Response, error) {
				response := hcAtomHTTPResponse(http.StatusOK, string(tt.data))
				response.Header.Set("Content-Type", tt.mimeType)
				return response, nil
			})})

			encoded, mimeType, err := provider.archiveResultURL(context.Background(), "https://8.8.8.8/result")
			require.NoError(t, err)
			require.NotEmpty(t, encoded)
			require.Equal(t, tt.mimeType, mimeType)
		})
	}
}

// Regression target: provider-owned JSONL must never exceed the 16 MiB scanner
// boundary after raw bytes expand to base64.
func TestHCAtomBatchProvider_OpenResultRejectsRawImageThatCannotFitJSONLScanner(t *testing.T) {
	oversized := make([]byte, 12<<20)
	copy(oversized, []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01"))
	resultTransport := hcAtomRoundTripFunc(func(*http.Request) (*http.Response, error) {
		response := hcAtomHTTPResponse(http.StatusOK, string(oversized))
		response.Header.Set("Content-Type", "image/png")
		return response, nil
	})
	cipher, account := hcAtomBatchTestAccount(t, "test")
	provider := NewHCAtomBatchImageProviderWithResultClientAndCredentialCipher(&fakeHCAtomBatchClient{task: &HCAtomBatchTask{
		TaskID: "hc-task-large-result", Status: "SUCCESS", ResultURL: "https://8.8.8.8/large.png",
	}}, &http.Client{Transport: resultTransport}, cipher)
	taskID, customID := "hc-task-large-result", "item_000001"

	_, _, err := provider.OpenResult(context.Background(), &BatchImageJob{ProviderJobName: &taskID, ProviderInputRef: &customID}, account)
	require.Error(t, err)
}

type hcAtomRoundTripFunc func(*http.Request) (*http.Response, error)

func (f hcAtomRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

type hcAtomFailingReadCloser struct{}

func (hcAtomFailingReadCloser) Read([]byte) (int, error) {
	return 0, errors.New("synthetic HC body read failure")
}
func (hcAtomFailingReadCloser) Close() error { return nil }

func hcAtomHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func hcAtomTestJPEGWithDimensions(t *testing.T, source []byte, width, height uint16) []byte {
	t.Helper()
	out := append([]byte(nil), source...)
	for i := 0; i+9 < len(out); i++ {
		if out[i] != 0xff || (out[i+1] != 0xc0 && out[i+1] != 0xc2) {
			continue
		}
		out[i+5], out[i+6] = byte(height>>8), byte(height)
		out[i+7], out[i+8] = byte(width>>8), byte(width)
		return out
	}
	t.Fatal("JPEG SOF marker not found")
	return nil
}

func hcAtomTestWebPVP8X(t *testing.T, imageChunks []byte, width, height uint32) []byte {
	t.Helper()
	require.Greater(t, width, uint32(0))
	require.Greater(t, height, uint32(0))
	var out bytes.Buffer
	out.WriteString("RIFF")
	out.Write(make([]byte, 4))
	out.WriteString("WEBP")
	out.WriteString("VP8X")
	require.NoError(t, binary.Write(&out, binary.LittleEndian, uint32(10)))
	out.Write([]byte{0, 0, 0, 0})
	for _, value := range []uint32{width - 1, height - 1} {
		out.Write([]byte{byte(value), byte(value >> 8), byte(value >> 16)})
	}
	out.Write(imageChunks)
	data := out.Bytes()
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(data)-8))
	return data
}

func hcAtomTestValidWebP(t *testing.T) []byte {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString("UklGRrIBAABXRUJQVlA4TKUBAAAvSsAYAA8w//M///MfeJAkbXvaSG7m8Q3GfYSBJekwQztm/IcZlgwnmWImn2BK7aFmBtnVir6q//8VOkFE/xm4baTIu8c48ArEo6+B3zFKYln3pqClSCKX0begFTAXFOLXHSyF8cCNcZEG4OywuA4KVVfJCiArU7GAgJI8+lJP/OKMT/fBAjevg1cYB7YVkFuWga2lyPi5I0HFy5YTpWIHg0RZpkniRVW9odHAKOwosWuOGdxIyn2OvaCDvhg/we6TwadPBPbqBV58MsLmMJ8yZnOWk8SRz4N+QoyPL+MnamzMvcE1rHNEr91F9GKZPVUcS9w7PhhH36suB9qPeYb/oLk6cuTiJ0wOK3m5h1cKjW6EVZCYMK7dxcKCBdgP9HkKr9gkAO2P8GKZGWVdIAatQa+1IDpt6qyorVwdy01xdW8Jkfk6xjEXmVQQ+HQdFr6OKhIN34dXWq0+0qr6EJSCeeVLH9+gvGTLyqM65PQ44ihzlTXxQKjKbAvshXgir7Lil9w4L2bvMycmjQcqXaMCO6BlY28i+FOLzbfI1vEqxAhotocAAA==")
	require.NoError(t, err)
	return data
}

type hcAtomBatchGroupRepo struct{ groups map[int64]*Group }

func (r *hcAtomBatchGroupRepo) GetByIDLite(_ context.Context, id int64) (*Group, error) {
	if group, ok := r.groups[id]; ok {
		return group, nil
	}
	return nil, ErrGroupNotFound
}

type fakeHCAtomBatchClient struct {
	task *HCAtomBatchTask
	err  error
}

func hcAtomBatchTestAccount(t *testing.T, apiKey string) (HCAtomCredentialCipher, *Account) {
	t.Helper()
	cipher, err := NewHCAtomCredentialCipher(strings.Repeat("77", 32))
	require.NoError(t, err)
	credentials, err := ProtectHCAtomAccountCredentials(nil, map[string]any{"api_key": apiKey}, cipher)
	require.NoError(t, err)
	return cipher, &Account{Platform: PlatformHCAtom, Type: AccountTypeAPIKey, Credentials: credentials}
}

func (f *fakeHCAtomBatchClient) Create(context.Context, string, string, HCAtomBatchCreateRequest) (*HCAtomBatchTask, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.task, nil
}
func (f *fakeHCAtomBatchClient) Get(context.Context, string, string) (*HCAtomBatchTask, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.task, nil
}
func (f *fakeHCAtomBatchClient) Delete(context.Context, string, string) error { return nil }
