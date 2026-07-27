export const HC_ATOM_IMAGE_BASE_URL = 'https://api-aigc.fzyinghe.com'

export const HC_ATOM_IMAGE_ENABLED_MODELS = [
  'seedream-5.0',
  'doubao-seedream-5.0-pro',
] as const

export const HC_ATOM_IMAGE_RESERVED_MODELS = [
  'dola-seedream-5.0-pro',
] as const

export const buildHCAtomImageCredentials = (rawAPIKey: string): Record<string, unknown> => {
  const apiKey = rawAPIKey.trim()
  if (!apiKey) throw new Error('HC-ATOM API Key 不能为空')

  return {
    api_key: apiKey,
    protocol: 'hc_atom',
    model_mapping: Object.fromEntries(
      HC_ATOM_IMAGE_ENABLED_MODELS.map((model) => [model, model]),
    ),
  }
}
