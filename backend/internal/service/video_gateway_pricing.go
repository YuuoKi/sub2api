package service

import (
	"sort"
	"strings"
)

const (
	VideoPricingCurrencyCNY           = BillingCurrencyCNY
	VideoPricingVersionSeedance202603 = "ark-seedance-2026-03"
	// VideoPricingVersionKling202607 is the Kling video catalog stamp.
	// PLACEHOLDER / provisional: CNY-per-second rates below are NOT official
	// tariffs. Production settle must fail-closed while this version is in use;
	// tiny_real may still estimate. Replace rates and bump the version stamp
	// only after official Kling credentials/pricing are confirmed.
	VideoPricingVersionKling202607 = "kling-video-2026-07"
	videoSeedancePricingVersion    = VideoPricingVersionSeedance202603
	videoKlingPricingVersion       = VideoPricingVersionKling202607
	videoTokensPerMillion          = 1000000.0
)

// IsKlingPricingProvisional reports whether the Kling catalog stamp is still a
// PLACEHOLDER / provisional version that must not be used for production settle.
func IsKlingPricingProvisional(version string) bool {
	v := strings.TrimSpace(version)
	if v == "" {
		v = VideoPricingVersionKling202607
	}
	switch v {
	case VideoPricingVersionKling202607:
		return true
	default:
		// Future official stamps (e.g. kling-video-2026-08-official) are non-provisional.
		return strings.Contains(strings.ToLower(v), "placeholder") ||
			strings.Contains(strings.ToLower(v), "provisional")
	}
}

type VideoPricingEntry struct {
	Provider                   string
	ModelMatch                 string
	DefaultCNYPerMillionTokens float64
	NoVideoCNYPerMillionTokens float64
	VideoCNYPerMillionTokens   float64
	NoAudioCNYPerMillionTokens float64
	AudioCNYPerMillionTokens   float64
	// StdCNYPerSecond / ProCNYPerSecond are Kling-style per-second CNY rates
	// (std vs pro mode). Zero means the entry is token-priced (Seedance).
	StdCNYPerSecond float64
	ProCNYPerSecond float64
	PricingVersion  string
}

type VideoPricingCatalog struct {
	entries []VideoPricingEntry
}

func NewVideoPricingCatalog(overrides []VideoPricingEntry) *VideoPricingCatalog {
	entries := append([]VideoPricingEntry(nil), defaultVideoPricingEntries()...)
	entries = append(entries, overrides...)
	sort.SliceStable(entries, func(i, j int) bool {
		return len(entries[i].ModelMatch) > len(entries[j].ModelMatch)
	})
	return &VideoPricingCatalog{entries: entries}
}

func (s *VideoGatewayService) SetVideoPricingCatalog(catalog *VideoPricingCatalog) {
	if catalog == nil {
		catalog = NewVideoPricingCatalog(nil)
	}
	s.videoPricing = catalog
}

func (s *VideoGatewayService) videoPricingCatalog() *VideoPricingCatalog {
	if s == nil || s.videoPricing == nil {
		return NewVideoPricingCatalog(nil)
	}
	return s.videoPricing
}

func (s *VideoGatewayService) applyVideoBillingMetadata(task *VideoTask) {
	if task == nil {
		return
	}
	task.Currency = NormalizeBillingCurrency(task.Currency)
	task.PricingSource = NormalizeBillingPricingSource(task.PricingSource)
	task.PricingVersion = NormalizeBillingPricingVersion(task.PricingVersion)
	if strings.EqualFold(strings.TrimSpace(task.Provider), VideoProviderSeedance) {
		task.Currency = BillingCurrencyCNY
		task.PricingSource = PricingSourceProviderUsage
		if _, version, ok := s.videoPricingCatalog().RateCNYPerMillionTokens(task); ok && strings.TrimSpace(version) != "" {
			task.PricingVersion = version
		}
		if task.PricingVersion == "" {
			task.PricingVersion = VideoPricingVersionSeedance202603
		}
		return
	}
	if strings.EqualFold(strings.TrimSpace(task.Provider), VideoProviderKling) {
		task.Currency = BillingCurrencyCNY
		task.PricingSource = PricingSourceProviderUsage
		if _, version, ok := s.videoPricingCatalog().RateCNYPerSecond(task); ok && strings.TrimSpace(version) != "" {
			task.PricingVersion = version
		}
		if task.PricingVersion == "" {
			task.PricingVersion = VideoPricingVersionKling202607
		}
		return
	}
	if task.PricingVersion == "" {
		task.PricingVersion = PricingVersionForSource(task.PricingSource)
	}
}

