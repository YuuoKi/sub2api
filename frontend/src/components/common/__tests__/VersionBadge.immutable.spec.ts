import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import VersionBadge from '../VersionBadge.vue'

const { fetchVersion, getVersion, checkUpdates, performUpdate, restartService } = vi.hoisted(() => ({
  fetchVersion: vi.fn(),
  getVersion: vi.fn(),
  checkUpdates: vi.fn(),
  performUpdate: vi.fn(),
  restartService: vi.fn(),
}))

vi.mock('@/api/admin/system', () => ({
  getVersion,
  checkUpdates,
  performUpdate,
  restartService,
  getRollbackVersions: vi.fn(),
  rollback: vi.fn(),
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({ isAdmin: true }),
  useAppStore: () => ({
    versionLoading: false,
    currentVersion: '广州内部版 2026.07.25-r151',
    buildCommit: 'abcdef0123456789abcdef0123456789abcdef01',
    buildDate: '2026-07-25T08:30:00Z',
    latestVersion: '',
    hasUpdate: false,
    fetchVersion,
  }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copied: { value: false }, copyToClipboard: vi.fn() }),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

describe('VersionBadge immutable deploy identity', () => {
  beforeEach(() => {
    fetchVersion.mockReset()
    getVersion.mockReset()
    checkUpdates.mockReset()
    performUpdate.mockReset()
    restartService.mockReset()
  })

  it('shows current deploy version only and has no self-update controls', async () => {
    const wrapper = mount(VersionBadge, {
      global: { stubs: { Icon: true } },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('广州内部版 2026.07.25-r151')
    expect(wrapper.text()).not.toMatch(/vabcdef|v广州/)

    const text = wrapper.text()
    expect(text).not.toContain('version.updateNow')
    expect(text).not.toContain('version.latestVersion')
    expect(text).not.toContain('version.rollback')
    expect(text).not.toContain('version.refresh')
    expect(text).not.toContain('version.restartNow')

    const buttons = wrapper.findAll('button')
    expect(
      buttons.every(
        (button) =>
          !button.text().includes('version.updateNow') &&
          !button.text().includes('version.rollback') &&
          !button.text().includes('version.refresh') &&
          !button.text().includes('version.restartNow')
      )
    ).toBe(true)

    expect(checkUpdates).not.toHaveBeenCalled()
    expect(performUpdate).not.toHaveBeenCalled()
    expect(restartService).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('details expose full SHA and build time without forced v prefix', async () => {
    const wrapper = mount(VersionBadge, {
      global: { stubs: { Icon: true } },
    })
    await wrapper.find('button').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('abcdef0123456789abcdef0123456789abcdef01')
    expect(wrapper.text()).toContain('2026-07-25T08:30:00Z')
    expect(wrapper.text()).not.toContain('vabcdef0123456789abcdef0123456789abcdef01')
    wrapper.unmount()
  })
})
