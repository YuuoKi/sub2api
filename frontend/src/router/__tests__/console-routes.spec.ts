import { describe, expect, it } from 'vitest'

// Mock 导航加载状态
import { vi } from 'vitest'

vi.mock('@/composables/useNavigationLoading', () => ({
  useNavigationLoadingState: () => ({
    startNavigation: vi.fn(),
    endNavigation: vi.fn(),
    isLoading: { value: false },
  }),
}))

vi.mock('@/composables/useRoutePrefetch', () => ({
  useRoutePrefetch: () => ({
    triggerPrefetch: vi.fn(),
    cancelPendingPrefetch: vi.fn(),
    resetPrefetchState: vi.fn(),
  }),
}))

vi.mock('@/api', () => ({
  authAPI: {
    getCurrentUser: vi.fn().mockResolvedValue({ data: {} }),
    logout: vi.fn(),
  },
  isTotp2FARequired: () => false,
}))

vi.mock('@/api/admin/system', () => ({
  getVersion: vi.fn(),
}))

vi.mock('@/api/auth', () => ({
  getPublicSettings: vi.fn(),
}))

import router from '@/router'

function findRoute(path: string) {
  return router.getRoutes().find((route) => route.path === path)
}

describe('Console v2 business routes', () => {
  it('registers /admin/console/overview requiring admin auth, distinct from /admin/video redirect', () => {
    const route = findRoute('/admin/console/overview')
    expect(route).toBeDefined()
    expect(route?.meta.requiresAuth).not.toBe(false)
    expect(route?.meta.requiresAdmin).toBe(true)

    // The existing /admin/video redirect must remain untouched (no mode-dependent swap introduced).
    const videoRedirect = findRoute('/admin/video')
    expect(videoRedirect?.redirect).toBe('/admin/video/providers')
  })

  it('registers /admin/console/key-vault requiring admin auth', () => {
    const route = findRoute('/admin/console/key-vault')
    expect(route).toBeDefined()
    expect(route?.meta.requiresAdmin).toBe(true)
  })

  it('registers /admin/console/staff requiring admin auth', () => {
    const route = findRoute('/admin/console/staff')
    expect(route).toBeDefined()
    expect(route?.meta.requiresAdmin).toBe(true)
  })

  it('registers /admin/console/ai-records requiring admin auth (no longer falls to catch-all)', () => {
    const resolved = router.resolve('/admin/console/ai-records')
    expect(resolved.name).toBe('AdminConsoleAiRecords')
    expect(resolved.name).not.toBe('NotFound')
    expect(resolved.meta.requiresAdmin).toBe(true)
  })

  it('registers /admin/generation-content requiring admin auth', () => {
    const route = findRoute('/admin/generation-content')
    expect(route).toBeDefined()
    expect(route?.meta.requiresAdmin).toBe(true)
  })

  it('registers /internal-pilot as a public branding route with no data', () => {
    const route = findRoute('/internal-pilot')
    expect(route).toBeDefined()
    expect(route?.meta.requiresAuth).toBe(false)
    expect(route?.meta.requiresAdmin).not.toBe(true)
  })

  it('keeps the catch-all 404 route intact', () => {
    const resolved = router.resolve('/this-path-does-not-exist')
    expect(resolved.name).toBe('NotFound')
  })
})
