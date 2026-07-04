import { describe, expect, it } from 'vitest'
import { sanitizeRedirectPath } from '@/utils/sanitizeRedirectPath'

describe('sanitizeRedirectPath', () => {
  it('allows safe relative paths', () => {
    expect(sanitizeRedirectPath('/dashboard')).toBe('/dashboard')
    expect(sanitizeRedirectPath('/settings/profile')).toBe('/settings/profile')
  })

  it('rejects open redirects', () => {
    expect(sanitizeRedirectPath('https://evil.example')).toBe('/dashboard')
    expect(sanitizeRedirectPath('//evil.example/path')).toBe('/dashboard')
    expect(sanitizeRedirectPath('/path\n/to')).toBe('/dashboard')
  })

  it('falls back when empty', () => {
    expect(sanitizeRedirectPath('')).toBe('/dashboard')
    expect(sanitizeRedirectPath(undefined)).toBe('/dashboard')
  })
})
