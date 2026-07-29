package service

import (
	"context"
	"strings"
)

const (
	HCAtomChatOrigin  = "https://ai-aigc.fzyinghe.com"
	HCAtomMediaOrigin = "https://api-aigc.fzyinghe.com"

	HCAtomCapabilityChat       = "chat"
	HCAtomCapabilityMessages   = "messages"
	HCAtomCapabilityImageSync  = "image_sync"
	HCAtomCapabilityImageAsync = "image_async"
	HCAtomCapabilityVideoV1    = "video_v1"
	HCAtomCapabilityVideoV3    = "video_v3"

	HCAtomPricingChannelToken = "channel_token"
	HCAtomPricingGroupImage   = "group_image"
	HCAtomPricingGroupVideo   = "group_video"

	HCAtomAuthBearer  = "bearer"
	HCAtomAuthXAPIKey = "x_api_key"

	HCAtomImageSeedreamModel       = "seedream-5.0"
	HCAtomImageDoubaoSeedreamModel = "doubao-seedream-5.0-pro"
	HCAtomImageGeminiModel         = "gemini-3.1-flash-image-preview"
	HCAtomImageGPTModel            = "gpt-image-2"
	HCAtomImageSGPTModel           = "s-gpt-image-2"
	HCAtomImageDolaModel           = "dola-seedream-5.0-pro"

	// Compatibility aliases retained for the existing batch-image harness.
	HCAtomImageAsyncI2IModel = HCAtomImageGPTModel
	HCAtomImageAsyncT2IModel = HCAtomImageGeminiModel
)

// HCAtomModelSpec is the single internal source of truth for HC routing. The
// upstream fields must never be serialized by public model-catalog handlers.
type HCAtomModelSpec struct {
	PublicModel        string
	UpstreamModel      string
	DisplayName        string
	Kind               string
	Capability         string
	Origin             string
	Path               string
	AuthScheme         string
	PublicCapabilities PublicModelCapabilities
	PricingPolicy      string
	Enabled            bool
}

