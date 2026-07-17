/**
 * JWT employee video simulation client.
 * Calls only /user/video/simulation/* — never admin video APIs.
 */

import { apiClient } from '../client'

export interface VideoSimulationContract {
  provider: 'mock'
  model: string
  label: string
  media_kind: 'image' | 'video'
  duration_seconds: number
  resolution: string
  currency: string
  pricing_source: string
  pricing_version: string
  cost_amount: number
  network: boolean
  billing: boolean
}

export type VideoSimulationTaskStatus =
  | 'queued'
  | 'submitted'
  | 'running'
  | 'succeeded'
  | 'failed'
  | 'cancelled'

export interface VideoSimulationTask {
  id: number
  provider: string
  model: string
  status: string
  prompt: string
  duration: number
  resolution: string
  cost: number
  currency: string
  pricing_source: string
  pricing_version: string
  error: string
  version: number
  created_at: string
  updated_at: string
  completed_at: string | null
}

export interface CreateSimulationTaskRequest {
  api_key_id: number
  prompt: string
  creation_key?: string
}

export interface VideoSimulationTaskList {
  items: VideoSimulationTask[]
}

export type VideoSimulationRequestOptions = {
  signal?: AbortSignal
}

export async function getSimulationContract(
  options?: VideoSimulationRequestOptions,
): Promise<VideoSimulationContract> {
  const { data } = options?.signal
    ? await apiClient.get<VideoSimulationContract>('/user/video/simulation/contract', {
        signal: options.signal,
      })
    : await apiClient.get<VideoSimulationContract>('/user/video/simulation/contract')
  return data
}

export async function createSimulationTask(
  payload: CreateSimulationTaskRequest,
  options?: VideoSimulationRequestOptions,
): Promise<VideoSimulationTask> {
  const { data } = options?.signal
    ? await apiClient.post<VideoSimulationTask>('/user/video/simulation/tasks', payload, {
        signal: options.signal,
      })
    : await apiClient.post<VideoSimulationTask>('/user/video/simulation/tasks', payload)
  return data
}

export async function listSimulationTasks(
  options?: VideoSimulationRequestOptions,
): Promise<VideoSimulationTaskList> {
  const { data } = options?.signal
    ? await apiClient.get<VideoSimulationTaskList>('/user/video/simulation/tasks', {
        signal: options.signal,
      })
    : await apiClient.get<VideoSimulationTaskList>('/user/video/simulation/tasks')
  return data
}

export async function getSimulationTask(
  id: number,
  options?: VideoSimulationRequestOptions,
): Promise<VideoSimulationTask> {
  const { data } = options?.signal
    ? await apiClient.get<VideoSimulationTask>(`/user/video/simulation/tasks/${id}`, {
        signal: options.signal,
      })
    : await apiClient.get<VideoSimulationTask>(`/user/video/simulation/tasks/${id}`)
  return data
}

export async function cancelSimulationTask(
  id: number,
  options?: VideoSimulationRequestOptions,
): Promise<VideoSimulationTask> {
  const { data } = options?.signal
    ? await apiClient.post<VideoSimulationTask>(
        `/user/video/simulation/tasks/${id}/cancel`,
        null,
        { signal: options.signal },
      )
    : await apiClient.post<VideoSimulationTask>(`/user/video/simulation/tasks/${id}/cancel`)
  return data
}

export async function downloadSimulationResult(
  id: number,
  options?: VideoSimulationRequestOptions,
): Promise<Blob> {
  const { data } = await apiClient.get<Blob>(`/user/video/simulation/tasks/${id}/result`, {
    responseType: 'blob',
    ...(options?.signal ? { signal: options.signal } : {}),
  })
  return data
}

export const videoSimulationAPI = {
  getContract: getSimulationContract,
  createTask: createSimulationTask,
  listTasks: listSimulationTasks,
  getTask: getSimulationTask,
  cancelTask: cancelSimulationTask,
  downloadResult: downloadSimulationResult,
}

export default videoSimulationAPI
