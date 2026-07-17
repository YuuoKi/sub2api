import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { nextTick } from 'vue'
import { useAppStore } from '@/stores/app'
import AppSidebar from '../AppSidebar.vue'

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    isAdmin: true,
    isSimpleMode: false,
    isAuthenticated: true,
  }),
}))

vi.mock('@/stores/onboarding', () => ({
  useOnboardingStore: () => ({
    isCurrentStep: () => false,
    nextStep: vi.fn(),
  }),
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({
    fetch: vi.fn(),
    opsMonitoringEnabled: true,
    customMenuItems: [],
  }),
}))

vi.mock('@/stores', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/stores')>()
  return {
    ...actual,
    useAuthStore: () => ({
      isAdmin: true,
      isSimpleMode: false,
      isAuthenticated: true,
    }),
    useOnboardingStore: () => ({
      isCurrentStep: () => false,
      nextStep: vi.fn(),
    }),
    useAdminSettingsStore: () => ({
      fetch: vi.fn(),
      opsMonitoringEnabled: true,
      customMenuItems: [],
    }),
  }
})

vi.mock('@/composables/useBatchImageAccess', () => ({
  useBatchImageAccess: () => ({
    canUseBatchImage: { value: false },
    refreshBatchImageAccess: vi.fn(),
  }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

describe('AppSidebar System expand when collapsed', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    const appStore = useAppStore()
    appStore.setSidebarCollapsed(true)
    appStore.setMobileOpen(false)
    appStore.cachedPublicSettings = {
      ...(appStore.cachedPublicSettings ?? {}),
      custom_menu_items: [],
      channel_monitor_enabled: true,
      available_channels_enabled: true,
      risk_control_enabled: true,
    } as typeof appStore.cachedPublicSettings
    Object.defineProperty(window, 'innerWidth', {
      configurable: true,
      writable: true,
      value: 1280,
    })
  })

  async function mountSidebar() {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: { template: '<div />' } },
        { path: '/admin/console/overview', component: { template: '<div />' } },
        { path: '/admin/system', component: { template: '<div />' } },
        { path: '/admin/ops', component: { template: '<div />' } },
      ],
    })
    await router.push('/admin/console/overview')
    await router.isReady()

    return mount(AppSidebar, {
      global: {
        plugins: [router],
        stubs: {
          VersionBadge: { template: '<span />' },
          RouterLink: {
            props: ['to'],
            template: '<a :href="typeof to === \'string\' ? to : to?.path"><slot /></a>',
          },
        },
      },
    })
  }

  it('expands collapsed sidebar so System children become reachable after clicking System', async () => {
    const appStore = useAppStore()
    const wrapper = await mountSidebar()
    await flushPromises()

    expect(appStore.sidebarCollapsed).toBe(true)
    expect(wrapper.text()).not.toContain('运行与配置')

    const systemButton = wrapper
      .findAll('button')
      .find((btn) => btn.text().includes('系统'))
    expect(systemButton).toBeTruthy()
    expect(systemButton!.attributes('aria-controls')).toBeTruthy()

    await systemButton!.trigger('click')
    await nextTick()
    await flushPromises()

    expect(appStore.sidebarCollapsed).toBe(false)
    expect(wrapper.text()).toContain('运行与配置')
    expect(wrapper.text()).toContain('高级与历史')

    const panelId = systemButton!.attributes('aria-controls')!
    expect(wrapper.find(`#${panelId}`).exists()).toBe(true)

    wrapper.unmount()
  })

  it('opens the mobile drawer when expanding System on a narrow viewport', async () => {
    Object.defineProperty(window, 'innerWidth', {
      configurable: true,
      writable: true,
      value: 390,
    })
    const appStore = useAppStore()
    appStore.setSidebarCollapsed(false)

    const wrapper = await mountSidebar()
    await flushPromises()

    const systemButton = wrapper
      .findAll('button')
      .find((btn) => btn.text().includes('系统'))
    expect(systemButton).toBeTruthy()

    await systemButton!.trigger('click')
    await nextTick()

    expect(appStore.mobileOpen).toBe(true)
    expect(wrapper.text()).toContain('运行与配置')

    wrapper.unmount()
  })
})