var hcAtomModelCatalog = []HCAtomModelSpec{
	{PublicModel: "gpt-5.6-sol", UpstreamModel: "gpt-5.6-sol", DisplayName: "GPT 5.6 Sol", Kind: "text", Capability: HCAtomCapabilityChat, Origin: HCAtomChatOrigin, Path: "/v1/chat/completions", AuthScheme: HCAtomAuthBearer, PublicCapabilities: PublicModelCapabilities{TaskMode: "sync"}, PricingPolicy: HCAtomPricingChannelToken, Enabled: true},
	{PublicModel: "gemini-3-flash-preview", UpstreamModel: "gemini-3-flash-preview", DisplayName: "Gemini 3 Flash Preview", Kind: "text", Capability: HCAtomCapabilityChat, Origin: HCAtomChatOrigin, Path: "/v1/chat/completions", AuthScheme: HCAtomAuthBearer, PublicCapabilities: PublicModelCapabilities{TaskMode: "sync", InputImage: true}, PricingPolicy: HCAtomPricingChannelToken, Enabled: true},
	{PublicModel: "claude-opus-4-6", UpstreamModel: "claude-opus-4-6", DisplayName: "Claude Opus 4.6", Kind: "text", Capability: HCAtomCapabilityMessages, Origin: HCAtomChatOrigin, Path: "/v1/messages", AuthScheme: HCAtomAuthBearer, PublicCapabilities: PublicModelCapabilities{TaskMode: "sync", InputImage: true}, PricingPolicy: HCAtomPricingChannelToken, Enabled: true},
	{PublicModel: HCAtomImageSeedreamModel, UpstreamModel: HCAtomImageSeedreamModel, DisplayName: "Seedream 5.0", Kind: "image", Capability: HCAtomCapabilityImageSync, Origin: HCAtomChatOrigin, Path: "/v1/images/generations", AuthScheme: HCAtomAuthBearer, PublicCapabilities: PublicModelCapabilities{TaskMode: "sync", Resolution: true}, PricingPolicy: HCAtomPricingGroupImage, Enabled: true},
	{PublicModel: HCAtomImageDoubaoSeedreamModel, UpstreamModel: HCAtomImageDoubaoSeedreamModel, DisplayName: "Doubao Seedream 5.0 Pro", Kind: "image", Capability: HCAtomCapabilityImageSync, Origin: HCAtomChatOrigin, Path: "/v1/images/generations", AuthScheme: HCAtomAuthBearer, PublicCapabilities: PublicModelCapabilities{TaskMode: "sync", InputImage: true, ReferenceImages: true, Resolution: true}, PricingPolicy: HCAtomPricingGroupImage, Enabled: true},
	{PublicModel: HCAtomImageGeminiModel, UpstreamModel: HCAtomImageGeminiModel, DisplayName: "Gemini 3.1 Flash Image Preview", Kind: "image", Capability: HCAtomCapabilityImageAsync, Origin: HCAtomMediaOrigin, Path: "/image/generation/tasks", AuthScheme: HCAtomAuthBearer, PublicCapabilities: PublicModelCapabilities{TaskMode: "async", InputImage: true, ReferenceImages: true, AspectRatio: true, Resolution: true}, PricingPolicy: HCAtomPricingGroupImage, Enabled: true},
	{PublicModel: HCAtomImageGPTModel, UpstreamModel: HCAtomImageGPTModel, DisplayName: "GPT Image 2", Kind: "image", Capability: HCAtomCapabilityImageAsync, Origin: HCAtomMediaOrigin, Path: "/image/generation/tasks", AuthScheme: HCAtomAuthXAPIKey, PublicCapabilities: PublicModelCapabilities{TaskMode: "async", InputImage: true, ReferenceImages: true, AspectRatio: true, Resolution: true}, PricingPolicy: HCAtomPricingGroupImage, Enabled: true},
	{PublicModel: HCAtomImageSGPTModel, UpstreamModel: HCAtomImageSGPTModel, DisplayName: "S-GPT Image 2", Kind: "image", Capability: HCAtomCapabilityImageAsync, Origin: HCAtomMediaOrigin, Path: "/image/generation/tasks", AuthScheme: HCAtomAuthXAPIKey, PublicCapabilities: PublicModelCapabilities{TaskMode: "async", InputImage: true, ReferenceImages: true, AspectRatio: true, Resolution: true}, PricingPolicy: HCAtomPricingGroupImage, Enabled: true},
	// The HC portal currently marks this authorized model as "待配置接口地址".
	// Keep the catalog record for auditability, but never advertise or dispatch it.
	{PublicModel: HCAtomImageDolaModel, UpstreamModel: HCAtomImageDolaModel, DisplayName: "Dola Seedream 5.0 Pro", Kind: "image", Capability: HCAtomCapabilityImageAsync, PricingPolicy: HCAtomPricingGroupImage, Enabled: false},
	{PublicModel: HCAtomVideoV1PublicModel, UpstreamModel: HCAtomVideoV1PublicModel, DisplayName: "Doubao Seedance 2.0", Kind: "video", Capability: HCAtomCapabilityVideoV1, Origin: HCAtomMediaOrigin, Path: "/video/generation/tasks", AuthScheme: HCAtomAuthBearer, PublicCapabilities: PublicModelCapabilities{TaskMode: "async", InputImage: true, ReferenceImages: true, AspectRatio: true, DurationSeconds: true, Audio: true}, PricingPolicy: HCAtomPricingGroupVideo, Enabled: true},
	{PublicModel: HCAtomSeedanceV3PublicModel, UpstreamModel: HCAtomSeedanceV3Model, DisplayName: "Doubao Seedance 2.0 V3", Kind: "video", Capability: HCAtomCapabilityVideoV3, Origin: HCAtomMediaOrigin, Path: HCAtomSeedanceV3Path, AuthScheme: HCAtomAuthBearer, PublicCapabilities: PublicModelCapabilities{TaskMode: "async", InputImage: true, FirstFrame: true, LastFrame: true, ReferenceImages: true, AspectRatio: true, DurationSeconds: true, Resolution: true, Audio: true}, PricingPolicy: HCAtomPricingGroupVideo, Enabled: true},
}

func LookupHCAtomModel(capability, publicModel string) (HCAtomModelSpec, bool) {
	capability, publicModel = strings.TrimSpace(capability), strings.TrimSpace(publicModel)
	for _, spec := range hcAtomModelCatalog {
		if spec.Enabled && spec.Capability == capability && spec.PublicModel == publicModel {
			return spec, true
		}
	}
	return HCAtomModelSpec{}, false
}

func HCAtomModelCatalog() []HCAtomModelSpec {
	out := make([]HCAtomModelSpec, len(hcAtomModelCatalog))
	copy(out, hcAtomModelCatalog)
	return out
}

