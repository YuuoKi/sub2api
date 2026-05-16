import { apiClient } from '../client'

export type VideoProvider = 'mock' | 'seedance' | 'kling'
export type VideoTaskType = 'text_to_video' | 'image_to_video' | 'reference_to_video'
export type VideoTaskStatus = 'queued' | 'submitted' | 'running' | 'succeeded' | 'failed' | 'cancelled'

export interface VideoProviderAccount {
  id: number
  provider: VideoProvider
  display_name: string
  enabled: boolean
  api_key_configured: boolean
  masked_key: string
  base_url: string
  default_model: string
  rate_limit_per_minute: number
  metadata_json: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface VideoProviderPayload {
  provider?: VideoProvider
  display_name?: string
  enabled?: boolean
  api_key?: string
  base_url?: string
  default_model?: string
  rate_limit_per_minute?: number
  metadata_json?: Record<string, unknown>
}

export interface VideoProviderTestResult {
  provider: VideoProvider
  configured: boolean
  reachable: boolean
  message: string
  normalized_status: string
  payload_preview?: Record<string, unknown>
}

export interface VideoTaskSummary {
  id: number
  provider: VideoProvider
  model: string
  task_type: VideoTaskType
  prompt: string
  status: VideoTaskStatus
  result_url: string
  error_message: string
  cost_estimate: number
  created_at: string
  updated_at: string
  completed_at: string | null
}

export interface VideoProviderStatus {
  provider: VideoProvider
  display_name: string
  enabled: boolean
  api_key_configured: boolean
  masked_key: string
  default_model: string
  updated_at: string
  today_tasks: number
  running_tasks: number
  failed_tasks: number
}

export interface VideoUsageSummary {
  provider: VideoProvider
  model: string
  status: VideoTaskStatus
  count: number
  cost_estimate: number
  duration: number
}

export interface VideoDashboard {
  today_tasks: number
  success_rate: number
  failed_tasks: number
  queued_tasks: number
  running_tasks: number
  provider_status: VideoProviderStatus[]
  recent_failures: VideoTaskSummary[]
  recent_successes: VideoTaskSummary[]
  usage_overview: VideoUsageSummary[]
}

export interface VideoTaskEvent {
  id: number
  video_task_id: number
  event_type: string
  message: string
  payload_json: Record<string, unknown>
  created_at: string
}

export interface VideoTask extends VideoTaskSummary {
  provider_account_id: number
  negative_prompt: string
  reference_image_url: string
  reference_video_url: string
  aspect_ratio: string
  duration: number
  resolution: string
  upstream_task_id: string
  created_by: number
  events?: VideoTaskEvent[]
}

export interface VideoTaskCreatePayload {
  provider_account_id: number
  task_type: VideoTaskType
  model?: string
  prompt: string
  negative_prompt?: string
  reference_image_url?: string
  reference_video_url?: string
  aspect_ratio?: string
  duration?: number
  resolution?: string
}

export interface VideoTaskListParams {
  page?: number
  page_size?: number
  status?: VideoTaskStatus | ''
  provider?: VideoProvider | ''
}

export interface VideoTaskListResponse {
  items: VideoTask[]
  total: number
  page: number
  page_size: number
  pages: number
}

async function listProviders(): Promise<{ items: VideoProviderAccount[] }> {
  const { data } = await apiClient.get<{ items: VideoProviderAccount[] }>('/admin/video/providers')
  return data
}

async function createProvider(payload: VideoProviderPayload): Promise<VideoProviderAccount> {
  const { data } = await apiClient.post<VideoProviderAccount>('/admin/video/providers', payload)
  return data
}

async function updateProvider(id: number, payload: VideoProviderPayload): Promise<VideoProviderAccount> {
  const { data } = await apiClient.patch<VideoProviderAccount>(`/admin/video/providers/${id}`, payload)
  return data
}

async function testProvider(id: number): Promise<VideoProviderTestResult> {
  const { data } = await apiClient.post<VideoProviderTestResult>(`/admin/video/providers/${id}/test`)
  return data
}

async function dashboard(): Promise<VideoDashboard> {
  const { data } = await apiClient.get<VideoDashboard>('/admin/video/dashboard')
  return data
}

async function listTasks(params: VideoTaskListParams = {}): Promise<VideoTaskListResponse> {
  const { data } = await apiClient.get<VideoTaskListResponse>('/video/tasks', { params })
  return data
}

async function createTask(payload: VideoTaskCreatePayload): Promise<VideoTask> {
  const { data } = await apiClient.post<VideoTask>('/video/tasks', payload)
  return data
}

async function getTask(id: number): Promise<VideoTask> {
  const { data } = await apiClient.get<VideoTask>(`/video/tasks/${id}`)
  return data
}

async function cancelTask(id: number): Promise<VideoTask> {
  const { data } = await apiClient.post<VideoTask>(`/video/tasks/${id}/cancel`)
  return data
}

export const videoAdminAPI = {
  listProviders,
  createProvider,
  updateProvider,
  testProvider,
  dashboard,
}

export const videoTaskAPI = {
  list: listTasks,
  create: createTask,
  get: getTask,
  cancel: cancelTask,
}

export default videoAdminAPI
