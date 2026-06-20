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
  employee_name: string
  team_name: string
  model: string
  created_at: string
  prompt_preview: string
  response_preview: string
  total_bytes: number
  truncated: boolean
}

export interface GenerationContentSamplesResponse {
  samples: GenerationSample[]
  is_live: boolean
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

export const generationContentAPI = {
  getStats,
  getSamples
}

export default generationContentAPI
