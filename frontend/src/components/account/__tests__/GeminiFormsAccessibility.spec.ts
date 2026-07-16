import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import type { ComponentPublicInstance } from 'vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

import GeminiApiKeyEditForm from '../GeminiApiKeyEditForm.vue'
import GeminiVertexEditForm from '../GeminiVertexEditForm.vue'

const sharedAdvancedProps = {
  proxies: [],
  proxyId: null,
  concurrency: 2,
  loadFactor: null,
  priority: 1,
  rateMultiplier: 1,
  review: false
}

afterEach(() => {
  document.body.innerHTML = ''
})

function expectEveryLabelToOwnAUniqueControl(
  wrappers: Array<VueWrapper<ComponentPublicInstance>>
): void {
  const labels = wrappers.flatMap((wrapper) => wrapper.findAll('label'))
  const controlIds = labels.map((label) => label.attributes('for'))

  expect(labels.length).toBeGreaterThan(0)
  expect(controlIds.every(Boolean)).toBe(true)
  expect(new Set(controlIds).size).toBe(controlIds.length)

  for (const label of labels) {
    const element = label.element as HTMLLabelElement
    expect(element.control).not.toBeNull()
    expect(element.control?.id).toBe(label.attributes('for'))
    expect(label.text().trim()).not.toBe('')
  }
}

describe('Gemini edit form accessible names', () => {
  it('associates every API Key label with a unique control across simultaneous forms', () => {
    const wrappers = [1, 2].map(() =>
      mount(GeminiApiKeyEditForm, {
        attachTo: document.body,
        props: {
          ...sharedAdvancedProps,
          apiKey: '',
          hasExistingApiKey: true,
          baseUrl: 'https://generativelanguage.googleapis.com',
          baseUrlHint: 'Gemini endpoint'
        },
        global: { stubs: { Icon: true } }
      })
    )

    expectEveryLabelToOwnAUniqueControl(wrappers)
    wrappers.forEach((wrapper) => wrapper.unmount())
  })

  it('associates every Vertex label with a unique control across simultaneous forms', () => {
    const wrappers = [1, 2].map(() =>
      mount(GeminiVertexEditForm, {
        attachTo: document.body,
        props: {
          ...sharedAdvancedProps,
          projectId: 'demo-project',
          clientEmail: 'owner@example.com',
          location: 'us-central1',
          hasServiceAccountJson: true
        },
        global: { stubs: { Icon: true } }
      })
    )

    expectEveryLabelToOwnAUniqueControl(wrappers)
    wrappers.forEach((wrapper) => wrapper.unmount())
  })
})
