import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import BossOverviewView from '../BossOverviewView.vue'

const mocks = vi.hoisted(() => ({
  getUsageTrend: vi.fn(),
  getModelStats: vi.fn(),
  getUserSpendingRanking: vi.fn(),
  accountsList: vi.fn(),
  showError: vi.fn(),
  routerPush: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    dashboard: {
      getUsageTrend: mocks.getUsageTrend,
      getModelStats: mocks.getModelStats,
      getUserSpendingRanking: mocks.getUserSpendingRanking,
    },
    accounts: {
      list: mocks.accountsList,
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: mocks.showError, showSuccess: vi.fn() }),
}))

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRouter: () => ({ push: mocks.routerPush }),
  }
})

vi.mock('vue-chartjs', () => ({
  Line: { template: '<div data-test="line-chart" />' },
}))

const AppLayoutStub = { template: '<div><slot /></div>' }
const IconStub = { template: '<i />' }
const RouterLinkStub = { props: ['to'], template: '<a><slot /></a>' }
const AnimatedNumberStub = { props: ['value', 'format'], template: '<span>{{ format ? format(value) : value }}</span>' }

function mountView() {
  return mount(BossOverviewView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        Icon: IconStub,
        RouterLink: RouterLinkStub,
        AnimatedNumber: AnimatedNumberStub,
      },
    },
  })
}

function emptyResponses() {
  mocks.getUsageTrend.mockResolvedValue({ trend: [] })
  mocks.getModelStats.mockResolvedValue({ models: [] })
  mocks.getUserSpendingRanking.mockResolvedValue({ ranking: [], total_actual_cost: 0, total_requests: 0, total_tokens: 0 })
  mocks.accountsList.mockResolvedValue({ items: [], total: 0 })
}

function modelRow(name: string, cost: number, requests = 1) {
  return {
    model: name,
    requests,
    input_tokens: 0,
    output_tokens: 0,
    cache_creation_tokens: 0,
    cache_read_tokens: 0,
    total_tokens: 0,
    cost,
    actual_cost: cost,
  }
}

describe('BossOverviewView model distribution', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    emptyResponses()
  })

  it('aggregates everything beyond Top 6 into a single 其他 row with the remainder share', async () => {
    mocks.getModelStats.mockResolvedValue({
      models: [
        modelRow('m1', 80),
        modelRow('m2', 70),
        modelRow('m3', 60),
        modelRow('m4', 50),
        modelRow('m5', 40),
        modelRow('m6', 30),
        modelRow('m7', 20),
        modelRow('m8', 10),
      ],
    })
    mocks.getUsageTrend.mockResolvedValue({ trend: [{ date: '2026-07-01', actual_cost: 1 }] })

    const wrapper = mountView()
    await flushPromises()

    const other = wrapper.findAll('button').find((button) => button.text().includes('其他'))
    expect(other).toBeTruthy()
    // 余项 20 + 10 = 30，占总量 360 的 8.3%
    expect(other!.text()).toContain('8.3%')
    // Top 6 原样保留，m7/m8 不单独出现
    expect(wrapper.text()).toContain('m6')
    expect(wrapper.text()).not.toContain('m7')
    expect(wrapper.text()).not.toContain('m8')
  })

  it('does not aggregate when there are 6 or fewer models', async () => {
    mocks.getModelStats.mockResolvedValue({
      models: [modelRow('m1', 50), modelRow('m2', 30), modelRow('m3', 20)],
    })
    mocks.getUsageTrend.mockResolvedValue({ trend: [{ date: '2026-07-01', actual_cost: 1 }] })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('m3')
    expect(wrapper.text()).not.toContain('其他')
  })
})

describe('BossOverviewView load failure vs empty state', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    emptyResponses()
  })

  it('shows the three-step guide card on a successful empty load', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('三步上手')
    expect(wrapper.find('[data-test="overview-load-error"]').exists()).toBe(false)
  })

  it('shows a neutral error hint instead of the guide card when loading fails', async () => {
    mocks.getUsageTrend.mockRejectedValue(new Error('dashboard backend down'))
    const wrapper = mountView()
    await flushPromises()

    expect(mocks.showError).toHaveBeenCalled()
    expect(wrapper.text()).not.toContain('三步上手')
    expect(wrapper.find('[data-test="overview-load-error"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="overview-load-error"]').text()).toContain('加载失败')
  })
})