type PublicModelCapabilities struct {
	TaskMode        string `json:"task_mode"`
	InputImage      bool   `json:"input_image"`
	FirstFrame      bool   `json:"first_frame"`
	LastFrame       bool   `json:"last_frame"`
	ReferenceImages bool   `json:"reference_images"`
	AspectRatio     bool   `json:"aspect_ratio"`
	DurationSeconds bool   `json:"duration_seconds"`
	Resolution      bool   `json:"resolution"`
	Audio           bool   `json:"audio"`
}

type PublicModel struct {
	ID           string                  `json:"id"`
	Model        string                  `json:"model"`
	DisplayName  string                  `json:"display_name"`
	Kind         string                  `json:"kind"`
	Capabilities PublicModelCapabilities `json:"capabilities"`
}

func PublicHCAtomModels(kind string) []PublicModel {
	kind = strings.ToLower(strings.TrimSpace(kind))
	models := make([]PublicModel, 0, len(hcAtomModelCatalog))
	for _, spec := range hcAtomModelCatalog {
		if !spec.Enabled || (kind != "" && spec.Kind != kind) {
			continue
		}
		models = append(models, PublicModel{ID: spec.PublicModel, Model: spec.PublicModel, DisplayName: spec.DisplayName, Kind: spec.Kind, Capabilities: spec.PublicCapabilities})
	}
	return models
}

// AuthorizedHCAtomPublicModels fails closed unless the current key's HC group
// explicitly allowlists the model, has the catalog-selected pricing policy,
// and can select a live HC account. Video uses a separate provider-account
// domain and is authorized by AuthorizedHCAtomVideoModels.
func (s *GatewayService) AuthorizedHCAtomPublicModels(ctx context.Context, apiKey *APIKey, kind string) []PublicModel {
	if s == nil || apiKey == nil || apiKey.Group == nil || apiKey.Group.Platform != PlatformHCAtom ||
		!apiKey.Group.IsActive() || !apiKey.Group.CustomModelsListEnabled() {
		return []PublicModel{}
	}
	allowed := make(map[string]struct{}, len(apiKey.Group.ModelsListConfig.Models))
	for _, model := range apiKey.Group.ModelsListConfig.Models {
		allowed[strings.TrimSpace(model)] = struct{}{}
	}
	candidates := PublicHCAtomModels(kind)
	out := make([]PublicModel, 0, len(candidates))
	for _, model := range candidates {
		if model.Kind == "video" {
			continue
		}
		if _, ok := allowed[model.Model]; !ok || !s.hcAtomPublicCapabilityEnabled(model.Model) {
			continue
		}
		spec, ok := lookupHCAtomPublicModel(model.Model)
		if !ok || !s.hcAtomPublicPricingConfigured(ctx, apiKey, spec) {
			continue
		}
		selection, err := s.SelectAccountWithLoadAwareness(ctx, apiKey.GroupID, "", model.Model, nil, "", 0)
		if err != nil || selection == nil || selection.Account == nil || selection.Account.Platform != PlatformHCAtom {
			if selection != nil && selection.ReleaseFunc != nil {
				selection.ReleaseFunc()
			}
			continue
		}
		if selection.ReleaseFunc != nil {
			selection.ReleaseFunc()
		}
		out = append(out, model)
	}
	return out
}

func lookupHCAtomPublicModel(model string) (HCAtomModelSpec, bool) {
	model = strings.TrimSpace(model)
	for _, spec := range hcAtomModelCatalog {
		if spec.Enabled && spec.PublicModel == model {
			return spec, true
		}
	}
	return HCAtomModelSpec{}, false
}

func (s *GatewayService) hcAtomPublicPricingConfigured(ctx context.Context, apiKey *APIKey, spec HCAtomModelSpec) bool {
	switch spec.PricingPolicy {
	case HCAtomPricingChannelToken:
		if s.resolver == nil {
			return false
		}
		resolved := s.resolver.Resolve(ctx, PricingInput{Model: spec.PublicModel, GroupID: apiKey.GroupID})
		return resolved != nil && resolved.Source == PricingSourceChannel && resolvedHCAtomPricingUsable(resolved)
	case HCAtomPricingGroupImage:
		group := apiKey.Group
		if group == nil || group.ImagePrice1K == nil || *group.ImagePrice1K <= 0 ||
			group.ImagePrice2K == nil || *group.ImagePrice2K <= 0 ||
			group.ImagePrice4K == nil || *group.ImagePrice4K <= 0 {
			return false
		}
		if spec.Capability == HCAtomCapabilityImageSync {
			return group.AllowImageGeneration
		}
		return spec.Capability == HCAtomCapabilityImageAsync && group.AllowBatchImageGeneration
	default:
		return false
	}
}

