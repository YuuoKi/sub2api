import { apiClient } from '../client'

export interface VideoProviderAccount {
  id: number; group_id: number; group_name: string; provider: 'seedance'; display_name: string
  enabled: boolean; api_key_configured: boolean; masked_key: string; base_url: string; default_model: string
  tiny_real_authorized_at?: string; tiny_real_authorized_by?: number; tiny_real_consumed_at?: string
}
export interface VideoProviderPayload { group_id?: number; provider?: 'seedance'; display_name?: string; enabled?: boolean; api_key?: string; base_url?: string; default_model?: string }
export interface VideoProviderContract { provider: 'seedance'; base_url: string; default_model: string; duration_seconds: number; resolution: string }
export interface VideoTaskAdmin {
  id: number; provider_account_id: number; provider: string; model: string; task_type: string; prompt: string; status: string
  upstream_task_id: string; result_url: string; last_frame_url: string; error_message: string; provider_error_code: string
  provider_error_message: string; cost_amount: number; provider_actual_cost_usd: number; currency: string
  real_dispatch_count: number; dispatch_state: string; created_by: number; created_at: string; updated_at: string; completed_at?: string
}
export interface VideoSystemCheck { provider_count: number; enabled_provider_count: number; authorized_provider_count: number; task_count: number; real_dispatch_count: number; global_tiny_real_consumed: boolean }
export interface VideoTaskPage { items: VideoTaskAdmin[]; total: number; page: number; page_size: number; pages: number }

const listProviders = async () => (await apiClient.get<{ items: VideoProviderAccount[] }>('/admin/video/providers')).data
const contract = async () => (await apiClient.get<VideoProviderContract>('/admin/video/contract')).data
const createProvider = async (payload: VideoProviderPayload) => (await apiClient.post<VideoProviderAccount>('/admin/video/providers', payload)).data
const updateProvider = async (id: number, payload: VideoProviderPayload) => (await apiClient.put<VideoProviderAccount>(`/admin/video/providers/${id}`, payload)).data
const authorizeTinyReal = async (id: number) => (await apiClient.post<VideoProviderAccount>(`/admin/video/providers/${id}/tiny-real-authorization`, { confirmation: 'tiny_real' })).data
const listTasks = async (page = 1, pageSize = 20, status = '') => (await apiClient.get<VideoTaskPage>('/admin/video/tasks', { params: { page, page_size: pageSize, status: status || undefined } })).data
const getTask = async (id: number) => (await apiClient.get<VideoTaskAdmin>(`/admin/video/tasks/${id}`)).data
const systemCheck = async () => (await apiClient.get<VideoSystemCheck>('/admin/video/system-check')).data
export default { contract, listProviders, createProvider, updateProvider, authorizeTinyReal, listTasks, getTask, systemCheck }
