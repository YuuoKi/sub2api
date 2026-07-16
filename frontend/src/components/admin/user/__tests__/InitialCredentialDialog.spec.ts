import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import InitialCredentialDialog from '../InitialCredentialDialog.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

describe('InitialCredentialDialog', () => {
  it('cannot close before copy or download acknowledgement and clears after close', async () => {
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText: vi.fn().mockResolvedValue(undefined) }
    })

    const wrapper = mount(InitialCredentialDialog, {
      props: {
        show: true,
        email: 'employee@example.com',
        credential: {
          temporary_password: 'one-time-secret',
          expires_at: '2026-07-17T00:00:00Z'
        }
      },
      global: {
        stubs: {
          BaseDialog: {
            emits: ['close'],
            template: '<div><slot /><slot name="footer" /></div>'
          }
        }
      }
    })

    expect(wrapper.get('[data-test="credential-close"]').attributes('disabled')).toBeDefined()
    await wrapper.get('[data-test="credential-copy"]').trigger('click')
    expect(wrapper.get('[data-test="credential-close"]').attributes('disabled')).toBeUndefined()
    await wrapper.get('[data-test="credential-close"]').trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(1)
  })
})