func defaultVideoPricingEntries() []VideoPricingEntry {
	return []VideoPricingEntry{
		{
			Provider:                   VideoProviderSeedance,
			ModelMatch:                 "doubao-seedance-2-0-fast",
			DefaultCNYPerMillionTokens: 37,
			NoVideoCNYPerMillionTokens: 37,
			VideoCNYPerMillionTokens:   22,
			PricingVersion:             videoSeedancePricingVersion,
		},
		{
			Provider:                   VideoProviderSeedance,
			ModelMatch:                 "doubao-seedance-2-0",
			DefaultCNYPerMillionTokens: 46,
			NoVideoCNYPerMillionTokens: 46,
			VideoCNYPerMillionTokens:   28,
			PricingVersion:             videoSeedancePricingVersion,
		},
		{
			Provider:                   VideoProviderSeedance,
			ModelMatch:                 "doubao-seedance-1-5-pro",
			DefaultCNYPerMillionTokens: 8,
			NoAudioCNYPerMillionTokens: 8,
			AudioCNYPerMillionTokens:   16,
			PricingVersion:             videoSeedancePricingVersion,
		},
		{
			Provider:                   VideoProviderSeedance,
			ModelMatch:                 "doubao-seedance-1-0-pro-fast",
			DefaultCNYPerMillionTokens: 4.2,
			PricingVersion:             videoSeedancePricingVersion,
		},
		{
			Provider:                   VideoProviderSeedance,
			ModelMatch:                 "doubao-seedance-1-0-pro",
			DefaultCNYPerMillionTokens: 15,
			PricingVersion:             videoSeedancePricingVersion,
		},
		{
			Provider:                   VideoProviderSeedance,
			ModelMatch:                 "doubao-seedance-1-0-lite",
			DefaultCNYPerMillionTokens: 10,
			PricingVersion:             videoSeedancePricingVersion,
		},
		// Kling entries: PLACEHOLDER CNY/sec rates (family × std/pro) until official
		// credentials/pricing are confirmed. Do not treat as production tariffs.
		{
			Provider:        VideoProviderKling,
			ModelMatch:      "kling-v2-6",
			StdCNYPerSecond: 0.8,
			ProCNYPerSecond: 1.6,
			PricingVersion:  videoKlingPricingVersion,
		},
		{
			Provider:        VideoProviderKling,
			ModelMatch:      "kling-2-6-pro",
			StdCNYPerSecond: 0.8,
			ProCNYPerSecond: 1.6,
			PricingVersion:  videoKlingPricingVersion,
		},
		{
			Provider:        VideoProviderKling,
			ModelMatch:      "kling-v3-omni",
			StdCNYPerSecond: 1.0,
			ProCNYPerSecond: 2.0,
			PricingVersion:  videoKlingPricingVersion,
		},
		{
			Provider:        VideoProviderKling,
			ModelMatch:      "kling-3-0-omni",
			StdCNYPerSecond: 1.0,
			ProCNYPerSecond: 2.0,
			PricingVersion:  videoKlingPricingVersion,
		},
		{
			Provider:        VideoProviderKling,
			ModelMatch:      "kling-video-o1",
			StdCNYPerSecond: 1.0,
			ProCNYPerSecond: 2.0,
			PricingVersion:  videoKlingPricingVersion,
		},
		{
			Provider:        VideoProviderKling,
			ModelMatch:      "kling-o1",
			StdCNYPerSecond: 1.0,
			ProCNYPerSecond: 2.0,
			PricingVersion:  videoKlingPricingVersion,
		},
		{
			Provider:        VideoProviderKling,
			ModelMatch:      "kling-3-0",
			StdCNYPerSecond: 0.8,
			ProCNYPerSecond: 1.6,
			PricingVersion:  videoKlingPricingVersion,
		},
		{
			Provider:        VideoProviderKling,
			ModelMatch:      "kling-v1",
			StdCNYPerSecond: 0.5,
			ProCNYPerSecond: 1.0,
			PricingVersion:  videoKlingPricingVersion,
		},
		{
			Provider:        VideoProviderKling,
			ModelMatch:      "kling-video-extend",
			StdCNYPerSecond: 0.5,
			ProCNYPerSecond: 1.0,
			PricingVersion:  videoKlingPricingVersion,
		},
		{
			Provider:        VideoProviderKling,
			ModelMatch:      "kling-avatar",
			StdCNYPerSecond: 0.5,
			ProCNYPerSecond: 1.0,
			PricingVersion:  videoKlingPricingVersion,
		},
		{
			Provider:        VideoProviderKling,
			ModelMatch:      "kling-lip-sync",
			StdCNYPerSecond: 0.5,
			ProCNYPerSecond: 1.0,
			PricingVersion:  videoKlingPricingVersion,
		},
		{
			Provider:        VideoProviderKling,
			ModelMatch:      "kling",
			StdCNYPerSecond: 0.5,
			ProCNYPerSecond: 1.0,
			PricingVersion:  videoKlingPricingVersion,
		},
	}
}

