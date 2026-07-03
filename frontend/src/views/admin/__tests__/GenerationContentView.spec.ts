import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi, beforeEach } from 'vitest'

import GenerationContentView from '../GenerationContentView.vue'

const { getStats, getSamples, getWeeklyReport } = vi.hoisted(() => ({
  getStats: vi.fn(),
  getSamples: vi.fn(),
  getWeeklyReport: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    generationContent: {
      getStats,
      getSamples,
      getWeeklyReport
    }
  }
}))

vi.mock('@/utils/format', () => ({
  formatBytes: (value: number) => `${value} B`,
  formatCompactNumber: (value: number) => String(value),
  formatCurrency: (value: number) => `$${value.toFixed(2)}`
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
    'admin.generationContent.title': 'Generation Content',
    'admin.generationContent.description': 'Production ledger',
    'admin.generationContent.live': 'Live',
    'admin.generationContent.exampleOff': 'Example off',
    'admin.generationContent.loading': 'Loading generation content...',
    'admin.generationContent.loadFailed': 'Failed to load generation content.',
    'admin.generationContent.weeklyReportLoadFailed': 'Weekly report failed to load.',
    'admin.generationContent.capturedToday': 'Captured today',
    'admin.generationContent.capturedWeek': 'Captured week',
    'admin.generationContent.distinctEmployees': 'Employees',
    'admin.generationContent.distinctTeams': 'Teams',
    'admin.generationContent.distinctModels': 'Models',
    'admin.generationContent.totalBytes': 'Bytes',
    'admin.generationContent.perDay': 'per day',
    'admin.generationContent.trendCaption': 'Seven-day trend',
    'admin.generationContent.weeklyEntries': 'Weekly entries',
    'admin.generationContent.weeklyCost': 'Weekly cost',
    'admin.generationContent.adopted': 'Adopted',
    'admin.generationContent.pending': 'Pending',
    'admin.generationContent.anomalies': 'Anomalies'
  }
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key
    })
  }
})

const AppLayoutStub = { template: '<div><slot /></div>' }
const StatCardStub = {
  props: ['title', 'value'],
  template: '<div class="stat-card"><span>{{ title }}</span><strong>{{ value }}</strong></div>'
}
const CaptureSparklineStub = { template: '<div data-test="sparkline" />' }
const ContentWallStub = {
  props: ['samples', 'isLive'],
  emits: ['updated'],
  template: '<div data-test="content-wall">{{ samples.length }} samples / live={{ isLive }}</div>'
}

describe('GenerationContentView', () => {
  beforeEach(() => {
    getStats.mockReset()
    getSamples.mockReset()
    getWeeklyReport.mockReset()
  })

  it('keeps stats and samples visible when only the weekly report fails', async () => {
    getStats.mockResolvedValue({
      captured_today: 1,
      captured_week: 2,
      distinct_employees: 1,
      distinct_teams: 1,
      distinct_models: 1,
      total_bytes: 128,
      daily_rate: 0.3,
      daily_series: [{ date: '2026-07-03', count: 1 }],
      is_live: true
    })
    getSamples.mockResolvedValue({
      is_live: true,
      samples: [{
        task_id: 42,
        employee_name: 'Alice',
        team_name: 'Drama',
        model: 'mock-video-v1',
        video_status: 'succeeded',
        cost_estimate: 0.08,
        created_at: '2026-07-03T01:00:00Z',
        prompt_preview: 'make a shot',
        response_preview: 'result asset',
        total_bytes: 128,
        adoption_status: 'pending',
        quality_score: null,
        adoption_notes: '',
        truncated: false
      }]
    })
    getWeeklyReport.mockRejectedValue(new Error('weekly report unavailable'))

    const wrapper = mount(GenerationContentView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          StatCard: StatCardStub,
          CaptureSparkline: CaptureSparklineStub,
          ContentWall: ContentWallStub
        }
      }
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Live')
    expect(wrapper.find('[data-test="content-wall"]').text()).toContain('1 samples')
    expect(wrapper.text()).toContain('Weekly report failed to load.')
  })

  it('keeps loaded samples live when stats fails but samples succeeds', async () => {
    getStats.mockRejectedValue(new Error('stats unavailable'))
    getSamples.mockResolvedValue({
      is_live: true,
      samples: [{
        task_id: 42,
        employee_name: 'Alice',
        team_name: 'Drama',
        model: 'mock-video-v1',
        video_status: 'succeeded',
        cost_estimate: 0.08,
        created_at: '2026-07-03T01:00:00Z',
        prompt_preview: 'make a shot',
        response_preview: 'result asset',
        total_bytes: 128,
        adoption_status: 'pending',
        quality_score: null,
        adoption_notes: '',
        truncated: false
      }]
    })
    getWeeklyReport.mockResolvedValue({
      period_start: '2026-07-01T00:00:00Z',
      period_end: '2026-07-08T00:00:00Z',
      entries: 1,
      video_tasks: 1,
      total_cost_estimate: 0.08,
      adopted_count: 0,
      rejected_count: 0,
      pending_count: 1,
      unreviewed_count: 0,
      adoption_rate: 0,
      anomalies: {
        failed_tasks: 0,
        missing_task_joins: 0,
        truncated_rows: 0
      },
      markdown: 'ok'
    })

    const wrapper = mount(GenerationContentView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          StatCard: StatCardStub,
          CaptureSparkline: CaptureSparklineStub,
          ContentWall: ContentWallStub
        }
      }
    })
    await flushPromises()

    expect(wrapper.find('[data-test="content-wall"]').text()).toContain('1 samples / live=true')
    expect(wrapper.text()).toContain('Failed to load generation content.')
  })
})
