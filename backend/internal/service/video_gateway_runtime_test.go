package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
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
		if !strings.Contains(string(body), `"model":"doubao-seedance-2-0-260128"`) {
			t.Fatalf("request body = %s", body)
		}
		return &http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"bad ` + secret + `"}}`)), Header: make(http.Header)}, nil
	})}
	adapter := NewSeedanceAdapter(client, "https://ark.cn-beijing.volces.com", secret)
	_, err := adapter.Create(context.Background(), VideoCreateRequest{Prompt: "x", Duration: 4, Resolution: "720p"})
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Fatalf("unredacted or missing error: %v", err)
	}
}
