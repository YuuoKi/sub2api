import type { VideoProvider, VideoProviderAccount, VideoTask, VideoTaskCreatePayload, VideoTaskStatus, VideoTaskType } from '@/api/admin/video'
import { isVideoGatewayDemoMode, videoGatewayDisplayProvider } from '@/utils/productMode'

export const providerOptions: Array<{ value: VideoProvider; label: string }> = [
  { value: 'mock', label: isVideoGatewayDemoMode ? '演示通道' : 'Mock Provider' },
  { value: 'seedance', label: 'Seedance 2.0' },
  { value: 'kling', label: 'Kling' },
]

export const statusOptions: Array<{ value: VideoTaskStatus; label: string }> = [
  { value: 'queued', label: isVideoGatewayDemoMode ? '网关排队中' : '排队中' },
  { value: 'submitted', label: isVideoGatewayDemoMode ? '已分发' : '已提交' },
  { value: 'running', label: isVideoGatewayDemoMode ? '上游处理中' : '处理中' },
  { value: 'succeeded', label: isVideoGatewayDemoMode ? '结果已回收' : '已成功' },
  { value: 'failed', label: isVideoGatewayDemoMode ? '调用失败' : '已失败' },
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
  if (isVideoGatewayDemoMode && provider === 'mock') return '演示视频 API'
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
    return '企业 AI 视频 API 网关演示调用'
  }
  return value.replace(/\[fail\]\s*/gi, '').trim() || '企业 AI 视频 API 网关演示调用'
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
      return isVideoGatewayDemoMode
        ? '演示通道用于验证 API 网关排队、分发、状态回收和失败记录，不调用外部模型。'
        : '演示通道用于本地闭环验证，不调用外部模型。'
    case 'seedance':
      return isVideoGatewayDemoMode
        ? 'Seedance 2.0 预留 API 通道，待配置正式或测试 API Key 后启用。'
        : 'Seedance 2.0 预留通道，待配置 API Key 后启用真实调用。'
    case 'kling':
      return isVideoGatewayDemoMode
        ? 'Kling 预留 API 通道，待配置正式或测试 API Key 后启用。'
        : 'Kling 预留通道，待配置 API Key 后启用真实调用。'
    default:
      return isVideoGatewayDemoMode ? '视频模型 API 通道' : '视频模型通道'
  }
}

export function providerKeyLabel(configured: boolean, maskedKey?: string, keyStatus?: string): string {
  if (keyStatus === 'normal') return isVideoGatewayDemoMode ? '正常可用' : '已配置 Key'
  if (keyStatus === 'missing') return '未配置 Key'
  if (keyStatus === 'disabled') return '停用'
  if (keyStatus === 'auth_failed') return '鉴权失败'
  if (keyStatus === 'rate_limited') return '触发限流'
  if (keyStatus === 'quota_exhausted') return '额度不足'
  if (!configured) return isVideoGatewayDemoMode ? '未配置 API Key' : '未配置 Key'
  if (isVideoGatewayDemoMode) return '已配置 / 已脱敏'
  return maskedKey ? `已脱敏：${maskedKey}` : '已配置 Key'
}

export function providerEnabledLabel(enabled: boolean): string {
  return enabled ? '启用' : '停用'
}

export function providerRuntimeStatus(provider: Pick<VideoProviderAccount, 'provider' | 'enabled' | 'api_key_configured'> & Partial<VideoProviderAccount>): string {
  if (!provider.enabled) return '停用'
  if (provider.route_available === true) return '正常可用'
  if (provider.route_skip_reason) return provider.route_skip_reason
  if (provider.diagnostic_type) return provider.diagnostic_type
  if (provider.key_status === 'missing') return '未配置 Key'
  if (provider.key_status === 'auth_failed') return '鉴权失败'
  if (provider.key_status === 'rate_limited') return '触发限流'
  if (provider.key_status === 'quota_exhausted') return '额度不足'
  if (provider.provider === 'mock') return isVideoGatewayDemoMode ? '演示可用' : '可演示'
  if (!provider.api_key_configured) return '待配置'
  return isVideoGatewayDemoMode ? '可调用' : '已配置'
}

export function providerRuntimeStatusClass(status: string): string {
  switch (status) {
    case '可演示':
    case '演示可用':
    case '已配置':
    case '可调用':
    case '正常可用':
      return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
    case '待配置':
    case '未配置 Key':
    case '触发限流':
      return 'bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
    case '停用':
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-200'
    default:
      return 'bg-red-50 text-red-700 dark:bg-red-500/10 dark:text-red-300'
  }
}

