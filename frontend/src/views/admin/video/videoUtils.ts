import type { VideoProvider, VideoTaskStatus, VideoTaskType } from '@/api/admin/video'

export const providerOptions: Array<{ value: VideoProvider; label: string }> = [
  { value: 'mock', label: 'Mock' },
  { value: 'seedance', label: 'Seedance 2.0' },
  { value: 'kling', label: 'Kling' },
]

export const statusOptions: Array<{ value: VideoTaskStatus; label: string }> = [
  { value: 'queued', label: 'Queued' },
  { value: 'submitted', label: 'Submitted' },
  { value: 'running', label: 'Running' },
  { value: 'succeeded', label: 'Succeeded' },
  { value: 'failed', label: 'Failed' },
  { value: 'cancelled', label: 'Cancelled' },
]

export const taskTypeOptions: Array<{ value: VideoTaskType; label: string }> = [
  { value: 'text_to_video', label: 'Text to video' },
  { value: 'image_to_video', label: 'Image to video' },
  { value: 'reference_to_video', label: 'Reference to video' },
]

export function providerLabel(provider: string): string {
  return providerOptions.find((item) => item.value === provider)?.label || provider
}

export function taskTypeLabel(taskType: string): string {
  return taskTypeOptions.find((item) => item.value === taskType)?.label || taskType
}

export function statusLabel(status: string): string {
  return statusOptions.find((item) => item.value === status)?.label || status
}

export function statusBadgeClass(status: string): string {
  switch (status) {
    case 'queued':
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-200'
    case 'submitted':
      return 'bg-blue-50 text-blue-700 dark:bg-blue-500/10 dark:text-blue-300'
    case 'running':
      return 'bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
    case 'succeeded':
      return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
    case 'failed':
      return 'bg-red-50 text-red-700 dark:bg-red-500/10 dark:text-red-300'
    case 'cancelled':
      return 'bg-slate-100 text-slate-700 dark:bg-slate-500/10 dark:text-slate-300'
    default:
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-200'
  }
}

export function providerBadgeClass(provider: string): string {
  switch (provider) {
    case 'mock':
      return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
    case 'seedance':
      return 'bg-sky-50 text-sky-700 dark:bg-sky-500/10 dark:text-sky-300'
    case 'kling':
      return 'bg-violet-50 text-violet-700 dark:bg-violet-500/10 dark:text-violet-300'
    default:
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-200'
  }
}

export function formatDate(value?: string | null): string {
  if (!value) return '-'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  return d.toLocaleString()
}

export function shortText(value: string, max = 96): string {
  if (!value) return '-'
  return value.length > max ? `${value.slice(0, max)}...` : value
}

export function isTerminalStatus(status: string): boolean {
  return status === 'succeeded' || status === 'failed' || status === 'cancelled'
}
