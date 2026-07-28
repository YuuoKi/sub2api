package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type hcAtomGatewayRoundTripFunc func(*http.Request) (*http.Response, error)

func (f hcAtomGatewayRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestHCAtomCatalogFixedRoutes(t *testing.T) {
	tests := []struct {
		capability   string
		model        string
		upstream     string
		kind         string
		endpoint     string
		pricing      string
		capabilities PublicModelCapabilities
	}{
		{HCAtomCapabilityChat, "gpt-5.6-sol", "gpt-5.6-sol", "text", "https://ai-aigc.fzyinghe.com/v1/chat/completions", HCAtomPricingChannelToken, PublicModelCapabilities{TaskMode: "sync"}},
		{HCAtomCapabilityChat, "gemini-3-flash-preview", "gemini-3-flash-preview", "text", "https://ai-aigc.fzyinghe.com/v1/chat/completions", HCAtomPricingChannelToken, PublicModelCapabilities{TaskMode: "sync", InputImage: true}},
		{HCAtomCapabilityMessages, "claude-opus-4-6", "claude-opus-4-6", "text", "https://ai-aigc.fzyinghe.com/v1/messages", HCAtomPricingChannelToken, PublicModelCapabilities{TaskMode: "sync", InputImage: true}},
		{HCAtomCapabilityImageSync, "seedream-5.0", "seedream-5.0", "image", "https://ai-aigc.fzyinghe.com/v1/images/generations", HCAtomPricingGroupImage, PublicModelCapabilities{TaskMode: "sync", ReferenceImages: true, AspectRatio: true, Resolution: true}},
		{HCAtomCapabilityImageAsync, HCAtomImageAsyncI2IModel, HCAtomImageAsyncI2IModel, "image", "https://api-aigc.fzyinghe.com/image/generation/tasks", HCAtomPricingGroupImage, PublicModelCapabilities{TaskMode: "async", InputImage: true, ReferenceImages: true, AspectRatio: true, Resolution: true}},
		{HCAtomCapabilityImageAsync, HCAtomImageAsyncT2IModel, HCAtomImageAsyncT2IModel, "image", "https://api-aigc.fzyinghe.com/image/generation/tasks", HCAtomPricingGroupImage, PublicModelCapabilities{TaskMode: "async", AspectRatio: true, Resolution: true}},
		{HCAtomCapabilityVideoV1, HCAtomVideoV1PublicModel, HCAtomVideoV1PublicModel, "video", "https://api-aigc.fzyinghe.com/video/generation/tasks", HCAtomPricingGroupVideo, PublicModelCapabilities{TaskMode: "async", InputImage: true, ReferenceImages: true, AspectRatio: true, DurationSeconds: true, Audio: true}},
		{HCAtomCapabilityVideoV3, HCAtomSeedanceV3PublicModel, HCAtomSeedanceV3Model, "video", "https://api-aigc.fzyinghe.com/v3/video/tasks", HCAtomPricingGroupVideo, PublicModelCapabilities{TaskMode: "async", InputImage: true, FirstFrame: true, LastFrame: true, ReferenceImages: true, AspectRatio: true, DurationSeconds: true, Resolution: true, Audio: true}},
	}
	require.Len(t, HCAtomModelCatalog(), len(tests))
	for _, test := range tests {
		t.Run(test.capability+"/"+test.model, func(t *testing.T) {
			spec, ok := LookupHCAtomModel(test.capability, test.model)
			require.True(t, ok)
			require.True(t, spec.Enabled)
			require.Equal(t, test.model, spec.PublicModel)
			require.Equal(t, test.upstream, spec.UpstreamModel)
			require.Equal(t, test.kind, spec.Kind)
			require.Equal(t, test.endpoint, spec.Origin+spec.Path)
			require.Equal(t, test.pricing, spec.PricingPolicy)
			require.Equal(t, test.capabilities, spec.PublicCapabilities)

			public, ok := lookupPublicModelForTest(PublicHCAtomModels(test.kind), test.model)
			require.True(t, ok)
			require.Equal(t, test.model, public.ID)
			require.Equal(t, spec.DisplayName, public.DisplayName)
			require.Equal(t, test.kind, public.Kind)
			require.Equal(t, test.capabilities, public.Capabilities)
		})
	}
}

func lookupPublicModelForTest(models []PublicModel, model string) (PublicModel, bool) {
	for _, item := range models {
		if item.Model == model {
			return item, true
		}
	}
	return PublicModel{}, false
}

func TestHCAtomFixedClientUsesOnlyCatalogRouteAndRedacts(t *testing.T) {
	const secret = "hc-secret-value"
	tests := []struct {
		name       string
		capability string
		model      string
		wantURL    string
	}{
		{"gpt chat", HCAtomCapabilityChat, "gpt-5.6-sol", "https://ai-aigc.fzyinghe.com/v1/chat/completions"},
		{"gemini chat", HCAtomCapabilityChat, "gemini-3-flash-preview", "https://ai-aigc.fzyinghe.com/v1/chat/completions"},
		{"claude messages", HCAtomCapabilityMessages, "claude-opus-4-6", "https://ai-aigc.fzyinghe.com/v1/messages"},
		{"sync image", HCAtomCapabilityImageSync, "seedream-5.0", "https://ai-aigc.fzyinghe.com/v1/images/generations"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &http.Client{Transport: hcAtomGatewayRoundTripFunc(func(request *http.Request) (*http.Response, error) {
				require.Equal(t, http.MethodPost, request.Method)
				require.Equal(t, test.wantURL, request.URL.String())
				require.Equal(t, "Bearer "+secret, request.Header.Get("Authorization"))
				body, err := io.ReadAll(request.Body)
				require.NoError(t, err)
				require.Contains(t, string(body), `"`+test.model+`"`)
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"usage":{"prompt_tokens":2,"completion_tokens":3}}`))}, nil
			})}
			response, err := NewHCAtomFixedClient(client).Do(context.Background(), test.capability, test.model, secret, []byte(`{"model":"client-value","messages":[]}`))
			require.NoError(t, err)
			data, err := readHCAtomResponse(response, secret)
			require.NoError(t, err)
			require.Equal(t, ClaudeUsage{InputTokens: 2, OutputTokens: 3}, hcAtomUsageFromJSON(data))
		})
	}

	t.Run("transport error is redacted", func(t *testing.T) {
		client := &http.Client{Transport: hcAtomGatewayRoundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("dial failed with " + secret)
		})}
		_, err := NewHCAtomFixedClient(client).Do(context.Background(), HCAtomCapabilityChat, "gpt-5.6-sol", secret, []byte(`{"model":"gpt-5.6-sol"}`))
		require.Error(t, err)
		require.NotContains(t, err.Error(), secret)
	})

	t.Run("response credential echo is rejected without echoing it", func(t *testing.T) {
		response := &http.Response{StatusCode: http.StatusBadGateway, Body: io.NopCloser(strings.NewReader(secret))}
		_, err := readHCAtomResponse(response, secret)
		require.Error(t, err)
		require.NotContains(t, err.Error(), secret)
	})

	t.Run("unknown model never reaches transport", func(t *testing.T) {
		calls := 0
		client := &http.Client{Transport: hcAtomGatewayRoundTripFunc(func(*http.Request) (*http.Response, error) {
			calls++
			return nil, nil
		})}
		_, err := NewHCAtomFixedClient(client).Do(context.Background(), HCAtomCapabilityChat, "not-in-catalog", secret, []byte(`{}`))
		require.Error(t, err)
		require.Zero(t, calls)
	})
}

func TestHCAtomUsageFromSSE(t *testing.T) {
	data := []byte("data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\ndata: {\"usage\":{\"prompt_tokens\":7,\"completion_tokens\":4}}\n\ndata: [DONE]\n\n")
	require.Equal(t, ClaudeUsage{InputTokens: 7, OutputTokens: 4}, hcAtomUsageFromSSE(data))
}

func TestHCAtomUsageFromAnthropicSSE(t *testing.T) {
	data := []byte("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":9,\"output_tokens\":0}}}\n\n" +
		"event: message_delta\ndata: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":5}}\n\n")
	require.Equal(t, ClaudeUsage{InputTokens: 9, OutputTokens: 5}, hcAtomUsageFromSSE(data))
}

func TestPublicHCAtomModelsExposeOnlySub2Contract(t *testing.T) {
	models := PublicHCAtomModels("image")
	require.Len(t, models, 3)
	data, err := json.Marshal(map[string]any{"object": "list", "data": models})
	require.NoError(t, err)
	payload := string(data)
	require.NotContains(t, payload, "fzyinghe.com")
	require.NotContains(t, payload, "provider")
	require.NotContains(t, payload, "base_url")
	require.NotContains(t, payload, "upstream")
	require.NotContains(t, payload, "pricing")
	require.NotContains(t, payload, "origin")
	require.NotContains(t, payload, "path")
	require.Contains(t, payload, `"task_mode":"sync"`)
	require.Contains(t, payload, `"task_mode":"async"`)
}

func TestHCAtomCatalogPricingPoliciesFailClosed(t *testing.T) {
	groupID := int64(77)
	price := 0.1
	fullImagePrices := func() *Group {
		return &Group{
			ID:           groupID,
			ImagePrice1K: &price,
			ImagePrice2K: &price,
			ImagePrice4K: &price,
		}
	}
	tests := []struct {
		name       string
		capability string
		model      string
		group      *Group
		resolver   *ModelPricingResolver
		want       bool
	}{
		{
			name:       "text requires channel token pricing",
			capability: HCAtomCapabilityChat,
			model:      "gpt-5.6-sol",
			group:      &Group{ID: groupID},
			want:       false,
		},
		{
			name:       "text accepts usable channel token pricing",
			capability: HCAtomCapabilityChat,
			model:      "gpt-5.6-sol",
			group:      &Group{ID: groupID},
			resolver:   newOpenAITokenImageChannelPricingResolverForTest(t, groupID, "gpt-5.6-sol"),
			want:       true,
		},
		{
			name:       "sync image accepts group image pricing without token resolver",
			capability: HCAtomCapabilityImageSync,
			model:      "seedream-5.0",
			group: func() *Group {
				group := fullImagePrices()
				group.AllowImageGeneration = true
				return group
			}(),
			want: true,
		},
		{
			name:       "sync image rejects incomplete group prices",
			capability: HCAtomCapabilityImageSync,
			model:      "seedream-5.0",
			group: &Group{
				ID:                   groupID,
				AllowImageGeneration: true,
				ImagePrice1K:         &price,
				ImagePrice2K:         &price,
			},
			want: false,
		},
		{
			name:       "async image rejects disabled group capability",
			capability: HCAtomCapabilityImageAsync,
			model:      "wan2.5-i2i-preview",
			group:      fullImagePrices(),
			want:       false,
		},
		{
			name:       "async image accepts group image pricing",
			capability: HCAtomCapabilityImageAsync,
			model:      "wan2.5-i2i-preview",
			group: func() *Group {
				group := fullImagePrices()
				group.AllowBatchImageGeneration = true
				return group
			}(),
			want: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spec, ok := LookupHCAtomModel(test.capability, test.model)
			require.True(t, ok)
			service := &GatewayService{resolver: test.resolver}
			apiKey := &APIKey{GroupID: &groupID, Group: test.group}
			require.Equal(t, test.want, service.hcAtomPublicPricingConfigured(context.Background(), apiKey, spec))
		})
	}
}
