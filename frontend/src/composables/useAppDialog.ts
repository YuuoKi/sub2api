import { readonly, shallowRef } from 'vue'

export interface ConfirmationOptions {
  title?: string
  message: string
  confirmText?: string
  cancelText?: string
  danger?: boolean
}

export interface TextPromptOptions extends ConfirmationOptions {
  label: string
  inputType?: 'text' | 'password'
  placeholder?: string
  required?: boolean
}

interface ConfirmDialogRequest extends ConfirmationOptions {
  id: number
  kind: 'confirm'
}

interface TextPromptDialogRequest extends TextPromptOptions {
  id: number
  kind: 'prompt'
}

export type AppDialogRequest = ConfirmDialogRequest | TextPromptDialogRequest
type DialogResult = boolean | string | null

interface QueuedDialog {
  request: AppDialogRequest
  resolve: (result: DialogResult) => void
}

const activeDialogState = shallowRef<AppDialogRequest | null>(null)
const queue: QueuedDialog[] = []
let activeResolver: QueuedDialog['resolve'] | null = null
let nextDialogId = 0

export const activeAppDialog = readonly(activeDialogState)

function activateNextDialog(): void {
  if (activeDialogState.value || queue.length === 0) return
  const next = queue.shift()
  if (!next) return
  activeResolver = next.resolve
  activeDialogState.value = next.request
}

function enqueueDialog(request: Omit<ConfirmDialogRequest, 'id'>, resolve: QueuedDialog['resolve']): void
function enqueueDialog(request: Omit<TextPromptDialogRequest, 'id'>, resolve: QueuedDialog['resolve']): void
function enqueueDialog(
  request: Omit<ConfirmDialogRequest, 'id'> | Omit<TextPromptDialogRequest, 'id'>,
  resolve: QueuedDialog['resolve']
): void {
  queue.push({ request: { ...request, id: ++nextDialogId } as AppDialogRequest, resolve })
  activateNextDialog()
}

export function requestConfirmation(options: ConfirmationOptions): Promise<boolean> {
  return new Promise((resolve) => {
    enqueueDialog({ kind: 'confirm', ...options }, (result) => resolve(result === true))
  })
}

export function requestTextPrompt(options: TextPromptOptions): Promise<string | null> {
  return new Promise((resolve) => {
    enqueueDialog({ kind: 'prompt', ...options }, (result) => {
      resolve(typeof result === 'string' ? result : null)
    })
  })
}

export function settleAppDialog(result: DialogResult): void {
  const resolver = activeResolver
  activeResolver = null
  activeDialogState.value = null
  resolver?.(result)
  queueMicrotask(activateNextDialog)
}
