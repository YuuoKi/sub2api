import { describe, expect, it } from 'vitest'
import {
  activeAppDialog,
  requestConfirmation,
  requestTextPrompt,
  settleAppDialog
} from '../useAppDialog'

describe('app dialog queue', () => {
  it('resolves cancellation without running the queued confirmation and advances in order', async () => {
    const confirmation = requestConfirmation({ message: 'Delete record', danger: true })
    const prompt = requestTextPrompt({ message: 'Restore record', label: 'Password', inputType: 'password' })

    expect(activeAppDialog.value?.kind).toBe('confirm')
    settleAppDialog(false)
    await expect(confirmation).resolves.toBe(false)
    await new Promise<void>((resolve) => queueMicrotask(resolve))

    expect(activeAppDialog.value?.kind).toBe('prompt')
    settleAppDialog('credential')
    await expect(prompt).resolves.toBe('credential')
  })
})
