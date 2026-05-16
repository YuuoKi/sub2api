import type { VideoProvider, VideoTask, VideoTaskCreatePayload, VideoTaskStatus, VideoTaskType } from '@/api/admin/video'
import { isVideoGatewayDemoMode, videoGatewayDisplayProvider } from '@/utils/productMode'

export const providerOptions: Array<{ value: VideoProvider; label: string }> = [
  { value: 'mock', label: isVideoGatewayDemoMode ? '演示通道' : 'Mock Provider' },
  { value: 'seedance', label: 'Seedance 2.0' },
  { value: 'kling', label: 'Kling' },
]

export const statusOptions: Array<{ value: VideoTaskStatus; label: string }> = [
  { value: 'queued', label: '排队中' },
  { value: 'submitted', label: '已提交' },
  { value: 'running', label: '处理中' },
  { value: 'succeeded', label: '已成功' },
  { value: 'failed', label: '已失败' },
  { value: 'cancelled', label: '已取消' },
]

export const taskTypeOptions: Array<{ value: VideoTaskType; label: string }> = [
  { value: 'text_to_video', label: '文生视频' },
  { value: 'image_to_video', label: '图生视频' },
  { value: 'reference_to_video', label: '参考视频生成' },
]

export function providerLabel(provider: string): string {
  return videoGatewayDisplayProvider(provider, providerOptions.find((item) => item.value === provider)?.label)
}

export function providerDisplayName(provider: { provider: string; display_name?: string }): string {
  return videoGatewayDisplayProvider(provider.provider, provider.display_name)
}

export function modelDisplayName(provider: string, model?: string): string {
  if (isVideoGatewayDemoMode && provider === 'mock') return '演示视频模型'
  if (isVideoGatewayDemoMode && provider === 'seedance') return 'seedance-2-0-pro'
  if (isVideoGatewayDemoMode && provider === 'kling') return 'kling-v1'
  return model?.trim() || '-'
}

export function promptDisplayText(prompt?: string): string {
  const value = prompt?.trim() || ''
  if (!value) return '-'
  if (!isVideoGatewayDemoMode) return value
  const normalized = value.toLowerCase()
  if (normalized.includes('mock') || normalized.includes('p0.5') || normalized.includes('local forced')) {
    return '企业 AI 视频网关演示任务'
  }
  return value.replace(/\[fail\]\s*/gi, '').trim() || '企业 AI 视频网关演示任务'
}

export function taskTypeLabel(taskType: string): string {
  return taskTypeOptions.find((item) => item.value === taskType)?.label || taskType
}

export function statusLabel(status: string): string {
  return statusOptions.find((item) => item.value === status)?.label || status
}

export function providerDescription(provider: string): string {
  switch (provider) {
    case 'mock':
      return '演示通道用于本地闭环验证，不调用外部模型。'
    case 'seedance':
      return 'Seedance 2.0 预留通道，待配置 API Key 后启用真实调用。'
    case 'kling':
      return 'Kling 预留通道，待配置 API Key 后启用真实调用。'
    default:
      return '视频模型通道'
  }
}

export function providerKeyLabel(configured: boolean, maskedKey?: string): string {
  if (!configured) return '未配置 Key'
  if (isVideoGatewayDemoMode) return '已脱敏'
  return maskedKey ? `已脱敏：${maskedKey}` : '已配置 Key'
}

export function providerEnabledLabel(enabled: boolean): string {
  return enabled ? '启用' : '停用'
}

export function providerRuntimeStatus(provider: { provider: string; enabled: boolean; api_key_configured: boolean }): string {
  if (!provider.enabled) return '停用'
  if (provider.provider === 'mock') return '可演示'
  if (!provider.api_key_configured) return '待配置'
  return '已配置'
}

export function providerRuntimeStatusClass(status: string): string {
  switch (status) {
    case '可演示':
    case '已配置':
      return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
    case '待配置':
      return 'bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
    case '停用':
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-200'
    default:
      return 'bg-red-50 text-red-700 dark:bg-red-500/10 dark:text-red-300'
  }
}