export function keyStatusClass(keyStatus?: string): string {
  switch (keyStatus) {
    case 'normal':
      return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
    case 'missing':
    case 'rate_limited':
      return 'bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
    case 'disabled':
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-200'
    default:
      return 'bg-red-50 text-red-700 dark:bg-red-500/10 dark:text-red-300'
  }
}

export function routeAccountLabel(task: Pick<VideoTask, 'provider_account_id' | 'provider_account_name'>): string {
  return task.provider_account_name?.trim() || `账号 #${task.provider_account_id}`
}

export function createdByLabel(task: Partial<VideoTask>): string {
  return task.created_by_label?.trim()
    || task.created_by_name?.trim()
    || task.created_by_email?.trim()
    || (task.created_by ? `用户 #${task.created_by}` : '-')
}

export function routingStrategyLabel(strategy?: string): string {
  if (strategy === 'least_inflight') return 'least_inflight（处理中最少）'
  if (strategy === 'explicit') return '指定账号'
  return strategy || '-'
}

export function eventTypeLabel(eventType: string): string {
  if (isVideoGatewayDemoMode) {
    switch (eventType) {
      case 'queued':
        return '任务进入网关队列'
      case 'routed':
        return '自动路由'
      case 'submitted':
        return '已分发至 API 通道'
      case 'running':
        return '上游处理中'
      case 'succeeded':
        return '状态回收完成'
      case 'failed':
        return '调用失败'
      case 'cancelled':
        return '调用已取消'
      case 'timeout':
        return '状态回收超时'
      default:
        return eventType
    }
  }
  switch (eventType) {
    case 'queued':
      return '任务已入队'
    case 'routed':
      return '已选择通道账号'
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
  if (isVideoGatewayDemoMode) {
    if (normalized === 'video task queued') return '任务进入网关队列'
    if (normalized === 'video task routed') return '已自动选择 API 账号'
    if (normalized === 'video task submitted to provider') return '已分发至 API 通道'
    if (normalized === 'video task status updated') return '上游状态已回收'
    if (normalized === 'video task succeeded') return '结果已入库'
    if (normalized === 'video task failed') return '调用失败'
    if (normalized === 'video task cancelled') return '调用已取消'
    if (normalized === 'video task timed out') return '状态回收超时'
  } else {
    if (normalized === 'video task queued') return '任务已进入视频队列'
    if (normalized === 'video task routed') return '已选择模型通道账号'
    if (normalized === 'video task submitted to provider') return '任务已提交到模型通道'
    if (normalized === 'video task status updated') return '任务状态已更新'
    if (normalized === 'video task succeeded') return '视频生成成功'
    if (normalized === 'video task failed') return '视频生成失败'
    if (normalized === 'video task cancelled') return '任务已取消'
    if (normalized === 'video task timed out') return '任务处理超时'
  }
  return message
}

export function providerTestMessage(message: string): string {
  const normalized = message.trim().toLowerCase()
  if (normalized === 'mock provider is local and ready') {
    return isVideoGatewayDemoMode
      ? '演示通道可用：API 网关流转验证已就绪，不调用外部模型。'
      : '演示通道可用：本地任务流转验证已就绪，不调用外部模型。'
  }
  if (normalized.includes('api key is not configured')) return '真实通道未配置 API Key：待配置正式或测试凭证后再启用。'
  if (normalized.includes('real network test is disabled')) return '真实通道已映射：当前演示不发起外部网络调用。'
  return message || '暂无测试结果'
}

export function errorMessageLabel(message?: string | null): string {
  if (!message) return ''
  const normalized = message.trim().toLowerCase()
  if (normalized === 'mock provider forced failure for p0 validation') {
    return isVideoGatewayDemoMode
      ? '调用失败：演示通道按提示词返回失败，失败原因已记录。可复制参数重新发起，或切换 API 通道。'
      : '生成失败：演示通道按提示词返回失败。可复制参数重新创建，或更换模型通道。'
  }
  if (normalized === 'video task timed out') return isVideoGatewayDemoMode ? 'API 调用状态回收超时' : '视频任务处理超时'
  if (normalized.startsWith('video provider submit failed:')) {
    return `${isVideoGatewayDemoMode ? '分发至 API 通道失败' : '提交到模型通道失败'}：${message.slice('video provider submit failed:'.length).trim()}`
  }
  if (normalized.startsWith('video provider poll failed:')) {
    return `${isVideoGatewayDemoMode ? '状态回收失败' : '查询模型通道状态失败'}：${message.slice('video provider poll failed:'.length).trim()}`
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
    provider_account_id: 0,
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
