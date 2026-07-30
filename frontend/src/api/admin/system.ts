/**
 * System API endpoints for admin operations.
 * Self-update / online rollback endpoints are removed (immutable build identity).
 */

import { apiClient } from '../client'

export interface VersionInfo {
  version: string
  build_commit: string
  build_date: string
  /** @deprecated kept optional for store cache shape; always false after hard cutover */
  current_version?: string
  latest_version?: string
  has_update?: boolean
  build_type?: string
  cached?: boolean
}

/**
 * Get current immutable deploy identity
 */
export async function getVersion(): Promise<VersionInfo> {
  const { data } = await apiClient.get<VersionInfo>('/admin/system/version')
  return data
}

/**
 * Restart the service (ops-only; not part of VersionBadge self-update UI)
 */
export async function restartService(): Promise<{ message: string }> {
  const { data } = await apiClient.post<{ message: string }>('/admin/system/restart')
  return data
}

export const systemAPI = {
  getVersion,
  restartService
}

export default systemAPI
