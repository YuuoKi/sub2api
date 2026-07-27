package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func TestHCAtomV3CreateUsesFixedEndpointBearerAndV3Payload(t *testing.T) {
	secret := "hc-test-secret"
	client := &http.Client{Transport: videoRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.URL.String() != "https://api-aigc.fzyinghe.com/v3/video/tasks" {
			t.Fatalf("request=%s %s", r.Method, r.URL)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+secret {
			t.Fatalf("authorization=%q", got)
		}
		body, _ := io.ReadAll(r.Body)
		want := `{"model":"doubao-seedance-2-0-260128","content":[{"type":"text","text":"a paper boat"},{"type":"image_url","image_url":{"url":"https://assets.example.test/boat.png"}}],"ratio":"16:9","generate_audio":true,"return_last_frame":true,"watermark":false,"duration":8,"resolution":"1080p"}`
		if string(body) != want {
			t.Fatalf("body=%s", body)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"hc-1","status":"queued"}`)), Header: make(http.Header)}, nil
	})}
	adapter := NewHCAtomV3Adapter(client, HCAtomSeedanceV3BaseURL, secret)
	created, err := adapter.Create(context.Background(), VideoCreateRequest{
		Model: HCAtomSeedanceV3PublicModel, Content: []VideoContentItem{{Type: "text", Text: "a paper boat"}, {Type: "image_url", ImageURL: &VideoContentURL{URL: "https://assets.example.test/boat.png"}}},
		Ratio: "16:9", GenerateAudio: true, ReturnLastFrame: true, Watermark: false, Duration: 8, Resolution: "1080p",
	})
	if err != nil || created.UpstreamTaskID != "hc-1" || created.Status != VideoStatusSubmitted {
		t.Fatalf("created=%#v err=%v", created, err)
	}
}

func TestHCAtomV3PollCancelAndRejectUnsafeInputs(t *testing.T) {
	requests := 0
	client := &http.Client{Transport: videoRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		switch requests {
		case 1:
			if r.Method != http.MethodGet || r.URL.Path != "/v3/video/tasks/hc-1" {
				t.Fatalf("poll=%s %s", r.Method, r.URL)
			}
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"hc-1","status":"succeeded","content":{"video_url":"https://cdn.example.test/v.mp4","last_frame_url":"https://cdn.example.test/v.jpg"},"usage":{"completion_tokens":12,"total_tokens":15}}`)), Header: make(http.Header)}, nil
		case 2:
			if r.Method != http.MethodDelete || r.URL.Path != "/v3/video/tasks/hc-1" {
				t.Fatalf("cancel=%s %s", r.Method, r.URL)
			}
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"hc-1","status":"cancelled"}`)), Header: make(http.Header)}, nil
		default:
			return nil, errors.New("unexpected network call")
		}
	})}
	adapter := NewHCAtomV3Adapter(client, HCAtomSeedanceV3BaseURL, "secret")
	polled, err := adapter.Poll(context.Background(), "hc-1")
	if err != nil || polled.Status != VideoStatusSucceeded || polled.CompletionTokens == nil || *polled.CompletionTokens != 12 || polled.LastFrameURL == "" {
		t.Fatalf("poll=%#v err=%v", polled, err)
	}
	cancelled, err := adapter.Cancel(context.Background(), "hc-1")
	if err != nil || cancelled.Status != VideoStatusCancelled {
		t.Fatalf("cancel=%#v err=%v", cancelled, err)
	}
	if err := ValidateHCAtomV3Request(VideoCreateRequest{Content: []VideoContentItem{{Type: "image_url", ImageURL: &VideoContentURL{URL: "data:image/png;base64,AAAA"}}}}); err == nil {
		t.Fatal("base64 accepted")
	}
	if err := ValidateHCAtomV3Request(VideoCreateRequest{Content: []VideoContentItem{{Type: "video_url", VideoURL: &VideoContentURL{URL: "https://127.0.0.1/v.mp4"}}}}); err == nil {
		t.Fatal("private URL accepted")
	}
	if err := ValidateHCAtomV3Request(VideoCreateRequest{Content: []VideoContentItem{{Type: "audio_url", AudioURL: &VideoContentURL{URL: "asset://asset-123"}}}}); err != nil {
		t.Fatalf("asset URL rejected: %v", err)
	}
}

func TestHCAtomV3AdapterRejectsNonAllowlistedOriginBeforeNetworkAndRedactsSecret(t *testing.T) {
	secret := "hc-secret-123"
	calls := 0
	adapter := NewHCAtomV3Adapter(&http.Client{Transport: videoRoundTripperFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New(secret) })}, "http://api-aigc.fzyinghe.com", secret)
	_, err := adapter.Create(context.Background(), VideoCreateRequest{Prompt: "x"})
	if err == nil || strings.Contains(err.Error(), secret) || calls != 0 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}

func TestHCAtomV3IsAnEnabledFixedVideoProvider(t *testing.T) {
	spec, ok := lookupVideoProvider(HCAtomSeedanceV3Provider)
	if !ok || !spec.AdapterReady || spec.DefaultBaseURL != HCAtomSeedanceV3BaseURL || spec.DefaultModel != HCAtomSeedanceV3PublicModel {
		t.Fatalf("provider=%#v found=%v", spec, ok)
	}
}

func TestHCAtomV3CreateContractKeepsLegacyPromptAndAllowsV3Fields(t *testing.T) {
	repo := &workerRepoStub{provider: VideoProviderAccount{ID: 10, GroupID: 9, Provider: HCAtomSeedanceV3Provider, Enabled: true, DefaultModel: HCAtomSeedanceV3PublicModel}}
	cfg := &config.Config{VideoGateway: config.VideoGatewayConfig{WorkerEnabled: true, HCAtomV3DispatchEnabled: true, SeedanceCNYPerMillionTokens: 2, USDCNYExchangeRate: 7, TinyRealEstimateCNY: .7, TinyRealMaximumCNY: 1.4}}
	created, err := NewVideoGatewayService(repo, NewSingleSmokeAuthorization(true), cfg, &videoAuthInvalidatorStub{}, &videoBillingInvalidatorStub{}).CreateTask(context.Background(), VideoTaskCreateCommand{Scope: VideoTaskScope{UserID: 1, APIKeyID: 2, GroupID: 9}, ProviderAccountID: 10, Prompt: "legacy prompt", Duration: 8, Resolution: "1080p", Ratio: "16:9", GenerateAudio: true})
	if err != nil || created == nil || created.Model != HCAtomSeedanceV3PublicModel || created.DurationSeconds != 8 || len(created.CreateRequest.Content) != 1 || created.CreateRequest.Content[0].Text != "legacy prompt" {
		t.Fatalf("created=%#v err=%v", created, err)
	}
}

type hcAtomClientStub struct {
	cancel  *VideoProviderTask
	err     error
	creates int
}

func (s *hcAtomClientStub) Create(context.Context, VideoCreateRequest) (*VideoProviderTask, error) {
	s.creates++
	return nil, s.err
}
func (s *hcAtomClientStub) Poll(context.Context, string) (*VideoProviderTask, error) { return nil, nil }
func (s *hcAtomClientStub) Cancel(context.Context, string) (*VideoProviderTask, error) {
	return s.cancel, s.err
}

func TestHCAtomDispatchedCancelFinalizesOnlyAfterConfirmedUpstreamCancel(t *testing.T) {
	for _, tc := range []struct {
		name          string
		client        *hcAtomClientStub
		wantFinalized int
	}{
		{"confirmed", &hcAtomClientStub{cancel: &VideoProviderTask{Status: VideoStatusCancelled}}, 1},
		{"delete_failure", &hcAtomClientStub{err: errors.New("timeout")}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &workerRepoStub{task: &VideoTask{ID: 7, APIKeyID: 2, GroupID: 9, ProviderAccountID: 10, Provider: HCAtomSeedanceV3Provider, Status: VideoStatusSubmitted, UpstreamTaskID: "hc-7", Version: 3, CreatedBy: 1, ReservationState: VideoReservationReserved, ReservedCostUSD: .2}, provider: VideoProviderAccount{ID: 10, GroupID: 9, Provider: HCAtomSeedanceV3Provider, BaseURL: HCAtomSeedanceV3BaseURL, EncryptedAPIKey: "cipher"}}
			svc := NewVideoGatewayService(repo, nil, &config.Config{}, &videoAuthInvalidatorStub{}, &videoBillingInvalidatorStub{})
			svc.ConfigureProviderClientFactory(keyDecryptStub{}, func(string, string, string) VideoProviderClient { return tc.client })
			_, _ = svc.CancelTask(context.Background(), 7, VideoTaskScope{UserID: 1, APIKeyID: 2, GroupID: 9})
			if len(repo.finalized) != tc.wantFinalized {
				t.Fatalf("finalized=%#v", repo.finalized)
			}
			if tc.wantFinalized == 1 && repo.finalized[0].Settlement != VideoSettlementRelease {
				t.Fatalf("finalization=%#v", repo.finalized[0])
			}
		})
	}
}

func TestVideoWorkerTransportFailureDoesNotInventIDOrRetry(t *testing.T) {
	repo := &workerRepoStub{begin: true, task: &VideoTask{ID: 7, GroupID: 9, ProviderAccountID: 10, Provider: HCAtomSeedanceV3Provider, Status: VideoStatusQueued, Version: 1, CreatedBy: 1, ReservationState: VideoReservationReserved, ReservedCostUSD: .2}, provider: VideoProviderAccount{ID: 10, GroupID: 9, Provider: HCAtomSeedanceV3Provider, BaseURL: HCAtomSeedanceV3BaseURL, EncryptedAPIKey: "cipher"}}
	client := &hcAtomClientStub{err: &VideoProviderTransportError{err: errors.New("connection reset")}}
	w := NewVideoGatewayWorker(repo, keyDecryptStub{}, func(string, string, string) VideoProviderClient { return client }, &videoAuthInvalidatorStub{}, &videoBillingInvalidatorStub{}, &config.Config{}, NewSingleSmokeAuthorization(true))
	if err := w.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if client.creates != 1 || len(repo.finalized) != 0 || repo.task.Status != VideoStatusReviewRequired || repo.task.UpstreamTaskID != "" {
		t.Fatalf("creates=%d finalized=%#v task=%#v", client.creates, repo.finalized, repo.task)
	}
}
