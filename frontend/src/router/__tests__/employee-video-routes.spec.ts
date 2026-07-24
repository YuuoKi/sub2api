import { describe, expect, it, vi } from 'vitest'

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

describe('employee video simulation routes', () => {
  it('registers create/list/detail with requiresAuth and without requiresAdmin', () => {
    for (const path of ['/video/create', '/video/tasks', '/video/tasks/:id']) {
      const route = findRoute(path)
      expect(route, path).toBeDefined()
      expect(route?.meta.requiresAuth).not.toBe(false)
      expect(route?.meta.requiresAdmin).not.toBe(true)
    }
  })

  it('keeps legacy admin video and console routes registered with requiresAdmin', () => {
    for (const path of [
      '/admin/console/overview',
      '/admin/console/key-vault',
      '/admin/console/staff',
      '/admin/video/tasks',
      '/admin/video/providers',
      '/admin/dashboard',
      '/admin/users',
      '/admin/subscriptions',
      '/admin/redeem',
      '/admin/promo-codes',
      '/admin/orders',
      '/admin/affiliates/invites',
    ]) {
      const resolved = router.resolve(path)
      expect(resolved.name, path).not.toBe('NotFound')
      expect(resolved.meta.requiresAdmin, path).toBe(true)
    }
  })
})
