package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type videoCatalogRepoStub struct {
	VideoGatewayRuntimeRepository
	providers []VideoProviderAccount
}

func (s *videoCatalogRepoStub) ListEnabledVideoProviders(context.Context, int64) ([]VideoProviderAccount, error) {
	return append([]VideoProviderAccount(nil), s.providers...), nil
}

func TestHCAtomVideoV1AdapterUsesFixedLegacyTaskEndpoint(t *testing.T) {
	client := &http.Client{Transport: videoRoundTripperFunc(func(request *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, request.Method)
		require.Equal(t, "https://api-aigc.fzyinghe.com/video/generation/tasks", request.URL.String())
		require.Equal(t, "Bearer secret", request.Header.Get("Authorization"))
		require.Equal(t, "application/json", request.Header.Get("Content-Type"))
		require.Equal(t, "video-create-123", request.Header.Get("Idempotency-Key"))
		var body map[string]any
		require.NoError(t, json.NewDecoder(request.Body).Decode(&body))
		require.Equal(t, map[string]any{
			"model":          HCAtomVideoV1PublicModel,
			"content":        []any{map[string]any{"type": "text", "text": "short scene"}},
			"generate_audio": true,
			"ratio":          "16:9",
			"duration":       float64(4),
			"watermark":      false,
		}, body)
		require.NotContains(t, body, "prompt")
		require.NotContains(t, body, "resolution")
		require.NotContains(t, body, "return_last_frame")
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"v1-1","status":"queued"}`)), Header: make(http.Header)}, nil
	})}
	task, err := NewHCAtomV1Adapter(client, HCAtomSeedanceV3BaseURL, "secret").Create(context.Background(), VideoCreateRequest{
		Prompt: "short scene", Ratio: "16:9", GenerateAudio: true, Duration: 4, Resolution: "720p", ReturnLastFrame: true,
		IdempotencyKey: "video-create-123",
	})
	require.NoError(t, err)
	require.Equal(t, "v1-1", task.UpstreamTaskID)
	require.Equal(t, VideoStatusSubmitted, task.Status)
}

func TestNormalizeHCAtomVideoV1ContentRoles(t *testing.T) {
	tests := []struct {
		name    string
		content []VideoContentItem
		wantErr bool
	}{
		{name: "text", content: []VideoContentItem{{Type: "text", Text: "scene"}}},
		{name: "reference image", content: []VideoContentItem{{Type: "image_url", Role: "reference_image", ImageURL: &VideoContentURL{URL: "https://assets.example.test/image.png"}}}},
		{name: "reference video", content: []VideoContentItem{{Type: "video_url", Role: "reference_video", VideoURL: &VideoContentURL{URL: "https://assets.example.test/video.mp4"}}}},
		{name: "reference audio", content: []VideoContentItem{{Type: "audio_url", Role: "reference_audio", AudioURL: &VideoContentURL{URL: "https://assets.example.test/audio.mp3"}}}},
		{name: "image requires role", content: []VideoContentItem{{Type: "image_url", ImageURL: &VideoContentURL{URL: "https://assets.example.test/image.png"}}}, wantErr: true},
		{name: "image rejects wrong role", content: []VideoContentItem{{Type: "image_url", Role: "first_frame", ImageURL: &VideoContentURL{URL: "https://assets.example.test/image.png"}}}, wantErr: true},
		{name: "text rejects role", content: []VideoContentItem{{Type: "text", Role: "reference_image", Text: "scene"}}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeHCAtomV1Content(VideoCreateRequest{Content: test.content})
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.content, got)
		})
	}
}

func TestHCAtomVideoV1AdapterPollAndCancelUseTaskPath(t *testing.T) {
	tests := []struct {
		name   string
		method string
		call   func(*HCAtomV1Adapter) (*VideoProviderTask, error)
	}{
		{name: "poll", method: http.MethodGet, call: func(adapter *HCAtomV1Adapter) (*VideoProviderTask, error) {
			return adapter.Poll(context.Background(), "v1/task")
		}},
		{name: "cancel", method: http.MethodDelete, call: func(adapter *HCAtomV1Adapter) (*VideoProviderTask, error) {
			return adapter.Cancel(context.Background(), "v1/task")
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &http.Client{Transport: videoRoundTripperFunc(func(request *http.Request) (*http.Response, error) {
				require.Equal(t, test.method, request.Method)
				require.Equal(t, "https://api-aigc.fzyinghe.com/video/generation/tasks/v1%2Ftask", request.URL.String())
				require.Equal(t, "Bearer secret", request.Header.Get("Authorization"))
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"v1/task","status":"running"}`)), Header: make(http.Header)}, nil
			})}
			task, err := test.call(NewHCAtomV1Adapter(client, HCAtomSeedanceV3BaseURL, "secret"))
			require.NoError(t, err)
			require.Equal(t, "v1/task", task.UpstreamTaskID)
		})
	}
}

