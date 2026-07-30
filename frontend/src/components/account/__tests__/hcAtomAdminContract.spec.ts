import { describe, expect, it } from 'vitest'
import {
  HC_ATOM_IMAGE_BASE_URL,
  HC_ATOM_IMAGE_ENABLED_MODELS,
  HC_ATOM_MEDIA_ENABLED_MODELS,
  HC_ATOM_TEXT_ENABLED_MODELS,
  HC_ATOM_VIDEO_ENABLED_MODELS,
  buildHCAtomImageCredentials,
  isHCAtomImageGroup,
  isHCAtomVideoGroup,
} from '../hcAtomAdminContract'

describe('HC-ATOM image account admin contract', () => {
  it('uses the fixed relay origin and exposes the complete media and video model allowlists', () => {
    expect(HC_ATOM_IMAGE_BASE_URL).toBe('https://api-aigc.fzyinghe.com')
    expect(HC_ATOM_TEXT_ENABLED_MODELS).toEqual([
      'gpt-5.6-sol',
      'gemini-3-flash-preview',
      'claude-opus-4-6',
    ])
    expect(HC_ATOM_IMAGE_ENABLED_MODELS).toEqual([
      'seedream-5.0',
      'doubao-seedream-5.0-pro',
      'gemini-3.1-flash-image-preview',
      'gpt-image-2',
      's-gpt-image-2',
    ])
    expect(HC_ATOM_MEDIA_ENABLED_MODELS).toEqual(HC_ATOM_IMAGE_ENABLED_MODELS)
    expect(HC_ATOM_VIDEO_ENABLED_MODELS).toEqual([
      'doubao-seedance-2.0',
      'doubao-seedance-2.0-v3',
    ])
  })

  it('builds the exact transient secret and fixed model mapping payload', () => {
    expect(buildHCAtomImageCredentials('  test-secret  ')).toEqual({
      api_key: 'test-secret',
      protocol: 'hc_atom',
      model_mapping: {
        'seedream-5.0': 'seedream-5.0',
        'doubao-seedream-5.0-pro': 'doubao-seedream-5.0-pro',
        'gemini-3.1-flash-image-preview': 'gemini-3.1-flash-image-preview',
        'gpt-image-2': 'gpt-image-2',
        's-gpt-image-2': 's-gpt-image-2',
      },
    })
  })

  it('never accepts an empty secret', () => {
    expect(() => buildHCAtomImageCredentials('   ')).toThrow('HC-ATOM API Key')
  })

  it('keeps image and video groups mutually exclusive by configured capability', () => {
    const image = {
      platform: 'hc_atom',
      status: 'active',
      allow_image_generation: true,
      allow_batch_image_generation: true,
      image_price_1k: 0.134,
      image_price_2k: 0.201,
      image_price_4k: 0.268,
      models_list_config: { enabled: true, models: [...HC_ATOM_IMAGE_ENABLED_MODELS] },
    } as any
    const video = {
      platform: 'hc_atom',
      status: 'active',
      allow_image_generation: false,
      allow_batch_image_generation: false,
      video_price_480p: 0.05,
      video_price_720p: 0.07,
      video_price_1080p: 0.25,
      models_list_config: { enabled: true, models: [...HC_ATOM_VIDEO_ENABLED_MODELS] },
    } as any
    expect(isHCAtomImageGroup(image)).toBe(true)
    expect(isHCAtomVideoGroup(image)).toBe(false)
    expect(isHCAtomVideoGroup(video)).toBe(true)
    expect(isHCAtomImageGroup(video)).toBe(false)
    expect(isHCAtomImageGroup({
      ...image,
      models_list_config: { enabled: true, models: [...HC_ATOM_IMAGE_ENABLED_MODELS, ...HC_ATOM_VIDEO_ENABLED_MODELS] },
    })).toBe(false)
    expect(isHCAtomImageGroup({
      ...image,
      models_list_config: { enabled: true, models: [...HC_ATOM_IMAGE_ENABLED_MODELS, 'dola-seedream-5.0-pro'] },
    })).toBe(false)
    expect(isHCAtomImageGroup({
      ...image,
      models_list_config: { enabled: true, models: [...HC_ATOM_IMAGE_ENABLED_MODELS, 'gpt-5.6-sol'] },
    })).toBe(false)
    expect(isHCAtomImageGroup({ ...image, image_price_2k: null })).toBe(false)
    expect(isHCAtomVideoGroup({ ...video, video_price_720p: 0 })).toBe(false)
  })
})
