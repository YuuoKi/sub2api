package service

import "strings"

// VideoProviderSpec describes one video platform known to the admin console.
// AdapterReady marks platforms with a working dispatch adapter; the rest are
// announced in the contract but cannot be created as channels yet.
type VideoProviderSpec struct {
	Provider       string `json:"provider"`
	DisplayName    string `json:"display_name"`
	DefaultBaseURL string `json:"default_base_url"`
	DefaultModel   string `json:"default_model"`
	AdapterReady   bool   `json:"adapter_ready"`
}

var videoProviderRegistry = []VideoProviderSpec{
	{Provider: "seedance", DisplayName: "Seedance 2.0", DefaultBaseURL: SeedanceBaseURL, DefaultModel: SeedanceModel, AdapterReady: true},
	{Provider: HCAtomVideoV1Provider, DisplayName: "HC-ATOM Video V1", DefaultBaseURL: HCAtomSeedanceV3BaseURL, DefaultModel: HCAtomVideoV1PublicModel, AdapterReady: true},
	{Provider: HCAtomSeedanceV3Provider, DisplayName: "HC-ATOM Seedance V3", DefaultBaseURL: HCAtomSeedanceV3BaseURL, DefaultModel: HCAtomSeedanceV3PublicModel, AdapterReady: true},
	{Provider: "jimeng", DisplayName: "即梦", AdapterReady: false},
	{Provider: "veo", DisplayName: "Veo 3.1", AdapterReady: false},
	{Provider: "kling", DisplayName: "快乐小马", AdapterReady: false},
}

// VideoProviderRegistry returns a copy of the known video platforms.
func VideoProviderRegistry() []VideoProviderSpec {
	out := make([]VideoProviderSpec, len(videoProviderRegistry))
	copy(out, videoProviderRegistry)
	return out
}

func lookupVideoProvider(provider string) (VideoProviderSpec, bool) {
	normalized := strings.ToLower(strings.TrimSpace(provider))
	for _, spec := range videoProviderRegistry {
		if spec.Provider == normalized {
			return spec, true
		}
	}
	return VideoProviderSpec{}, false
}
