package service

// QCanvasModelKind is a coarse, non-sensitive description of a model
// capability attached to an authenticated key's group.
type QCanvasModelKind string

const (
	QCanvasModelKindText  QCanvasModelKind = "text"
	QCanvasModelKindImage QCanvasModelKind = "image"
	QCanvasModelKindVideo QCanvasModelKind = "video"
)

// QCanvasModelKindsForGroup derives capabilities only from the active HC-ATOM
// group contract and the canonical HC-ATOM catalog. Unknown, disabled or
// unpriced capabilities are ignored; no model-name heuristics are used.
func QCanvasModelKindsForGroup(group *Group) []QCanvasModelKind {
	if group == nil || group.Platform != PlatformHCAtom || !group.IsActive() || !group.ModelsListConfig.Enabled {
		return nil
	}

	available := make(map[QCanvasModelKind]struct{}, 3)
	for _, model := range group.ModelsListConfig.Models {
		spec, ok := lookupHCAtomPublicModel(model)
		if !ok || !spec.Enabled {
			continue
		}
		switch spec.Kind {
		case string(QCanvasModelKindText):
			available[QCanvasModelKindText] = struct{}{}
		case string(QCanvasModelKindImage):
			if qcanvasImageCapabilityConfigured(group, spec.Capability) {
				available[QCanvasModelKindImage] = struct{}{}
			}
		case string(QCanvasModelKindVideo):
			if qcanvasVideoPricingConfigured(group) {
				available[QCanvasModelKindVideo] = struct{}{}
			}
		}
	}

	ordered := []QCanvasModelKind{QCanvasModelKindText, QCanvasModelKindImage, QCanvasModelKindVideo}
	result := make([]QCanvasModelKind, 0, len(available))
	for _, kind := range ordered {
		if _, ok := available[kind]; ok {
			result = append(result, kind)
		}
	}
	return result
}

func qcanvasImageCapabilityConfigured(group *Group, capability string) bool {
	if group.ImagePrice1K == nil || *group.ImagePrice1K <= 0 ||
		group.ImagePrice2K == nil || *group.ImagePrice2K <= 0 ||
		group.ImagePrice4K == nil || *group.ImagePrice4K <= 0 {
		return false
	}
	switch capability {
	case HCAtomCapabilityImageSync:
		return group.AllowImageGeneration
	case HCAtomCapabilityImageAsync:
		return group.AllowBatchImageGeneration
	default:
		return false
	}
}

func qcanvasVideoPricingConfigured(group *Group) bool {
	return group.VideoPrice480P != nil && *group.VideoPrice480P > 0 &&
		group.VideoPrice720P != nil && *group.VideoPrice720P > 0 &&
		group.VideoPrice1080P != nil && *group.VideoPrice1080P > 0
}
