import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import TextPromptDialog from '../TextPromptDialog.vue'

afterEach(() => {
  document.body.innerHTML = ''
  document.body.classList.remove('modal-open')
})

describe('TextPromptDialog', () => {
  it('provides an associated label, masks passwords, and restores focus after cancel', async () => {
    const trigger = document.createElement('button')
    document.body.appendChild(trigger)
    trigger.focus()

    const wrapper = mount(TextPromptDialog, {
      attachTo: document.body,
      props: {
        show: false,
        title: 'Restore backup',
        message: 'This replaces current data.',
        label: 'Administrator password',
        inputType: 'password',
        confirmText: 'Restore',
        cancelText: 'Cancel'
      },
      global: {
        stubs: { Icon: true, teleport: true }
      }
    })

    await wrapper.setProps({ show: true })
    await nextTick()

    const input = wrapper.get('input')
    expect(input.attributes('type')).toBe('password')
    expect(wrapper.get('label').attributes('for')).toBe(input.attributes('id'))
    expect(document.activeElement).toBe(input.element)

    await input.setValue('secret-value')
    await wrapper.get('button.btn-secondary').trigger('click')
    expect(wrapper.emitted('cancel')).toHaveLength(1)
    expect(wrapper.emitted('confirm')).toBeUndefined()

    await wrapper.setProps({ show: false })
    await nextTick()
    expect(document.activeElement).toBe(trigger)
    await wrapper.setProps({ show: true })
    await nextTick()
    expect((wrapper.get('input').element as HTMLInputElement).value).toBe('')

    wrapper.unmount()
  })

  it('submits only a non-empty required value and supports Escape cancellation', async () => {
    const wrapper = mount(TextPromptDialog, {
      attachTo: document.body,
      props: {
        show: true,
        title: 'Enter value',
        label: 'Value',
        confirmText: 'Continue',
        cancelText: 'Cancel'
      },
      global: {
        stubs: { Icon: true, teleport: true }
      }
    })

    const confirmButton = wrapper.get('button.btn-danger')
    expect(confirmButton.attributes('disabled')).toBeDefined()
    await confirmButton.trigger('click')
    expect(wrapper.emitted('confirm')).toBeUndefined()

    await wrapper.get('input').setValue('approved')
    await confirmButton.trigger('click')
    expect(wrapper.emitted('confirm')).toEqual([['approved']])

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await nextTick()
    expect(wrapper.emitted('cancel')).toHaveLength(1)

    wrapper.unmount()
  })
})
