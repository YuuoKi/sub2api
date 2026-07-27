import { describe, expect, it } from 'vitest'
import {
  HC_ATOM_IMAGE_BASE_URL,
  HC_ATOM_IMAGE_ENABLED_MODELS,
  HC_ATOM_IMAGE_RESERVED_MODELS,
  buildHCAtomImageCredentials,
} from '../hcAtomAdminContract'

describe('HC-ATOM image account admin contract', () => {
  it('uses the fixed relay origin and exposes only enabled image models', () => {
    expect(HC_ATOM_IMAGE_BASE_URL).toBe('https://api-aigc.fzyinghe.com')
    expect(HC_ATOM_IMAGE_ENABLED_MODELS).toEqual([
      'seedream-5.0',
      'doubao-seedream-5.0-pro',
    ])
    expect(HC_ATOM_IMAGE_RESERVED_MODELS).toEqual(['dola-seedream-5.0-pro'])
    expect(HC_ATOM_IMAGE_ENABLED_MODELS).not.toContain('dola-seedream-5.0-pro')
  })

  it('builds the exact transient secret and fixed model mapping payload', () => {
    expect(buildHCAtomImageCredentials('  test-secret  ')).toEqual({
      api_key: 'test-secret',
      protocol: 'hc_atom',
      model_mapping: {
        'seedream-5.0': 'seedream-5.0',
        'doubao-seedream-5.0-pro': 'doubao-seedream-5.0-pro',
      },
    })
  })

  it('never accepts an empty secret', () => {
    expect(() => buildHCAtomImageCredentials('   ')).toThrow('HC-ATOM API Key')
  })
})
