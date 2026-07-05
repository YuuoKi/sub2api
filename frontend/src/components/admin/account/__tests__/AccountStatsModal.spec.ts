import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import AccountStatsModal from '../AccountStatsModal.vue'

const { getStats } = vi.hoisted(() => ({
  getStats: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getStats
    }
  }
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

vi.mock('vue-chartjs', () => ({
  Line: { template: '<div data-test="line-chart"></div>' }
}))

const BaseDialogStub = {
  props: ['show', 'title'],
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
}
const ModelDistributionChartStub = {
  props: ['usdCnyRate'],
  template: '<div data-test="model-chart">rate={{ usdCnyRate }}</div>'
}
const EndpointDistributionChartStub = {
  props: ['usdCnyRate'],
  template: '<div data-test="endpoint-chart">rate={{ usdCnyRate }}</div>'
}

function accountStatsResponse() {
  return {
    usd_cny_rate: 7.5,
    history: [],
    summary: {
      days: 30,
      actual_days_used: 1,
      total_cost: 1,
      total_user_cost: 0.5,
      total_standard_cost: 2,
      total_requests: 3,
      total_tokens: 4,
      avg_daily_cost: 1,
      avg_daily_user_cost: 0.5,
      avg_daily_requests: 3,
      avg_daily_tokens: 4,
      avg_duration_ms: 120,
      today: null,
      highest_cost_day: null,
      highest_request_day: null
    },
    models: [],
    endpoints: [],
    upstream_endpoints: []
  }
}

describe('admin AccountStatsModal', () => {
  beforeEach(() => {
    getStats.mockReset()
    getStats.mockResolvedValue(accountStatsResponse())
  })

  it('uses the account stats USD/CNY rate for cards and distribution charts', async () => {
    const wrapper = mount(AccountStatsModal, {
      props: {
        show: false,
        account: {
          id: 42,
          name: 'Seedance Pool',
          platform: 'volcengine',
          type: 'apikey',
          status: 'active'
        } as any
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          LoadingSpinner: true,
          Icon: true,
          ModelDistributionChart: ModelDistributionChartStub,
          EndpointDistributionChart: EndpointDistributionChartStub
        }
      }
    })
    await wrapper.setProps({ show: true })
    await flushPromises()

    expect(wrapper.text()).toContain('¥7.50')
    expect(wrapper.find('[data-test="model-chart"]').text()).toContain('rate=7.5')
    expect(wrapper.findAll('[data-test="endpoint-chart"]').every((chart) => chart.text().includes('rate=7.5'))).toBe(true)
  })
})
