export const HC_ATOM_IMAGE_BASE_URL = 'https://api-aigc.fzyinghe.com'

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

export const HC_ATOM_MEDIA_ENABLED_MODELS = [
  ...HC_ATOM_TEXT_ENABLED_MODELS,
  ...HC_ATOM_IMAGE_ENABLED_MODELS,
] as const

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
