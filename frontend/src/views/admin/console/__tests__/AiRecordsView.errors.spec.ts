import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AiRecordsView from '../AiRecordsView.vue'

const mocks = vi.hoisted(() => ({
  usageList: vi.fn(),
  usageGetStats: vi.fn(),
  getSamples: vi.fn(),
  getWeeklyReport: vi.fn(),
  usersList: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    usage: {
      list: mocks.usageList,
      getStats: mocks.usageGetStats,
    },
    generationContent: {
      getSamples: mocks.getSamples,
      getWeeklyReport: mocks.getWeeklyReport,
    },
    users: {
      list: mocks.usersList,
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess: mocks.showSuccess, showError: mocks.showError }),
}))

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRoute: () => ({ query: {} }),
  }
})

const AppLayoutStub = { template: '<div><slot /></div>' }
const IconStub = { template: '<i />' }
const RouterLinkStub = { template: '<a><slot /></a>' }
const ContentWallStub = {
  props: ['samples', 'isLive', 'usdCnyRate'],
  emits: ['updated'],
  template: '<div data-test="content-wall">{{ samples.length }} samples / live={{ isLive }}</div>',
}

const GLOBAL_STUBS = {
  global: {
    stubs: { AppLayout: AppLayoutStub, Icon: IconStub, RouterLink: RouterLinkStub, ContentWall: ContentWallStub },
  },
}

function resolvedLogsPage() {
  return { items: [], total: 0 }
}

function resolvedStats() {
  return {
    total_requests: 0,
    total_input_tokens: 0,
    total_output_tokens: 0,
    total_cache_tokens: 0,
    total_cache_creation_tokens: 0,
    total_cache_read_tokens: 0,
    total_tokens: 0,
    total_cost: 0,
    total_actual_cost: 0,
    total_account_cost: 0,
    average_duration_ms: 0,
  }
}

describe('AiRecordsView error surfacing', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.usageList.mockResolvedValue(resolvedLogsPage())
    mocks.usageGetStats.mockResolvedValue(resolvedStats())
    mocks.getSamples.mockResolvedValue({ samples: [], is_live: true })
    mocks.getWeeklyReport.mockResolvedValue(null)
    mocks.usersList.mockResolvedValue({ items: [], total: 0 })
  })

  it('shows an inline error banner (not the empty state) when loadSamples rejects', async () => {
    mocks.getSamples.mockRejectedValue(new Error('samples backend down'))

    const wrapper = mount(AiRecordsView, GLOBAL_STUBS)
    await flushPromises()

    // Switch to the prompts tab where the samples wall lives.
    await wrapper.find('[data-test="tab-prompts"]').trigger('click')

    expect(wrapper.find('[data-test="samples-error"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="samples-error"]').text()).toContain('samples backend down')
    // ContentWall (and its own "honest empty" banner) must not render while the error is showing.
    expect(wrapper.find('[data-test="content-wall"]').exists()).toBe(false)
  })

  it('shows an inline error banner (not the empty summary state) when loadWeeklyReport rejects', async () => {
    mocks.getWeeklyReport.mockRejectedValue(new Error('weekly report backend down'))

    const wrapper = mount(AiRecordsView, GLOBAL_STUBS)
    await flushPromises()

    await wrapper.find('[data-test="tab-prompts"]').trigger('click')

    expect(wrapper.find('[data-test="weekly-report-error"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="weekly-report-error"]').text()).toContain('weekly report backend down')
    expect(wrapper.text()).not.toContain('经验周报摘要')
  })

  it('clears a previous samples error once a subsequent reload succeeds', async () => {
    mocks.getSamples.mockRejectedValueOnce(new Error('samples backend down'))
    const wrapper = mount(AiRecordsView, GLOBAL_STUBS)
    await flushPromises()
    await wrapper.find('[data-test="tab-prompts"]').trigger('click')
    expect(wrapper.find('[data-test="samples-error"]').exists()).toBe(true)

    mocks.getSamples.mockResolvedValue({ samples: [], is_live: true })
    await wrapper.find('[data-test="reload"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-test="samples-error"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="content-wall"]').exists()).toBe(true)
  })

  it('does not swallow the samples rejection as a toast (it is surfaced inline instead)', async () => {
    mocks.getSamples.mockRejectedValue(new Error('samples backend down'))

    const wrapper = mount(AiRecordsView, GLOBAL_STUBS)
    await flushPromises()

    expect(mocks.showError).not.toHaveBeenCalledWith(expect.stringContaining('samples backend down'))
    expect(wrapper.exists()).toBe(true)
  })
})

