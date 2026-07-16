import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import VersionBadge from '../VersionBadge.vue'

const {
  performUpdate,
  restartService,
  fetchVersion,
  clearVersionCache,
} = vi.hoisted(() => ({
  performUpdate: vi.fn(),
  restartService: vi.fn(),
  fetchVersion: vi.fn(),
  clearVersionCache: vi.fn(),
}))

vi.mock('@/api/admin/system', () => ({
  performUpdate,
  restartService,
  getRollbackVersions: vi.fn(),
  rollback: vi.fn(),
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({ isAdmin: true }),
  useAppStore: () => ({
    versionLoading: false,
    currentVersion: '1.0.0',
    latestVersion: '1.1.0',
    hasUpdate: true,
    releaseInfo: null,
    buildType: 'release',
    fetchVersion,
    clearVersionCache,
  }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copied: { value: false }, copyToClipboard: vi.fn() }),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

async function mountReadyForRestart() {
  performUpdate.mockResolvedValue({ message: 'updated', need_restart: true })
  const wrapper = mount(VersionBadge, {
    global: { stubs: { Icon: true } },
  })

  await wrapper.find('button').trigger('click')
  const updateButton = wrapper.findAll('button').find(button => button.text().includes('version.updateNow'))
  expect(updateButton).toBeDefined()
  await updateButton!.trigger('click')
  await flushPromises()
  return wrapper
}

function restartButton(wrapper: ReturnType<typeof mount>) {
  const button = wrapper.findAll('button').find(candidate => candidate.text().includes('version.restartNow'))
  expect(button).toBeDefined()
  return button!
}

describe('VersionBadge restart state', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    performUpdate.mockReset()
    restartService.mockReset()
    fetchVersion.mockReset()
    clearVersionCache.mockReset()
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    vi.clearAllTimers()
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('treats an HTTP restart rejection as failed without health polling', async () => {
    restartService.mockRejectedValue(Object.assign(new Error('request failed'), {
      response: { status: 503, data: { message: 'restart rejected by service manager' } },
    }))
    const wrapper = await mountReadyForRestart()

    await restartButton(wrapper).trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('restart rejected by service manager')
    expect(wrapper.text()).toContain('version.retry')
    expect(fetch).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('stops after five failed health checks in unknown state and allows retry', async () => {
    restartService.mockRejectedValue(new Error('connection closed before a response'))
    vi.mocked(fetch).mockResolvedValue({ ok: false } as Response)
    const wrapper = await mountReadyForRestart()

    await restartButton(wrapper).trigger('click')
    await flushPromises()
    await vi.runAllTimersAsync()
    await flushPromises()

    expect(fetch).toHaveBeenCalledTimes(5)
    expect(wrapper.text()).toContain('common.unknown')
    expect(wrapper.text()).toContain('version.retry')
    expect(wrapper.findAll('button').some(button => button.text().includes('version.retry') && !button.attributes('disabled'))).toBe(true)
    wrapper.unmount()
  })

  it('clears restart timers when unmounted', async () => {
    restartService.mockResolvedValue({ message: 'accepted' })
    const wrapper = await mountReadyForRestart()

    await restartButton(wrapper).trigger('click')
    await flushPromises()
    expect(vi.getTimerCount()).toBeGreaterThan(0)

    wrapper.unmount()
    expect(vi.getTimerCount()).toBe(0)
  })
})
