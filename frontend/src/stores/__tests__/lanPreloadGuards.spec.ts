import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { useAppStore } from '@/stores/app'
import { useAnnouncementStore } from '@/stores/announcements'
import { useSubscriptionStore } from '@/stores/subscriptions'
import { useAdminSettingsStore } from '@/stores/adminSettings'

const mocks = vi.hoisted(() => ({
  announcementsList: vi.fn(),
  getActiveSubscriptions: vi.fn(),
  getSettings: vi.fn(),
  getPaymentConfig: vi.fn(),
}))

vi.mock('@/api', () => ({
  announcementsAPI: { list: mocks.announcementsList },
  adminAPI: {
    settings: { getSettings: mocks.getSettings },
    payment: { getConfig: mocks.getPaymentConfig },
  },
}))

vi.mock('@/api/subscriptions', () => ({
  default: { getActiveSubscriptions: mocks.getActiveSubscriptions },
}))

function setLanAdminMode(enabled: boolean) {
  const appStore = useAppStore()
  appStore.cachedPublicSettings = { lan_admin_mode_enabled: enabled } as typeof appStore.cachedPublicSettings
}

describe('LAN admin preload guards (backend-mode guard 恒 403 的端点)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setActivePinia(createPinia())
    mocks.announcementsList.mockResolvedValue([])
    mocks.getActiveSubscriptions.mockResolvedValue([])
    mocks.getSettings.mockResolvedValue({})
    mocks.getPaymentConfig.mockResolvedValue({ data: { enabled: false } })
  })

  it('skips announcement preloads entirely in LAN mode', async () => {
    setLanAdminMode(true)
    const store = useAnnouncementStore()
    await store.fetchAnnouncements(true)
    expect(mocks.announcementsList).not.toHaveBeenCalled()
  })

  it('still loads announcements outside LAN mode', async () => {
    setLanAdminMode(false)
    const store = useAnnouncementStore()
    await store.fetchAnnouncements(true)
    expect(mocks.announcementsList).toHaveBeenCalledTimes(1)
  })

  it('skips subscription preload and polling in LAN mode', async () => {
    vi.useFakeTimers()
    try {
      setLanAdminMode(true)
      const store = useSubscriptionStore()
      await store.fetchActiveSubscriptions(true)
      expect(mocks.getActiveSubscriptions).not.toHaveBeenCalled()

      store.startPolling()
      await vi.advanceTimersByTimeAsync(6 * 60 * 1000)
      expect(mocks.getActiveSubscriptions).not.toHaveBeenCalled()
      store.stopPolling()
    } finally {
      vi.useRealTimers()
    }
  })

  it('still fetches subscriptions outside LAN mode', async () => {
    setLanAdminMode(false)
    const store = useSubscriptionStore()
    await store.fetchActiveSubscriptions(true)
    expect(mocks.getActiveSubscriptions).toHaveBeenCalledTimes(1)
  })

  it('skips admin settings fetch in LAN mode and keeps cached ops default', async () => {
    setLanAdminMode(true)
    const store = useAdminSettingsStore()
    await store.fetch(true)
    expect(mocks.getSettings).not.toHaveBeenCalled()
    expect(mocks.getPaymentConfig).not.toHaveBeenCalled()
    // ops 默认开启：内网控制台导航仍能看到「系统健康」
    expect(store.opsMonitoringEnabled).toBe(true)
  })

  it('still fetches admin settings outside LAN mode', async () => {
    setLanAdminMode(false)
    const store = useAdminSettingsStore()
    await store.fetch(true)
    expect(mocks.getSettings).toHaveBeenCalledTimes(1)
    expect(mocks.getPaymentConfig).toHaveBeenCalledTimes(1)
  })
})
