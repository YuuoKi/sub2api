import { afterEach, describe, expect, it, vi } from 'vitest'
import { createIdempotencyKey } from '../idempotencyKey'

const UUID_V4_RE =
  /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i

describe('createIdempotencyKey', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('prefers crypto.randomUUID when present', () => {
    vi.spyOn(crypto, 'randomUUID').mockReturnValue('11111111-2222-4333-8444-555555555555')
    expect(createIdempotencyKey()).toBe('11111111-2222-4333-8444-555555555555')
  })

  it('falls back to getRandomValues when randomUUID is missing (HTTP / non-secure)', () => {
    const original = crypto.randomUUID
    Object.defineProperty(crypto, 'randomUUID', {
      configurable: true,
      value: undefined,
    })
    try {
      const key = createIdempotencyKey()
      expect(key).toMatch(UUID_V4_RE)
    } finally {
      Object.defineProperty(crypto, 'randomUUID', {
        configurable: true,
        value: original,
      })
    }
  })
})
