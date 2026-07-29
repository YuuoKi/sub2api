import type { AdminGroup } from '@/types'

export const HC_ATOM_SYNC_IMAGE_BASE_URL = 'https://ai-aigc.fzyinghe.com'
export const HC_ATOM_ASYNC_IMAGE_BASE_URL = 'https://api-aigc.fzyinghe.com'
// Kept for existing form defaults. Requests are routed by the backend catalog,
// not by a user-editable base URL.
export const HC_ATOM_IMAGE_BASE_URL = HC_ATOM_ASYNC_IMAGE_BASE_URL

export const HC_ATOM_TEXT_ENABLED_MODELS = [
  'gpt-5.6-sol',
  'gemini-3-flash-preview',
  'claude-opus-4-6',
] as const

export const HC_ATOM_IMAGE_ENABLED_MODELS = [
  'seedream-5.0',
  'doubao-seedream-5.0-pro',
  'gemini-3.1-flash-image-preview',
  'gpt-image-2',
  's-gpt-image-2',
] as const

// The HC credential authorized for this release is image-only. Text accounts
// remain separate and must never be silently added to an image group or key.
export const HC_ATOM_MEDIA_ENABLED_MODELS = HC_ATOM_IMAGE_ENABLED_MODELS

// Authorized by HC but not dispatchable until HC publishes an endpoint.
export const HC_ATOM_MEDIA_PENDING_MODELS = ['dola-seedream-5.0-pro'] as const

export const HC_ATOM_VIDEO_V1_MODEL = 'doubao-seedance-2.0'
export const HC_ATOM_VIDEO_V3_MODEL = 'doubao-seedance-2.0-v3'

export const HC_ATOM_VIDEO_ENABLED_MODELS = [
  HC_ATOM_VIDEO_V1_MODEL,
  HC_ATOM_VIDEO_V3_MODEL,
] as const

export const HC_ATOM_MEDIA_GROUP_PRICES = {
  image_price_1k: 0.134,
  image_price_2k: 0.201,
  image_price_4k: 0.268,
} as const

export const HC_ATOM_VIDEO_GROUP_PRICES = {
  video_price_480p: 0.05,
  video_price_720p: 0.07,
  video_price_1080p: 0.25,
} as const

export const buildHCAtomImageCredentials = (rawAPIKey: string): Record<string, unknown> => {
  const apiKey = rawAPIKey.trim()
  if (!apiKey) throw new Error('HC-ATOM API Key 不能为空')

  return {
    api_key: apiKey,
    protocol: 'hc_atom',
    model_mapping: Object.fromEntries(
      HC_ATOM_MEDIA_ENABLED_MODELS.map((model) => [model, model]),
    ),
  }
}

const configuredModels = (group: Pick<AdminGroup, 'models_list_config'>): string[] => (
  group.models_list_config?.enabled
    ? group.models_list_config.models.map((model) => model.trim()).filter(Boolean)
    : []
)

export const isHCAtomImageGroup = (group: AdminGroup): boolean => {
  if (group.platform !== 'hc_atom' || group.status !== 'active') return false
  if (!group.allow_image_generation || !group.allow_batch_image_generation) return false
  if (
    (group.image_price_1k ?? 0) <= 0
    || (group.image_price_2k ?? 0) <= 0
    || (group.image_price_4k ?? 0) <= 0
  ) return false
  const models = new Set(configuredModels(group))
  return models.size === HC_ATOM_IMAGE_ENABLED_MODELS.length
    && HC_ATOM_IMAGE_ENABLED_MODELS.every((model) => models.has(model))
}

export const isHCAtomVideoGroup = (group: AdminGroup): boolean => {
  if (group.platform !== 'hc_atom' || group.status !== 'active') return false
  if (
    (group.video_price_480p ?? 0) <= 0
    || (group.video_price_720p ?? 0) <= 0
    || (group.video_price_1080p ?? 0) <= 0
  ) return false
  const models = new Set(configuredModels(group))
  return HC_ATOM_VIDEO_ENABLED_MODELS.some((model) => models.has(model))
    && HC_ATOM_IMAGE_ENABLED_MODELS.every((model) => !models.has(model))
}
