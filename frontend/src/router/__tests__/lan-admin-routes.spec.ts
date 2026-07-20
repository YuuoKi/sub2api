import { describe, expect, it } from 'vitest'

import { isLanAdminWebRouteAllowed } from '@/router'

describe('lan_admin web surface policy', () => {
  it('allows only bootstrap/login before an administrator is authenticated', () => {
    expect(isLanAdminWebRouteAllowed('/login', false, false)).toBe(true)
    expect(isLanAdminWebRouteAllowed('/setup', false, false)).toBe(true)

    for (const path of [
      '/home',
      '/register',
      '/email-verify',
      '/auth/oidc/callback',
      '/key-usage',
      '/payment/result',
      '/internal-pilot',
    ]) {
      expect(isLanAdminWebRouteAllowed(path, false, false), path).toBe(false)
    }
  })

  it('rejects every employee page even for an authenticated administrator', () => {
    for (const path of [
      '/dashboard',
      '/keys',
      '/usage',
      '/video/create',
      '/video/tasks',
      '/subscriptions',
      '/redeem',
      '/affiliate',
      '/custom/9',
    ]) {
      expect(isLanAdminWebRouteAllowed(path, true, true), path).toBe(false)
    }
  })

  it('allows only an authenticated administrator to change a temporary password', () => {
    expect(isLanAdminWebRouteAllowed('/change-temporary-password', true, true)).toBe(true)
    expect(isLanAdminWebRouteAllowed('/change-temporary-password', false, false)).toBe(false)
    expect(isLanAdminWebRouteAllowed('/change-temporary-password', true, false)).toBe(false)
  })

  it('allows the five administrator capability areas and their operational detail pages', () => {
    for (const path of [
      '/admin/console/overview',
      '/admin/console/key-vault',
      '/admin/console/staff',
      '/admin/accounts',
      '/admin/channels/pricing',
      '/admin/video/providers',
      '/admin/video/tasks',
      '/admin/video/tasks/task-1',
      '/admin/console/ai-records',
      '/admin/generation-content',
      '/admin/usage',
      '/admin/ops',
      '/admin/video/system-check',
      '/admin/settings',
    ]) {
      expect(isLanAdminWebRouteAllowed(path, true, true), path).toBe(true)
    }
  })

  it('rejects legacy commercial, user-management and review surfaces for administrators', () => {
    for (const path of [
      '/admin/dashboard',
      '/admin/users',
      '/admin/subscriptions',
      '/admin/redeem',
      '/admin/promo-codes',
      '/admin/orders',
      '/admin/affiliates',
      '/admin/announcements',
      '/internal-pilot',
    ]) {
      expect(isLanAdminWebRouteAllowed(path, true, true), path).toBe(false)
    }
  })
})
