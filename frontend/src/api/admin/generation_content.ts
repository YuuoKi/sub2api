/**
 * Admin Generation-Content API endpoints (read-only 看板：护城河快照 + 样本墙)
 * Reads the ai_generation_content capture side-table. Admin-only (backend adminAuth).
 */

import { apiClient } from '../client'

export interface GenerationContentDailyPoint {
  date: string // "YYYY-MM-DD" (UTC)
  count: number
}

export interface GenerationContentStats {
  captured_today: number
  captured_week: number
  distinct_employees: number
  distinct_teams: number
  distinct_models: number
  total_bytes: number
  daily_rate: number
  daily_series: GenerationContentDailyPoint[]
  is_live: boolean
}

export interface GenerationSample {
  task_id: number | null
  employee_name: string
  team_name: string
  model: string
  video_status: string
  cost_estimate: number
  currency?: string
  created_at: string
  prompt_preview: string
  response_preview: string
  total_bytes: number
  adoption_status: AdoptionStatus | ''
  quality_score: number | null
  adoption_notes: string
  truncated: boolean
}

export interface GenerationContentSamplesResponse {
  samples: GenerationSample[]
  is_live: boolean
}

export type AdoptionStatus = 'adopted' | 'rejected' | 'pending'

export interface UpdateAdoptionPayload {
  adoption_status: AdoptionStatus
  quality_score?: number | null
  notes?: string
}

export interface UpdateAdoptionResponse {
  enabled: boolean
  saved: boolean
  reason?: string
  task_id: number
  adoption_status: AdoptionStatus
  quality_score?: number | null
  notes?: string
}

export interface WeeklyReportAnomalies {
  failed_tasks: number
  missing_task_joins: number
  truncated_rows: number
}

export interface GenerationContentWeeklyReport {
  period_start: string
  period_end: string
  entries: number
  video_tasks: number
  total_cost_estimate: number
  adopted_count: number
  rejected_count: number
  pending_count: number
  unreviewed_count: number
  adoption_rate: number
  anomalies: WeeklyReportAnomalies
  markdown: string
}

/**
 * Get capture stats snapshot (counts / distinct / volume + 7-day series).
 */
export async function getStats(options?: { signal?: AbortSignal }): Promise<GenerationContentStats> {
  const { data } = await apiClient.get<GenerationContentStats>('/admin/generation-content/stats', {
    signal: options?.signal
  })
  return data
}

/**
 * Get recent redacted samples for the content wall.
 */
export async function getSamples(options?: {
  signal?: AbortSignal
}): Promise<GenerationContentSamplesResponse> {
  const { data } = await apiClient.get<GenerationContentSamplesResponse>(
    '/admin/generation-content/samples',
    { signal: options?.signal }
  )
  return data
}

export async function updateAdoption(
  taskId: number,
  payload: UpdateAdoptionPayload
): Promise<UpdateAdoptionResponse> {
  const { data } = await apiClient.post<UpdateAdoptionResponse>(
    `/admin/generation-content/${taskId}/adoption`,
    payload
  )
  return data
}

export async function getWeeklyReport(options?: {
  signal?: AbortSignal
}): Promise<GenerationContentWeeklyReport> {
  const { data } = await apiClient.get<GenerationContentWeeklyReport>(
    '/admin/generation-content/weekly-report',
    { signal: options?.signal }
  )
  return data
}

export const generationContentAPI = {
  getStats,
  getSamples,
  updateAdoption,
  getWeeklyReport
}

export default generationContentAPI
