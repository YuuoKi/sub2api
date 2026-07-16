package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type videoRoundTripperFunc func(*http.Request) (*http.Response, error)

func (f videoRoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSingleSmokeAuthorizationDefaultsDeniedAndIsConsumedOnce(t *testing.T) {
	denied := NewSingleSmokeAuthorization(false)
	if err := denied.Consume(); !errors.Is(err, ErrVideoRealDispatchDenied) {
		t.Fatalf("default authorization error = %v", err)
	}
	authorized := NewSingleSmokeAuthorization(true)
	if err := authorized.Consume(); err != nil {
		t.Fatalf("first consume: %v", err)
	}
	if err := authorized.Consume(); !errors.Is(err, ErrVideoRealDispatchConsumed) {
		t.Fatalf("second consume error = %v", err)
	}
}

func TestVideoActualUSDComputesProviderCostForSettlementCapDecision(t *testing.T) {
	cfg := &config.Config{VideoGateway: config.VideoGatewayConfig{SeedanceCNYPerMillionTokens: 2, USDCNYExchangeRate: 7, TinyRealMaximumCNY: 0.1}}
	got, err := videoActualUSD(1_000_000, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got != 0.28571429 {
		t.Fatalf("actual USD = %.8f", got)
	}
	for _, bad := range []*config.Config{nil, {}, {VideoGateway: config.VideoGatewayConfig{SeedanceCNYPerMillionTokens: 2}}} {
		if _, err := videoActualUSD(1, bad); err == nil {
			t.Fatal("expected incomplete pricing rejection")
		}
	}
}

func TestSeedanceAdapterNilClientUsesFiniteTimeout(t *testing.T) {
	adapter := NewSeedanceAdapter(nil, "https://ark.cn-beijing.volces.com", "synthetic-key")
	if adapter.client == nil || adapter.client == http.DefaultClient || adapter.client.Timeout != 30*time.Second {
		t.Fatalf("client=%#v", adapter.client)
	}
}

func TestVideoMaximumUSDFailsClosedWithoutServerCapAndFX(t *testing.T) {
	valid := &config.Config{VideoGateway: config.VideoGatewayConfig{SeedanceCNYPerMillionTokens: 2, USDCNYExchangeRate: 7, TinyRealEstimateCNY: 0.7, TinyRealMaximumCNY: 1.4}}
	got, err := videoMaximumUSD(valid)
	if err != nil || got != 0.2 {
		t.Fatalf("got=%v err=%v", got, err)
	}
	for _, bad := range []*config.Config{nil, {}, {VideoGateway: config.VideoGatewayConfig{USDCNYExchangeRate: 7}}, {VideoGateway: config.VideoGatewayConfig{SeedanceCNYPerMillionTokens: 2, USDCNYExchangeRate: 7, TinyRealEstimateCNY: 2, TinyRealMaximumCNY: 1}}} {
		if _, err := videoMaximumUSD(bad); err == nil {
			t.Fatal("expected fail-closed pricing")
		}
	}
}

func TestSeedancePollMapsRunningAndValidatesAssets(t *testing.T) {
	responses := []string{
		`{"id":"up-1","status":"running"}`,
		`{"id":"up-1","status":"succeeded","content":{"video_url":"https://cdn.example.test/a.mp4"},"last_frame_url":"https://cdn.example.test/a.jpg","usage":{"completion_tokens":245025}}`,
		`{"id":"up-1","status":"succeeded","content":{"video_url":"http://127.0.0.1/a.mp4"},"usage":{"completion_tokens":1}}`,
	}
	client := &http.Client{Transport: videoRoundTripperFunc(func(*http.Request) (*http.Response, error) {
		body := responses[0]
		responses = responses[1:]
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	a := NewSeedanceAdapter(client, "https://ark.cn-beijing.volces.com", "synthetic-key")
	running, err := a.Poll(context.Background(), "up-1")
	if err != nil || running.Status != VideoStatusRunning {
		t.Fatalf("running=%#v err=%v", running, err)
	}
	done, err := a.Poll(context.Background(), "up-1")
	if err != nil || done.CompletionTokens == nil || *done.CompletionTokens != 245025 {
		t.Fatalf("done=%#v err=%v", done, err)
	}
	if _, err := a.Poll(context.Background(), "up-1"); err == nil {
		t.Fatal("unsafe asset URL accepted")
	}
}

func TestValidateTinyRealContractRequiresTextOnlyFourSeconds720p(t *testing.T) {
	valid := VideoCreateRequest{Prompt: "a paper boat", Duration: 4, Resolution: "720p"}
	if err := ValidateTinyRealContract(valid); err != nil {
		t.Fatalf("valid tiny_real: %v", err)
	}
	for _, input := range []VideoCreateRequest{
		{Prompt: "", Duration: 4, Resolution: "720p"},
		{Prompt: "x", Duration: 5, Resolution: "720p"},
		{Prompt: "x", Duration: 4, Resolution: "1080p"},
		{Prompt: "x", Duration: 4, Resolution: "720p", ReferenceImageURL: "https://example.test/x.png"},
	} {
		if err := ValidateTinyRealContract(input); err == nil {
			t.Fatalf("expected rejection for %#v", input)
		}
	}
}

func TestSeedanceAdapterRejectsNonHTTPSAndNonArkHostsBeforeNetwork(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: videoRoundTripperFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New("network must not be reached")
	})}
	for _, baseURL := range []string{"http://ark.cn-beijing.volces.com", "https://evil.example.com"} {
		adapter := NewSeedanceAdapter(client, baseURL, "secret-value")
		_, err := adapter.Create(context.Background(), VideoCreateRequest{Prompt: "x", Duration: 4, Resolution: "720p"})
		if err == nil {
			t.Fatalf("expected base URL rejection for %s", baseURL)
		}
	}
	if calls != 0 {
		t.Fatalf("network calls = %d, want 0", calls)
	}
}

func TestSeedanceAdapterUsesConfirmedModelAndRedactsSecrets(t *testing.T) {
	secret := "ark-secret-123"
	client := &http.Client{Transport: videoRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(r.Body)
		want := `{"model":"doubao-seedance-2-0-260128","content":[{"type":"text","text":"x"}],"duration":4,"resolution":"720p","return_last_frame":true}`
		if string(body) != want {
			t.Fatalf("request body = %s, want %s", body, want)
		}
		return &http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"bad ` + secret + `"}}`)), Header: make(http.Header)}, nil
	})}
	adapter := NewSeedanceAdapter(client, "https://ark.cn-beijing.volces.com", secret)
	_, err := adapter.Create(context.Background(), VideoCreateRequest{Prompt: "x", Duration: 4, Resolution: "720p", ReturnLastFrame: true})
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Fatalf("unredacted or missing error: %v", err)
	}
}