export function eventTypeLabel(eventType: string): string {
  switch (eventType) {
    case 'queued':
      return '任务已入队'
    case 'submitted':
      return '已提交到通道'
    case 'running':
      return '通道处理中'
    case 'succeeded':
      return '生成成功'
    case 'failed':
      return '生成失败'
    case 'cancelled':
      return '任务已取消'
    case 'timeout':
      return '任务超时'
    default:
      return eventType
  }
}

export function eventMessageLabel(message: string, eventType: string): string {
  const normalized = message.trim().toLowerCase()
  if (!normalized) return eventTypeLabel(eventType)
  if (normalized === 'video task queued') return '任务已进入视频队列'
  if (normalized === 'video task submitted to provider') return '任务已提交到模型通道'
  if (normalized === 'video task status updated') return '任务状态已更新'
  if (normalized === 'video task succeeded') return '视频生成成功'
  if (normalized === 'video task failed') return '视频生成失败'
  if (normalized === 'video task cancelled') return '任务已取消'
  if (normalized === 'video task timed out') return '任务处理超时'
  return message
}

export function providerTestMessage(message: string): string {
  const normalized = message.trim().toLowerCase()
  if (normalized === 'mock provider is local and ready') return '演示通道可用：本地任务流转验证已就绪，不调用外部模型。'
  if (normalized.includes('api key is not configured')) return '真实通道未配置 API Key：待配置正式凭证后再启用测试。'
  if (normalized.includes('real network test is disabled')) return '真实通道已映射：当前演示不发起外部网络调用。'
  return message || '暂无测试结果'
}

export function errorMessageLabel(message?: string | null): string {
  if (!message) return ''
  const normalized = message.trim().toLowerCase()
  if (normalized === 'mock provider forced failure for p0 validation') return '生成失败：演示通道按提示词返回失败。可复制参数重新创建，或更换模型通道。'
  if (normalized === 'video task timed out') return '视频任务处理超时'
  if (normalized.startsWith('video provider submit failed:')) {
    return `提交到模型通道失败：${message.slice('video provider submit failed:'.length).trim()}`
  }
  if (normalized.startsWith('video provider poll failed:')) {
    return `查询模型通道状态失败：${message.slice('video provider poll failed:'.length).trim()}`
  }
  return message
}

export function statusBadgeClass(status: string): string {
  switch (status) {
    case 'queued':
      return 'bg-slate-100 text-slate-700 dark:bg-slate-500/10 dark:text-slate-300'
    case 'submitted':
      return 'bg-sky-50 text-sky-700 dark:bg-sky-500/10 dark:text-sky-300'
    case 'running':
      return 'bg-blue-50 text-blue-700 dark:bg-blue-500/10 dark:text-blue-300'
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

export const VIDEO_TASK_DRAFT_KEY = 'video_gateway_task_draft'

export function saveTaskDraft(task: VideoTask): void {
  const draft: VideoTaskCreatePayload = {
    provider_account_id: task.provider_account_id,
    task_type: task.task_type,
    model: task.model,
    prompt: task.prompt,
    negative_prompt: task.negative_prompt,
    reference_image_url: task.reference_image_url,
    reference_video_url: task.reference_video_url,
    aspect_ratio: task.aspect_ratio,
    duration: task.duration,
    resolution: task.resolution,
  }
  sessionStorage.setItem(VIDEO_TASK_DRAFT_KEY, JSON.stringify(draft))
}

export function loadTaskDraft(): VideoTaskCreatePayload | null {
  const raw = sessionStorage.getItem(VIDEO_TASK_DRAFT_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as VideoTaskCreatePayload
  } catch {
    return null
  } finally {
    sessionStorage.removeItem(VIDEO_TASK_DRAFT_KEY)
  }
}
