import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import VideoTaskDetailView from '../VideoTaskDetailView.vue'

const mocks = vi.hoisted(() => ({
  getTask: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { id: '41' } }),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    video: {
      getTask: mocks.getTask,
      createAssetHandoff: vi.fn(),
    },
  },
}))

const AppLayoutStub = defineComponent({
  name: 'AppLayout',
  template: '<main><slot /></main>',
})

const RouterLinkStub = defineComponent({
  name: 'RouterLink',
  template: '<a><slot /></a>',
})

describe('VideoTaskDetailView evidence contract', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    window.localStorage.clear()
    mocks.getTask.mockResolvedValue({
      id: 41,
      api_key_id: 12,
      group_id: 13,
      provider_account_id: 7,
      provider: 'seedance',
      model: 'doubao-seedance-2-0-260128',
      request_model: 'doubao-seedance-2-0-260128',
      request_duration_seconds: 4,
      request_resolution: '720p',
      upstream_model: null,
      upstream_duration_seconds: null,
      upstream_resolution: null,
      billing_model: null,
      billing_duration_seconds: null,
      billing_resolution: null,
      task_type: 'text_to_video',
      prompt: 'test prompt',
      status: 'succeeded',
      upstream_task_id: 'upstream-41',
      result_url: '',
      last_frame_url: '',
      duration_seconds: 4,
      resolution: '720p',
      usage_total_tokens: 321,
      error_message: '',
      provider_error_code: '',
      provider_error_message: '',
      reserved_cost_usd: 0.2,
      reservation_state: 'captured',
      reserved_at: '2026-07-16T08:00:00Z',
      cost_amount: 0.1,
      provider_actual_cost_usd: 0.08,
      balance_before_usd: null,
      balance_after_usd: null,
      balance_delta_usd: null,
      authorization_consumed_at: null,
      authorization_consumed_by: null,
      currency: 'USD',
      real_dispatch_count: 1,
      dispatch_state: 'accepted',
      created_by: 9,
      created_at: '2026-07-16T08:00:00Z',
      updated_at: '2026-07-16T08:01:00Z',
      completed_at: '2026-07-16T08:01:00Z',
    })
  })

  it('separates request, upstream, and billing facts without copying request specs into unavailable fields', async () => {
    const wrapper = mount(VideoTaskDetailView, {
      global: { stubs: { AppLayout: AppLayoutStub, RouterLink: RouterLinkStub } },
    })
    await flushPromises()

    const blocks = wrapper.findAll('.video-task-spec-block')
    const request = blocks.find((block) => block.get('h3').text() === '请求规格')
    const upstream = blocks.find((block) => block.get('h3').text() === '上游回传规格')
    const billing = blocks.find((block) => block.get('h3').text() === '计费规格')

    expect(request?.text()).toContain('4 秒')
    expect(request?.text()).toContain('720p')
    expect(upstream?.text()).toContain('不可用（后端未提供）')
    expect(billing?.text()).toContain('0.2 USD')
    expect(billing?.text()).toContain('captured')
    expect(billing?.text()).toContain('0.08 USD')
    expect(billing?.text()).not.toContain('4 秒')
    expect(billing?.text()).not.toContain('720p')
    expect(wrapper.text()).toContain('授权消费状态')
    expect(wrapper.text()).toContain('不可用（后端未提供）')
    expect(wrapper.text()).toContain('余额差分')
    expect(wrapper.text()).toContain('不可用（后端未提供）')
  })

  it('compares every available raw specification field and highlights mismatches', async () => {
    mocks.getTask.mockResolvedValue({
      ...(await mocks.getTask()),
      request_model: 'request-model',
      request_duration_seconds: 4,
      request_resolution: '720p',
      upstream_model: 'upstream-model',
      upstream_duration_seconds: 4,
      upstream_resolution: '720p',
      billing_model: 'upstream-model',
      billing_duration_seconds: 4,
      billing_resolution: '1080p',
      balance_before_usd: 9.8,
      balance_after_usd: 9.7,
      balance_delta_usd: -0.1,
      authorization_consumed_at: '2026-07-16T08:00:30Z',
      authorization_consumed_by: 5,
    })

    const wrapper = mount(VideoTaskDetailView, {
      global: { stubs: { AppLayout: AppLayoutStub, RouterLink: RouterLinkStub } },
    })
    await flushPromises()

    expect(wrapper.get('.video-spec-comparison').text()).toContain('不一致')
    const mismatches = wrapper.findAll('.video-spec-value--mismatch').map((field) => field.text())
    expect(mismatches).toEqual(expect.arrayContaining(['request-model', 'upstream-model', '720p', '1080p']))
    expect(wrapper.text()).toContain('9.8 → 9.7（-0.1 USD）')
    expect(wrapper.text()).toContain('已消费（授权人 #5')
  })
})