func TestHCAtomVideoV1CreateTaskKeepsProviderSpecificCatalogModel(t *testing.T) {
	repo := &workerRepoStub{provider: VideoProviderAccount{
		ID: 10, GroupID: 9, Provider: HCAtomVideoV1Provider, Enabled: true,
		BaseURL: HCAtomSeedanceV3BaseURL, DefaultModel: HCAtomVideoV1PublicModel,
	}}
	cfg := &config.Config{
		HCAtom: config.HCAtomConfig{VideoV1Enabled: true},
		VideoGateway: config.VideoGatewayConfig{
			WorkerEnabled: true, HCAtomV1DispatchEnabled: true,
			SeedanceCNYPerMillionTokens: 2, USDCNYExchangeRate: 7, TinyRealEstimateCNY: .7, TinyRealMaximumCNY: 1.4,
		},
	}
	task, err := NewVideoGatewayService(repo, NewSingleSmokeAuthorization(true), cfg, &videoAuthInvalidatorStub{}, &videoBillingInvalidatorStub{}).CreateTask(context.Background(), VideoTaskCreateCommand{
		Scope: VideoTaskScope{UserID: 1, APIKeyID: 2, GroupID: 9}, ProviderAccountID: 10,
		Model: HCAtomVideoV1PublicModel, Prompt: "short scene", Duration: 4, Resolution: "720p",
	})
	require.NoError(t, err)
	require.Equal(t, HCAtomVideoV1Provider, task.Provider)
	require.Equal(t, HCAtomVideoV1PublicModel, task.Model)
}

func TestAuthorizedHCAtomVideoModelsRequiresPriceProviderAndGates(t *testing.T) {
	price := 0.2
	provider := VideoProviderAccount{
		ID: 10, GroupID: 9, Provider: HCAtomVideoV1Provider, Enabled: true, APIKeyConfigured: true,
		EncryptedAPIKey: "ciphertext", BaseURL: HCAtomMediaOrigin, DefaultModel: HCAtomVideoV1PublicModel,
	}
	repo := &workerRepoStub{provider: provider}
	cfg := &config.Config{HCAtom: config.HCAtomConfig{VideoV1Enabled: true}, VideoGateway: config.VideoGatewayConfig{HCAtomV1DispatchEnabled: true}}
	service := NewVideoGatewayService(repo, NewSingleSmokeAuthorization(true), cfg, nil, nil)
	group := &Group{
		ID: 9, Platform: PlatformHCAtom, Status: StatusActive,
		VideoPrice480P: &price, VideoPrice720P: &price, VideoPrice1080P: &price,
		ModelsListConfig: GroupModelsListConfig{Enabled: true, Models: []string{HCAtomVideoV1PublicModel}},
	}
	models := service.AuthorizedHCAtomVideoModels(context.Background(), VideoTaskScope{UserID: 1, APIKeyID: 2, GroupID: 9}, group)
	require.Len(t, models, 1)
	require.Equal(t, HCAtomVideoV1PublicModel, models[0].Model)

	cfg.VideoGateway.HCAtomV1DispatchEnabled = false
	require.Empty(t, service.AuthorizedHCAtomVideoModels(context.Background(), VideoTaskScope{UserID: 1, APIKeyID: 2, GroupID: 9}, group))
}

