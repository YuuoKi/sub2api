import { describe, expect, it } from 'vitest'
import {
  HC_ATOM_IMAGE_BASE_URL,
  HC_ATOM_IMAGE_ENABLED_MODELS,
  HC_ATOM_MEDIA_ENABLED_MODELS,
  HC_ATOM_VIDEO_ENABLED_MODELS,
  buildHCAtomImageCredentials,
} from '../hcAtomAdminContract'

describe('HC-ATOM image account admin contract', () => {
  it('uses the fixed relay origin and exposes the complete media and video model allowlists', () => {
    expect(HC_ATOM_IMAGE_BASE_URL).toBe('https://api-aigc.fzyinghe.com')
    expect(HC_ATOM_MEDIA_ENABLED_MODELS).toEqual([
      'gpt-5.6-sol',
      'gemini-3-flash-preview',
      'claude-opus-4-6',
      'seedream-5.0',
      'wan2.5-t2i-preview',
      'wan2.5-i2i-preview',
    ])
    expect(HC_ATOM_IMAGE_ENABLED_MODELS).toBe(HC_ATOM_MEDIA_ENABLED_MODELS)
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
        'gpt-5.6-sol': 'gpt-5.6-sol',
        'gemini-3-flash-preview': 'gemini-3-flash-preview',
        'claude-opus-4-6': 'claude-opus-4-6',
        'seedream-5.0': 'seedream-5.0',
        'wan2.5-t2i-preview': 'wan2.5-t2i-preview',
        'wan2.5-i2i-preview': 'wan2.5-i2i-preview',
      },
    })
  })

  it('never accepts an empty secret', () => {
    expect(() => buildHCAtomImageCredentials('   ')).toThrow('HC-ATOM API Key')
  })
})
