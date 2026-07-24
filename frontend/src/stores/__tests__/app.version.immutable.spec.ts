import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { getVersion } from '@/api/admin/system'
import { useAppStore } from '@/stores/app'
import type { PublicSettings } from '@/types'

const { getVersionMock } = vi.hoisted(() => ({
  getVersionMock: vi.fn(),
}))

vi.mock('@/api/admin/system', () => ({
  getVersion: getVersionMock,
}))

vi.mock('@/api/auth', () => ({
  getPublicSettings: vi.fn(),
}))

function createPublicSettings(overrides: Partial<PublicSettings> = {}): PublicSettings {
  return {
    lan_admin_mode_enabled: false,
    registration_enabled: true,
    email_verify_enabled: false,
    force_email_on_third_party_signup: false,
    registration_email_suffix_whitelist: [],
    promo_code_enabled: false,
    password_reset_enabled: false,
    invitation_code_enabled: false,
    turnstile_enabled: false,
    turnstile_site_key: '',
    site_name: 'test',
    site_logo: '',
    site_subtitle: '',
    api_base_url: '',
    contact_info: '',
    doc_url: '',
    home_content: '',
    hide_ccs_import_button: false,
    payment_enabled: false,
    risk_control_enabled: false,
    table_default_page_size: 20,
    table_page_size_options: [20],
    custom_menu_items: [],
    custom_endpoints: [],
    linuxdo_oauth_enabled: false,
    oidc_oauth_enabled: false,
    oidc_oauth_provider_name: '',
    github_oauth_enabled: false,
    google_oauth_enabled: false,
    backend_mode_enabled: false,
    version: '广州内部版 2026.07.25-r151',
    build_commit: 'abcdef0123456789',
    build_date: '2026-07-25T08:30:00Z',
    balance_low_notify_enabled: false,
    account_quota_notify_enabled: false,
    balance_low_notify_threshold: 0,
    channel_monitor_enabled: true,
    channel_monitor_default_interval_seconds: 60,
    available_channels_enabled: false,
    service_quota_enabled: false,
    affiliate_enabled: false,
    ...overrides,
  }
}

describe('app store immutable version identity', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    getVersionMock.mockReset()
    delete (window as Window & { __APP_CONFIG__?: PublicSettings }).__APP_CONFIG__
  })

  it('fetchVersion never calls check-updates path and uses public build identity', async () => {
    window.__APP_CONFIG__ = createPublicSettings()
    const store = useAppStore()
    store.initFromInjectedConfig()

    const result = await store.fetchVersion(true)

    expect(getVersion).not.toHaveBeenCalled()
    expect(result?.version).toBe('广州内部版 2026.07.25-r151')
    expect(store.currentVersion).toBe('广州内部版 2026.07.25-r151')
    expect(store.buildCommit).toBe('abcdef0123456789')
    expect(store.buildDate).toBe('2026-07-25T08:30:00Z')
    expect(store.hasUpdate).toBe(false)
  })
})
