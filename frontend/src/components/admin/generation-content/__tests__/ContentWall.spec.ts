import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi, beforeEach } from 'vitest'

import ContentWall from '../ContentWall.vue'
import type { GenerationSample } from '@/api/admin/generation_content'

const { updateAdoption } = vi.hoisted(() => ({
  updateAdoption: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    generationContent: {
      updateAdoption
    }
  }
}))

vi.mock('@/utils/format', () => ({
  formatBytes: (value: number) => `${value} B`,
  formatCurrency: (value: number) => `$${value.toFixed(2)}`,
  formatRelativeTime: () => 'just now'
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
    'admin.generationContent.adoptionStatus.pending': 'Pending',
    'admin.generationContent.adoptionStatus.adopted': 'Adopted',
    'admin.generationContent.adoptionStatus.rejected': 'Rejected',
    'admin.generationContent.adoptionSave': 'Save',
    'admin.generationContent.adoptionSaving': 'Saving...',
    'admin.generationContent.adoptionNotSaved.content_capture_disabled': 'Content capture is disabled; feedback was not saved.',
    'admin.generationContent.adoptionNotSaved.adoption_feedback_unavailable': 'Adoption feedback is unavailable; feedback was not saved.',
    'admin.generationContent.adoptionNotSaved.task_not_found': 'Task was not found; feedback was not saved.',
    'admin.generationContent.adoptionNotSaved.default': 'Feedback was not saved.',
    'admin.generationContent.adoptionSaveFailed': 'Save failed; feedback was not saved.',
    'admin.generationContent.promptLabel': 'Prompt',
    'admin.generationContent.responseLabel': 'Response',
    'admin.generationContent.truncated': 'Truncated',
    'admin.generationContent.exampleBannerTitle': 'No live samples',
    'admin.generationContent.exampleBannerDesc': 'Enable capture to show samples.'
  }
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key
    })
  }
})

function sample(overrides: Partial<GenerationSample> = {}): GenerationSample {
  return {
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
    truncated: false,
    ...overrides
  }
}

function mountWall(samples: GenerationSample[] = [sample()]) {
  return mount(ContentWall, {
    props: {
      isLive: true,
      samples
    }
  })
}

describe('ContentWall adoption feedback', () => {
  beforeEach(() => {
    updateAdoption.mockReset()
  })

  it('emits updated after the backend confirms feedback was saved', async () => {
    updateAdoption.mockResolvedValue({
      enabled: true,
      saved: true,
      task_id: 42,
      adoption_status: 'adopted'
    })
    const wrapper = mountWall()

    await wrapper.find('select').setValue('adopted')
    await wrapper.find('button').trigger('click')
    await flushPromises()

    expect(updateAdoption).toHaveBeenCalledWith(42, {
      adoption_status: 'adopted',
      quality_score: null,
      notes: ''
    })
    expect(wrapper.emitted('updated')).toHaveLength(1)
  })

  it('shows backend not-saved reason without refreshing samples', async () => {
    updateAdoption.mockResolvedValue({
      enabled: false,
      saved: false,
      reason: 'content_capture_disabled',
      task_id: 42,
      adoption_status: 'pending'
    })
    const wrapper = mountWall()
    const select = wrapper.find('select')
    const score = wrapper.find('input[type="number"]')
    const notes = wrapper.find('textarea')

    await select.setValue('adopted')
    await score.setValue('0.72')
    await notes.setValue('still needs review')
    await wrapper.find('button').trigger('click')
    await flushPromises()

    expect(updateAdoption).toHaveBeenCalledWith(42, {
      adoption_status: 'adopted',
      quality_score: 0.72,
      notes: 'still needs review'
    })
    expect(wrapper.emitted('updated')).toBeUndefined()
    expect(wrapper.text()).toContain('Content capture is disabled; feedback was not saved.')
    expect((select.element as HTMLSelectElement).value).toBe('adopted')
    expect((score.element as HTMLInputElement).value).toBe('0.72')
    expect((notes.element as HTMLTextAreaElement).value).toBe('still needs review')
  })

  it.each([
    ['adoption_feedback_unavailable', 'Adoption feedback is unavailable; feedback was not saved.'],
    ['task_not_found', 'Task was not found; feedback was not saved.'],
    ['unexpected_reason', 'Feedback was not saved.']
  ])('maps not-saved reason %s to a user-visible message', async (reason, message) => {
    updateAdoption.mockResolvedValue({
      enabled: true,
      saved: false,
      reason,
      task_id: 42,
      adoption_status: 'pending'
    })
    const wrapper = mountWall()

    await wrapper.find('button').trigger('click')
    await flushPromises()

    expect(wrapper.emitted('updated')).toBeUndefined()
    expect(wrapper.text()).toContain(message)
  })

  it('shows an error when the save request fails', async () => {
    updateAdoption.mockRejectedValue(new Error('network down'))
    const wrapper = mountWall()

    await wrapper.find('button').trigger('click')
    await flushPromises()

    expect(wrapper.emitted('updated')).toBeUndefined()
    expect(wrapper.text()).toContain('Save failed; feedback was not saved.')
  })

  it('disables the save button while the save request is in flight', async () => {
    let resolveSave: (value: unknown) => void = () => {}
    updateAdoption.mockReturnValue(new Promise((resolve) => {
      resolveSave = resolve
    }))
    const wrapper = mountWall()
    const button = wrapper.find('button')

    await button.trigger('click')
    expect(button.attributes('disabled')).toBeDefined()

    resolveSave({
      enabled: true,
      saved: true,
      task_id: 42,
      adoption_status: 'pending'
    })
    await flushPromises()
  })
})