func TestAuthorizedHCAtomVideoModelsReturnsV1AndV3Independently(t *testing.T) {
	price := 0.2
	repo := &videoCatalogRepoStub{providers: []VideoProviderAccount{
		{ID: 10, GroupID: 9, Provider: HCAtomVideoV1Provider, Enabled: true, APIKeyConfigured: true, BaseURL: HCAtomMediaOrigin, DefaultModel: HCAtomVideoV1PublicModel},
		{ID: 11, GroupID: 9, Provider: HCAtomSeedanceV3Provider, Enabled: true, APIKeyConfigured: true, BaseURL: HCAtomMediaOrigin, DefaultModel: HCAtomSeedanceV3PublicModel},
	}}
	cfg := &config.Config{
		HCAtom:       config.HCAtomConfig{VideoV1Enabled: true},
		VideoGateway: config.VideoGatewayConfig{HCAtomV1DispatchEnabled: true, HCAtomV3DispatchEnabled: true, HCAtomV3ProductionEnabled: true},
	}
	service := NewVideoGatewayService(repo, NewSingleSmokeAuthorization(true), cfg, nil, nil)
	group := &Group{
		ID: 9, Platform: PlatformHCAtom, Status: StatusActive,
		VideoPrice480P: &price, VideoPrice720P: &price, VideoPrice1080P: &price,
		ModelsListConfig: GroupModelsListConfig{
			Enabled: true, Models: []string{HCAtomVideoV1PublicModel, HCAtomSeedanceV3PublicModel},
		},
	}

	models := service.AuthorizedHCAtomVideoModels(context.Background(), VideoTaskScope{UserID: 1, APIKeyID: 2, GroupID: 9}, group)
	require.Len(t, models, 2)
	require.Equal(t, []string{HCAtomVideoV1PublicModel, HCAtomSeedanceV3PublicModel}, []string{models[0].Model, models[1].Model})

	cfg.VideoGateway.HCAtomV1DispatchEnabled = false
	models = service.AuthorizedHCAtomVideoModels(context.Background(), VideoTaskScope{UserID: 1, APIKeyID: 2, GroupID: 9}, group)
	require.Len(t, models, 1)
	require.Equal(t, HCAtomSeedanceV3PublicModel, models[0].Model)

	cfg.VideoGateway.HCAtomV1DispatchEnabled = true
	cfg.VideoGateway.HCAtomV3ProductionEnabled = false
	models = service.AuthorizedHCAtomVideoModels(context.Background(), VideoTaskScope{UserID: 1, APIKeyID: 2, GroupID: 9}, group)
	require.Len(t, models, 1)
	require.Equal(t, HCAtomVideoV1PublicModel, models[0].Model)

	cfg.VideoGateway.HCAtomV3ProductionEnabled = true
	cfg.VideoGateway.HCAtomV3DispatchEnabled = false
	models = service.AuthorizedHCAtomVideoModels(context.Background(), VideoTaskScope{UserID: 1, APIKeyID: 2, GroupID: 9}, group)
	require.Len(t, models, 1)
	require.Equal(t, HCAtomVideoV1PublicModel, models[0].Model)
}

func TestAuthorizedHCAtomVideoModelsAllowsV3ProductionWithoutSmokeGate(t *testing.T) {
	price := 0.2
	repo := &videoCatalogRepoStub{providers: []VideoProviderAccount{{
		ID: 11, GroupID: 9, Provider: HCAtomSeedanceV3Provider, Enabled: true,
		APIKeyConfigured: true, BaseURL: HCAtomMediaOrigin, DefaultModel: HCAtomSeedanceV3PublicModel,
	}}}
	cfg := &config.Config{VideoGateway: config.VideoGatewayConfig{
		HCAtomV3ProductionEnabled: true, HCAtomV3DispatchEnabled: true,
	}}
	service := NewVideoGatewayService(repo, NewSingleSmokeAuthorization(false), cfg, nil, nil)
	group := &Group{
		ID: 9, Platform: PlatformHCAtom, Status: StatusActive,
		VideoPrice480P: &price, VideoPrice720P: &price, VideoPrice1080P: &price,
		ModelsListConfig: GroupModelsListConfig{Enabled: true, Models: []string{HCAtomSeedanceV3PublicModel}},
	}

	models := service.AuthorizedHCAtomVideoModels(
		context.Background(), VideoTaskScope{UserID: 1, APIKeyID: 2, GroupID: 9}, group,
	)

	require.Len(t, models, 1)
	require.Equal(t, HCAtomSeedanceV3PublicModel, models[0].Model)
}

func TestAuthorizedHCAtomVideoModelsIncludesOfficialSeedanceProductionAlias(t *testing.T) {
	price := 0.2
	authorizedAt := time.Now().UTC()
	repo := &videoCatalogRepoStub{providers: []VideoProviderAccount{{
		ID: 12, GroupID: 9, Provider: "seedance", Enabled: true, APIKeyConfigured: true,
		BaseURL: SeedanceBaseURL, DefaultModel: SeedanceModel, TinyRealAuthorizedAt: &authorizedAt,
	}}}
	cfg := &config.Config{VideoGateway: config.VideoGatewayConfig{SeedanceProductionEnabled: true}}
	service := NewVideoGatewayService(repo, NewSingleSmokeAuthorization(false), cfg, nil, nil)
	group := &Group{
		ID: 9, Platform: PlatformHCAtom, Status: StatusActive,
		VideoPrice480P: &price, VideoPrice720P: &price, VideoPrice1080P: &price,
		ModelsListConfig: GroupModelsListConfig{Enabled: true, Models: []string{SeedancePublicModel}},
	}

	models := service.AuthorizedHCAtomVideoModels(
		context.Background(), VideoTaskScope{UserID: 1, APIKeyID: 2, GroupID: 9}, group,
	)

	require.Len(t, models, 1)
	require.Equal(t, SeedancePublicModel, models[0].Model)
	require.Equal(t, []int{4}, models[0].Capabilities.DurationSecondsValues)
	require.Equal(t, []string{"720p"}, models[0].Capabilities.ResolutionValues)
	require.False(t, models[0].Capabilities.InputImage)
}
