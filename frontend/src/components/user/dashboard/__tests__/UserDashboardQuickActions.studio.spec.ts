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

  it('shows the entry without batch-image access and opens the project creator directly', async () => {
    const open = vi.spyOn(window, 'open').mockImplementation(() => null)
    const wrapper = mount(UserDashboardQuickActions, {
      global: { stubs: { Icon: true } }
    })

    const entry = wrapper.find('[data-testid="studio-v2-entry"]')
    expect(entry.exists()).toBe(true)
    await entry.trigger('click')

    expect(open).toHaveBeenCalledWith(
      'http://127.0.0.1:5174/projects',
      '_blank',
      'noopener,noreferrer'
    )
    expect(routerPush).not.toHaveBeenCalled()
  })
})
