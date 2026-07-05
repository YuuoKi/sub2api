/**
 * Admin API Keys API endpoints
 * Handles API key management for administrators
 */

import { apiClient } from '../client'
import type { ApiKey } from '@/types'

export interface UpdateApiKeyGroupResult {
  api_key: ApiKey
  auto_granted_group_access: boolean
  granted_group_id?: number
  granted_group_name?: string
}

/**
 * Update an API key's group binding
 * @param id - API Key ID
 * @param groupId - Group ID (0 to unbind, positive to bind, null/undefined to skip)
 * @returns Updated API key with auto-grant info
 */
export async function updateApiKeyGroup(id: number, groupId: number | null): Promise<UpdateApiKeyGroupResult> {
  const { data } = await apiClient.put<UpdateApiKeyGroupResult>(`/admin/api-keys/${id}`, {
    group_id: groupId === null ? 0 : groupId
  })
  return data
}

export interface AdminCreateApiKeyPayload {
  name: string
  group_id?: number | null
  quota?: number // USD, 0 = unlimited
  expires_in_days?: number
  rate_limit_5h?: number
  rate_limit_1d?: number
  rate_limit_7d?: number
}

/**
 * Issue a new API key bound to the specified user (员工开卡)
 * @param userId - Owner user ID
 * @param payload - Key parameters
 * @returns Created API key (full key value is only returned here)
 */
export async function createApiKeyForUser(userId: number, payload: AdminCreateApiKeyPayload): Promise<ApiKey> {
  const { data } = await apiClient.post<ApiKey>(`/admin/users/${userId}/api-keys`, payload)
  return data
}

export interface AdminUpdateApiKeyPayload {
  name?: string
  // 后端 admin 契约只接受 active/disabled（见 apikey_handler.go oneof 校验）
  status?: 'active' | 'disabled'
  quota?: number // 0 = unlimited
  reset_quota?: boolean
  expires_at?: string // ISO 8601; empty string clears expiration
  rate_limit_5h?: number
  rate_limit_1d?: number
  rate_limit_7d?: number
  reset_rate_limit_usage?: boolean
  group_id?: number | null
}

/**
 * Update an API key's admin-managed fields (name/status/quota/expiry/rate limits/group)
 * @param id - API Key ID
 * @param payload - Fields to update
 * @returns Updated API key with auto-grant info
 */
export async function updateApiKey(id: number, payload: AdminUpdateApiKeyPayload): Promise<UpdateApiKeyGroupResult> {
  const { data } = await apiClient.put<UpdateApiKeyGroupResult>(`/admin/api-keys/${id}`, payload)
  return data
}

/**
 * Delete an API key regardless of owner
 * @param id - API Key ID
 */
export async function deleteApiKey(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/api-keys/${id}`)
  return data
}

export const apiKeysAPI = {
  updateApiKeyGroup,
  createApiKeyForUser,
  updateApiKey,
  deleteApiKey
}

export default apiKeysAPI
