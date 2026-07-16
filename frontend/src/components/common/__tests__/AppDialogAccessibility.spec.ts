import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

import AppDialogHost from '../AppDialogHost.vue'
import BaseDialog from '../BaseDialog.vue'
import {
  activeAppDialog,
  requestConfirmation,
  settleAppDialog
} from '@/composables/useAppDialog'

afterEach(() => {
  if (activeAppDialog.value) settleAppDialog(false)
  document.body.innerHTML = ''
  document.body.classList.remove('modal-open')
})

describe('application dialog accessibility', () => {
  it('restores the trigger after AppDialogHost settles and unmounts the dialog', async () => {
    const trigger = document.createElement('button')
    trigger.textContent = 'Disable employee'
    document.body.appendChild(trigger)
    const wrapper = mount(AppDialogHost, {
      attachTo: document.body,
      global: { stubs: { Icon: true } }
    })

    trigger.focus()
    const result = requestConfirmation({ message: 'Disable this employee?' })
    await nextTick()

    const buttons = Array.from(
      document.body.querySelectorAll<HTMLButtonElement>('.modal-footer button')
    )
    expect(buttons).toHaveLength(2)
    buttons[1]?.click()

    await expect(result).resolves.toBe(true)
    await nextTick()
    expect(document.querySelector('[role="dialog"]')).toBeNull()
    expect(document.activeElement).toBe(trigger)

    wrapper.unmount()
  })

  it('cycles Tab and Shift+Tab inside the open dialog', async () => {
    const trigger = document.createElement('button')
    trigger.textContent = 'Open dialog'
    document.body.appendChild(trigger)
    trigger.focus()

    const wrapper = mount(BaseDialog, {
      attachTo: document.body,
      props: {
        show: true,
        title: 'Focusable dialog',
        showCloseButton: false
      },
      slots: {
        default: '<button id="first-dialog-action" type="button">First</button>',
        footer: '<button id="last-dialog-action" type="button">Last</button>'
      },
      global: { stubs: { Icon: true } }
    })
    await nextTick()

    const first = document.querySelector<HTMLButtonElement>('#first-dialog-action')
    const last = document.querySelector<HTMLButtonElement>('#last-dialog-action')
    expect(first).not.toBeNull()
    expect(last).not.toBeNull()
    expect(document.activeElement).toBe(first)

    last?.focus()
    document.dispatchEvent(
      new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true })
    )
    expect(document.activeElement).toBe(first)

    first?.focus()
    document.dispatchEvent(
      new KeyboardEvent('keydown', {
        key: 'Tab',
        shiftKey: true,
        bubbles: true,
        cancelable: true
      })
    )
    expect(document.activeElement).toBe(last)

    wrapper.unmount()
  })
  it('restores focus only once when close is followed by unmount', async () => {
    const trigger = document.createElement('button')
    trigger.textContent = 'Open dialog'
    document.body.appendChild(trigger)
    trigger.focus()
    const focusSpy = vi.spyOn(trigger, 'focus')

    const wrapper = mount(BaseDialog, {
      attachTo: document.body,
      props: {
        show: true,
        title: 'Single restore',
        showCloseButton: false
      },
      slots: {
        default: '<button type="button">Dialog action</button>'
      },
      global: { stubs: { Icon: true } }
    })
    await nextTick()
    focusSpy.mockClear()

    await wrapper.setProps({ show: false })
    expect(focusSpy).toHaveBeenCalledTimes(1)

    wrapper.unmount()
    expect(focusSpy).toHaveBeenCalledTimes(1)
  })
})
