import { apiClient } from '../client'

export interface VideoProviderAccount {
  id: number; group_id: number; group_name: string; provider: 'seedance'; display_name: string
  enabled: boolean; api_key_configured: boolean; masked_key: string; base_url: string; default_model: string
  tiny_real_authorized_at?: string; tiny_real_authorized_by?: number; tiny_real_consumed_at?: string
}
export interface VideoProviderPayload { group_id?: number; provider?: 'seedance'; display_name?: string; enabled?: boolean; api_key?: string; base_url?: string; default_model?: string }
export interface VideoProviderContract { provider: 'seedance'; base_url: string; default_model: string; duration_seconds: number; resolution: string }
export interface VideoTaskAdmin {
  id: number; api_key_id: number; group_id: number; provider_account_id: number; provider: string; model: string; task_type: string; prompt: string; status: string
  request_model: string; request_duration_seconds: number; request_resolution: string
  upstream_model: string | null; upstream_duration_seconds: number | null; upstream_resolution: string | null
  billing_model: string | null; billing_duration_seconds: number | null; billing_resolution: string | null
  upstream_task_id: string; result_url: string; last_frame_url: string; error_message: string; provider_error_code: string
  duration_seconds: number; resolution: string; usage_total_tokens?: number
  provider_error_message: string; reserved_cost_usd: number; reservation_state: string; reserved_at: string | null
  reservation_window_5h_start: string | null; reservation_window_1d_start: string | null; reservation_window_7d_start: string | null
  cost_amount: number; provider_actual_cost_usd: number; currency: string
  balance_before_usd: number | null; balance_after_usd: number | null; balance_delta_usd: number | null
  authorization_consumed_at: string | null; authorization_consumed_by: number | null
  real_dispatch_count: number; dispatch_state: string; created_by: number; created_at: string; updated_at: string; completed_at?: string
  // 定价来源与快照（Task 2C1 canonical pricing provenance）
  pricing_source?: string; pricing_version?: string
  pricing_cny_per_million_completion_tokens?: number | null
  pricing_usd_cny_exchange_rate?: number | null
  pricing_maximum_cny?: number | null
  // 本地资产落盘状态（Task 2C2 local asset persistence）
  local_asset_available?: boolean
  local_asset_download_url?: string | null
  local_asset_saved_at?: string | null
}
export interface VideoSystemCheck { provider_count: number; enabled_provider_count: number; authorized_provider_count: number; task_count: number; real_dispatch_count: number; global_tiny_real_consumed: boolean }
export interface VideoTaskPage { items: VideoTaskAdmin[]; total: number; page: number; page_size: number; pages: number }
export type AssetHandoffKind = 'image' | 'video'
export interface IssuedAssetHandoff {
  ticket: string
  source_task_id: number
  asset_kind: AssetHandoffKind
  expires_at: string
}

const listProviders = async () => (await apiClient.get<{ items: VideoProviderAccount[] }>('/admin/video/providers')).data
const contract = async () => (await apiClient.get<VideoProviderContract>('/admin/video/contract')).data
const createProvider = async (payload: VideoProviderPayload) => (await apiClient.post<VideoProviderAccount>('/admin/video/providers', payload)).data
const updateProvider = async (id: number, payload: VideoProviderPayload) => (await apiClient.put<VideoProviderAccount>(`/admin/video/providers/${id}`, payload)).data
const authorizeTinyReal = async (id: number) => (await apiClient.post<VideoProviderAccount>(`/admin/video/providers/${id}/tiny-real-authorization`, { confirmation: 'tiny_real' })).data
const listTasks = async (page = 1, pageSize = 20, status = '') => (await apiClient.get<VideoTaskPage>('/admin/video/tasks', { params: { page, page_size: pageSize, status: status || undefined } })).data
const getTask = async (id: number) => (await apiClient.get<VideoTaskAdmin>(`/admin/video/tasks/${id}`)).data
export const createAssetHandoff = async (id: number, assetKind: AssetHandoffKind) => (
  await apiClient.post<IssuedAssetHandoff>(`/admin/video/tasks/${id}/asset-handoffs`, { asset_kind: assetKind })
).data

const requireLoopbackOrigin = (rawURL: string, label: string): URL => {
  const url = new URL(rawURL)
  const host = url.hostname.toLowerCase()
  const isLoopback = host === 'localhost' || host === '::1' || /^127(?:\.\d{1,3}){3}$/.test(host)
  if (!isLoopback) throw new Error(`${label}必须是 loopback origin`)
  if (url.protocol !== 'http:' && url.protocol !== 'https:') throw new Error(`${label}必须使用 HTTP 或 HTTPS`)
  if (url.username || url.password) throw new Error(`${label}不得包含用户名或密码`)
  if (url.pathname !== '/' || url.search || url.hash) throw new Error(`${label}必须是纯 origin，不得包含路径、查询或片段`)
  return url
}

export const buildQCanvasAssetHandoffTargetURL = (qcanvasBaseURL: string): string => {
  if (!qcanvasBaseURL.trim()) throw new Error('QCanvas 地址未配置')
  return new URL('/asset-handoff', requireLoopbackOrigin(qcanvasBaseURL, 'QCanvas 地址')).toString()
}

export const buildQCanvasAssetHandoffWindowName = (sourceOrigin: string, nonce: string): string => {
  const origin = requireLoopbackOrigin(sourceOrigin, '上游来源').origin
  if (!/^[A-Za-z0-9_-]{16,128}$/.test(nonce)) throw new Error('资产交接 nonce 无效')
  return `qcanvas-asset-handoff:${encodeURIComponent(origin)}:${nonce}`
}

export const startQCanvasAssetHandoffTransfer = (
  targetWindow: Window,
  ticket: string,
  qcanvasBaseURL: string,
  sourceWindow: Window = window
): (() => void) => {
  if (ticket.trim().length < 20) throw new Error('交接票据格式无效')
  const targetURL = buildQCanvasAssetHandoffTargetURL(qcanvasBaseURL)
  const targetOrigin = new URL(targetURL).origin
  const nonce = sourceWindow.crypto.randomUUID()
  targetWindow.name = buildQCanvasAssetHandoffWindowName(sourceWindow.location.origin, nonce)
  let timeoutId = 0
  const cleanup = (): void => {
    sourceWindow.removeEventListener('message', handleMessage)
    if (timeoutId) sourceWindow.clearTimeout(timeoutId)
  }
  const handleMessage = (event: MessageEvent<unknown>): void => {
    if (event.source !== targetWindow || event.origin !== targetOrigin) return
    const payload = event.data && typeof event.data === 'object' ? event.data as Record<string, unknown> : {}
    if (payload.type !== 'qcanvas-asset-handoff-ready' || payload.nonce !== nonce) return
    targetWindow.postMessage({ type: 'sub2api-asset-handoff-ticket', nonce, ticket: ticket.trim() }, targetOrigin)
    cleanup()
  }
  sourceWindow.addEventListener('message', handleMessage)
  timeoutId = sourceWindow.setTimeout(cleanup, 270_000)
  targetWindow.location.replace(targetURL)
  return cleanup
}
const systemCheck = async () => (await apiClient.get<VideoSystemCheck>('/admin/video/system-check')).data
export default { contract, listProviders, createProvider, updateProvider, authorizeTinyReal, listTasks, getTask, createAssetHandoff, systemCheck }