func (c *VideoPricingCatalog) RateCNYPerMillionTokens(task *VideoTask) (float64, string, bool) {
	if c == nil || task == nil {
		return 0, "", false
	}
	provider := strings.ToLower(strings.TrimSpace(task.Provider))
	model := normalizeVideoPricingModel(task.Model)
	for _, entry := range c.entries {
		if strings.ToLower(strings.TrimSpace(entry.Provider)) != provider {
			continue
		}
		if entry.StdCNYPerSecond > 0 || entry.ProCNYPerSecond > 0 {
			continue // per-second entries are not token-priced
		}
		if !strings.Contains(model, normalizeVideoPricingModel(entry.ModelMatch)) {
			continue
		}
		return entry.rateForTask(task), firstNonEmptyVideo(entry.PricingVersion, videoSeedancePricingVersion), true
	}
	return 0, "", false
}

// RateCNYPerSecond looks up Kling-style placeholder CNY/sec rates (std vs pro).
func (c *VideoPricingCatalog) RateCNYPerSecond(task *VideoTask) (float64, string, bool) {
	if c == nil || task == nil {
		return 0, "", false
	}
	provider := strings.ToLower(strings.TrimSpace(task.Provider))
	model := normalizeVideoPricingModel(task.Model)
	for _, entry := range c.entries {
		if strings.ToLower(strings.TrimSpace(entry.Provider)) != provider {
			continue
		}
		if entry.StdCNYPerSecond <= 0 && entry.ProCNYPerSecond <= 0 {
			continue
		}
		if !strings.Contains(model, normalizeVideoPricingModel(entry.ModelMatch)) {
			continue
		}
		return entry.perSecondRateForTask(task), firstNonEmptyVideo(entry.PricingVersion, videoKlingPricingVersion), true
	}
	return 0, "", false
}

func (e VideoPricingEntry) rateForTask(task *VideoTask) float64 {
	if task != nil && task.HasVideoInput && e.VideoCNYPerMillionTokens > 0 {
		return e.VideoCNYPerMillionTokens
	}
	if task != nil && !task.HasVideoInput && e.NoVideoCNYPerMillionTokens > 0 {
		return e.NoVideoCNYPerMillionTokens
	}
	if task != nil && videoBoolPtrValue(task.GenerateAudio) && e.AudioCNYPerMillionTokens > 0 {
		return e.AudioCNYPerMillionTokens
	}
	if task != nil && !videoBoolPtrValue(task.GenerateAudio) && e.NoAudioCNYPerMillionTokens > 0 {
		return e.NoAudioCNYPerMillionTokens
	}
	return e.DefaultCNYPerMillionTokens
}

func (e VideoPricingEntry) perSecondRateForTask(task *VideoTask) float64 {
	mode := "std"
	if task != nil {
		mode = klingModeFromTask(task)
	}
	if mode == "pro" && e.ProCNYPerSecond > 0 {
		return e.ProCNYPerSecond
	}
	if e.StdCNYPerSecond > 0 {
		return e.StdCNYPerSecond
	}
	return e.ProCNYPerSecond
}

func normalizeVideoPricingModel(model string) string {
	out := strings.ToLower(strings.TrimSpace(model))
	out = strings.ReplaceAll(out, ".", "-")
	out = strings.ReplaceAll(out, "_", "-")
	return out
}

func videoBoolPtrValue(v *bool) bool {
	return v != nil && *v
}