func (s *GatewayService) hcAtomPublicCapabilityEnabled(model string) bool {
	if s == nil || s.cfg == nil {
		return false
	}
	if _, ok := LookupHCAtomModel(HCAtomCapabilityChat, model); ok {
		return s.cfg.HCAtom.LLMEnabled
	}
	if _, ok := LookupHCAtomModel(HCAtomCapabilityMessages, model); ok {
		return s.cfg.HCAtom.LLMEnabled
	}
	if _, ok := LookupHCAtomModel(HCAtomCapabilityImageSync, model); ok {
		return s.cfg.HCAtom.SyncImageEnabled
	}
	if _, ok := LookupHCAtomModel(HCAtomCapabilityImageAsync, model); ok {
		return s.cfg.BatchImage.HCAtomEnabled
	}
	return false
}

func resolvedHCAtomPricingUsable(resolved *ResolvedPricing) bool {
	switch resolved.Mode {
	case BillingModePerRequest, BillingModeImage:
		return resolved.DefaultPerRequestPrice > 0 || len(resolved.RequestTiers) > 0
	default:
		if len(resolved.Intervals) > 0 {
			return true
		}
		pricing := resolved.BasePricing
		return pricing != nil && (pricing.InputPricePerToken > 0 || pricing.OutputPricePerToken > 0 || pricing.ImageOutputPricePerToken > 0)
	}
}

func (s *VideoGatewayService) AuthorizedHCAtomVideoModels(ctx context.Context, scope VideoTaskScope, group *Group) []PublicModel {
	if s == nil || s.cfg == nil || s.repo == nil || s.gate == nil || !s.gate.Allowed() ||
		group == nil || group.ID != scope.GroupID || group.Platform != PlatformHCAtom || !group.IsActive() ||
		group.VideoPrice480P == nil || *group.VideoPrice480P <= 0 ||
		group.VideoPrice720P == nil || *group.VideoPrice720P <= 0 ||
		group.VideoPrice1080P == nil || *group.VideoPrice1080P <= 0 {
		return []PublicModel{}
	}
	providers, err := s.repo.ListEnabledVideoProviders(ctx, scope.GroupID)
	if err != nil {
		return []PublicModel{}
	}
	enabledV1, enabledV3 := false, false
	for _, provider := range providers {
		if !provider.Enabled || !provider.APIKeyConfigured || strings.TrimSpace(provider.EncryptedAPIKey) == "" ||
			strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/") != HCAtomMediaOrigin {
			continue
		}
		switch provider.Provider {
		case HCAtomVideoV1Provider:
			if provider.DefaultModel == HCAtomVideoV1PublicModel {
				enabledV1 = enabledV1 || (s.cfg.HCAtom.VideoV1Enabled && s.cfg.VideoGateway.HCAtomV1DispatchEnabled)
			}
		case HCAtomSeedanceV3Provider:
			if provider.DefaultModel == HCAtomSeedanceV3PublicModel {
				enabledV3 = enabledV3 || s.cfg.VideoGateway.HCAtomV3DispatchEnabled
			}
		}
	}
	if !enabledV1 && !enabledV3 {
		return []PublicModel{}
	}
	out := make([]PublicModel, 0, 2)
	for _, model := range PublicHCAtomModels("video") {
		if (model.Model == HCAtomVideoV1PublicModel && enabledV1) || (model.Model == HCAtomSeedanceV3PublicModel && enabledV3) {
			out = append(out, model)
		}
	}
	return out
}

func isHCAtomCatalogModel(value string) bool {
	value = strings.TrimSpace(value)
	for _, spec := range hcAtomModelCatalog {
		if spec.Enabled && (spec.PublicModel == value || spec.UpstreamModel == value) {
			return true
		}
	}
	return false
}

func isHCAtomBatchEnabledModel(model string) bool {
	spec, ok := LookupHCAtomModel(HCAtomCapabilityImageAsync, model)
	return ok && spec.Enabled
}