describe('AiRecordsView weekly report summary', () => {
  const weeklyReportFixture = (overrides: Record<string, unknown> = {}) => ({
    period_start: '2026-07-13T00:00:00Z',
    period_end: '2026-07-20T00:00:00Z',
    entries: 12,
    video_tasks: 5,
    total_cost_estimate: 1.5,
    usd_cny_rate: 7.2,
    adopted_count: 3,
    rejected_count: 1,
    pending_count: 2,
    unreviewed_count: 0,
    adoption_rate: 0.6,
    anomalies: { failed_tasks: 0, missing_task_joins: 0, truncated_rows: 0 },
    markdown: '# Weekly Production Ledger\nPeriod: ...\nEntries: 12',
    ...overrides,
  })

  beforeEach(() => {
    vi.clearAllMocks()
    mocks.usageList.mockResolvedValue(resolvedLogsPage())
    mocks.usageGetStats.mockResolvedValue(resolvedStats())
    mocks.getSamples.mockResolvedValue({ samples: [], is_live: true })
    mocks.usersList.mockResolvedValue({ items: [], total: 0 })
  })

  it('renders structured Chinese fields instead of the raw English ledger markdown', async () => {
    mocks.getWeeklyReport.mockResolvedValue(weeklyReportFixture())
    const wrapper = mount(AiRecordsView, GLOBAL_STUBS)
    await flushPromises()
    await wrapper.find('[data-test="tab-prompts"]').trigger('click')

    const panel = wrapper.find('[data-test="weekly-report"]')
    expect(panel.exists()).toBe(true)
    for (const label of ['周期', '条目', '视频任务', '成本估算', '采纳情况', '异常']) {
      expect(panel.text()).toContain(label)
    }
    expect(panel.text()).toContain('12')
    expect(panel.text()).toContain('采纳率 60%')
    expect(panel.text()).not.toContain('Weekly Production Ledger')
  })

  it('hides the whole panel when entries and cost are both zero (empty state card takes over)', async () => {
    mocks.getWeeklyReport.mockResolvedValue(weeklyReportFixture({ entries: 0, video_tasks: 0, total_cost_estimate: 0 }))
    const wrapper = mount(AiRecordsView, GLOBAL_STUBS)
    await flushPromises()
    await wrapper.find('[data-test="tab-prompts"]').trigger('click')

    expect(wrapper.find('[data-test="weekly-report"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('经验周报摘要')
  })

  it('still shows the panel on zero data when failed-task anomalies exist (失败信号不能吞)', async () => {
    mocks.getWeeklyReport.mockResolvedValue(weeklyReportFixture({
      entries: 0,
      video_tasks: 0,
      total_cost_estimate: 0,
      anomalies: { failed_tasks: 2, missing_task_joins: 0, truncated_rows: 0 },
    }))
    const wrapper = mount(AiRecordsView, GLOBAL_STUBS)
    await flushPromises()
    await wrapper.find('[data-test="tab-prompts"]').trigger('click')

    const panel = wrapper.find('[data-test="weekly-report"]')
    expect(panel.exists()).toBe(true)
    expect(panel.text()).toContain('失败任务 2')
  })
})

describe('AiRecordsView AbortSignal wiring', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.usageList.mockResolvedValue(resolvedLogsPage())
    mocks.usageGetStats.mockResolvedValue(resolvedStats())
    mocks.getSamples.mockResolvedValue({ samples: [], is_live: true })
    mocks.getWeeklyReport.mockResolvedValue(null)
    mocks.usersList.mockResolvedValue({ items: [], total: 0 })
  })

  it('passes an AbortSignal into every concurrently-loaded request', async () => {
    const wrapper = mount(AiRecordsView, GLOBAL_STUBS)
    await flushPromises()

    expect(mocks.usageList.mock.calls[0][1]?.signal).toBeInstanceOf(AbortSignal)
    expect(mocks.usageGetStats.mock.calls[0][1]?.signal).toBeInstanceOf(AbortSignal)
    expect(mocks.getSamples.mock.calls[0][0]?.signal).toBeInstanceOf(AbortSignal)
    expect(mocks.getWeeklyReport.mock.calls[0][0]?.signal).toBeInstanceOf(AbortSignal)
    expect(mocks.usersList.mock.calls[0][3]?.signal).toBeInstanceOf(AbortSignal)

    wrapper.unmount()
  })

  it('aborts all in-flight requests on unmount', async () => {
    const wrapper = mount(AiRecordsView, GLOBAL_STUBS)
    await flushPromises()

    const logsSignal: AbortSignal = mocks.usageList.mock.calls[0][1].signal
    const staffSignal: AbortSignal = mocks.usersList.mock.calls[0][3].signal
    expect(logsSignal.aborted).toBe(false)
    expect(staffSignal.aborted).toBe(false)

    wrapper.unmount()

    expect(logsSignal.aborted).toBe(true)
    expect(staffSignal.aborted).toBe(true)
  })

  it('aborts the previous reload family in-flight request when a new reload starts', async () => {
    const wrapper = mount(AiRecordsView, GLOBAL_STUBS)
    await flushPromises()

    const firstSignal: AbortSignal = mocks.usageList.mock.calls[0][1].signal
    expect(firstSignal.aborted).toBe(false)

    await wrapper.find('[data-test="reload"]').trigger('click')

    expect(firstSignal.aborted).toBe(true)
    await flushPromises()
  })
})
