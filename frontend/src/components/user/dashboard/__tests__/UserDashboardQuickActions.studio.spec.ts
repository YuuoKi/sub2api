import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import UserDashboardQuickActions from '../UserDashboardQuickActions.vue'
import { buildQCanvasProjectsURL } from '@/utils/qcanvas'

const routerPush = vi.fn()
const refreshBatchImageAccess = vi.fn()

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: routerPush })
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

vi.mock('@/composables/useBatchImageAccess', () => ({
  useBatchImageAccess: () => ({
    canUseBatchImage: false,
    refreshBatchImageAccess
  })
}))

describe('UserDashboardQuickActions Studio V2 entry', () => {
  beforeEach(() => {
    routerPush.mockReset()
    refreshBatchImageAccess.mockReset()
    vi.restoreAllMocks()
  })

  it('builds the projects page with a local default and an explicit override', () => {
    expect(buildQCanvasProjectsURL()).toBe('http://127.0.0.1:5174/projects')
    expect(buildQCanvasProjectsURL('https://studio.internal.example')).toBe(
      'https://studio.internal.example/projects'
    )
  })

  it('rejects unsafe or ambiguous configured base URLs', () => {
    expect(() => buildQCanvasProjectsURL('file:///tmp/qcanvas')).toThrow('HTTP 或 HTTPS')
    expect(() => buildQCanvasProjectsURL('https://user:pass@studio.example')).toThrow('用户名或密码')
    expect(() => buildQCanvasProjectsURL('https://studio.example/qcanvas')).toThrow('纯 origin')
    expect(() => buildQCanvasProjectsURL('https://studio.example?tenant=1')).toThrow('纯 origin')
  })

  it('shows a native project link without batch-image access', () => {
    const wrapper = mount(UserDashboardQuickActions, {
      global: { stubs: { Icon: true } }
    })

    const entry = wrapper.find('[data-testid="studio-v2-entry"]')
    expect(entry.exists()).toBe(true)
    expect(entry.attributes('href')).toBe('http://127.0.0.1:5174/projects')
    expect(entry.attributes('target')).toBe('_blank')
    expect(entry.attributes('rel')).toBe('noopener noreferrer')
    expect(routerPush).not.toHaveBeenCalled()
  })
})
