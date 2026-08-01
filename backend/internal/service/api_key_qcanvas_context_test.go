package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQCanvasModelKindsForGroup(t *testing.T) {
	price := func(value float64) *float64 { return &value }
	video := &Group{
		ID:              11,
		Platform:        PlatformHCAtom,
		Status:          StatusActive,
		VideoPrice480P:  price(0.05),
		VideoPrice720P:  price(0.07),
		VideoPrice1080P: price(0.25),
		ModelsListConfig: GroupModelsListConfig{Enabled: true, Models: []string{
			HCAtomVideoV1PublicModel,
			HCAtomSeedanceV3PublicModel,
		}},
	}
	media := &Group{
		ID:                        12,
		Platform:                  PlatformHCAtom,
		Status:                    StatusActive,
		AllowImageGeneration:      true,
		AllowBatchImageGeneration: true,
		ImagePrice1K:              price(0.134),
		ImagePrice2K:              price(0.201),
		ImagePrice4K:              price(0.268),
		ModelsListConfig: GroupModelsListConfig{Enabled: true, Models: []string{
			HCAtomImageSeedreamModel,
			HCAtomImageDoubaoSeedreamModel,
			HCAtomImageGeminiModel,
			HCAtomImageGPTModel,
			HCAtomImageSGPTModel,
		}},
	}

	require.Equal(t, []QCanvasModelKind{QCanvasModelKindVideo}, QCanvasModelKindsForGroup(video))

	require.Equal(t, []QCanvasModelKind{QCanvasModelKindImage}, QCanvasModelKindsForGroup(media))

	mixed := *video
	mixed.AllowImageGeneration = true
	mixed.ImagePrice1K = price(0.134)
	mixed.ImagePrice2K = price(0.201)
	mixed.ImagePrice4K = price(0.268)
	mixed.ModelsListConfig.Models = append(append([]string{}, video.ModelsListConfig.Models...), HCAtomImageSeedreamModel, "gpt-5.6-sol")
	require.Equal(t, []QCanvasModelKind{QCanvasModelKindText, QCanvasModelKindImage, QCanvasModelKindVideo}, QCanvasModelKindsForGroup(&mixed))

	require.Empty(t, QCanvasModelKindsForGroup(&Group{ID: 13, Platform: PlatformHCAtom, Status: StatusActive}))
	require.Empty(t, QCanvasModelKindsForGroup(&Group{ID: 14, Platform: PlatformOpenAI, Status: StatusActive, ModelsListConfig: video.ModelsListConfig}))
}
