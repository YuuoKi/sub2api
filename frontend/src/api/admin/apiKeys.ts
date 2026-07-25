/**
 * Admin API Keys API endpoints
 * Handles API key management for administrators
 */

import { apiClient } from '../client'
import type { ApiKey } from '@/types'
import { createIdempotencyKey } from '@/utils/idempotencyKey'

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
  custom_key?: string
  ip_whitelist?: string[]
  ip_blacklist?: string[]
  quota?: number // USD, 0 = unlimited
  expires_in_days?: number
  rate_limit_5h?: number
  rate_limit_1d?: number
  rate_limit_7d?: number
}

export interface AdminCreateQCanvasKeyPairPayload {
  video_group_id: number
  media_group_id: number
}

export interface QCanvasKeyPairResponse {
  video: ApiKey
  media: ApiKey
}

/**
 * Issue a new API key bound to the specified member (member 开卡).
 * Requires a per-call Idempotency-Key so retried submissions never mint a second card;
 * a replayed request returns the same key metadata with an empty `key` (see R5 key-once semantics).
 * @param userId - Owner user ID
 * @param payload - Key parameters
 * @param idempotencyKey - Unique key for this creation attempt (defaults to a fresh UUID)
 * @returns Created API key (the full key value is only ever returned here, once)
 */
export async function createApiKeyForUser(
  userId: number,
  payload: AdminCreateApiKeyPayload,
  idempotencyKey: string = createIdempotencyKey()
): Promise<ApiKey> {
  const { data } = await apiClient.post<ApiKey>(`/admin/users/${userId}/api-keys`, payload, {
    headers: { 'Idempotency-Key': idempotencyKey }
  })
  return data
}

/**
 * Atomically issue the two logical QCanvas credentials for one existing user.
 * A replay uses the same Idempotency-Key and deliberately returns blank key
 * values, so the two full secrets are only visible in the first response.
 */
export async function createQCanvasKeyPairForUser(
  userId: number,
  payload: AdminCreateQCanvasKeyPairPayload,
  idempotencyKey: string = createIdempotencyKey()
): Promise<QCanvasKeyPairResponse> {
  const { data } = await apiClient.post<QCanvasKeyPairResponse>(`/admin/users/${userId}/qcanvas-key-pair`, payload, {
    headers: { 'Idempotency-Key': idempotencyKey }
  })
  return data
}

export interface AdminUpdateApiKeyFieldsPayload {
  name?: string
  // 后端 admin 契约只接受 active/disabled（见 apikey_handler.go oneof 校验）
  status?: 'active' | 'disabled'
  quota?: number // 0 = unlimited
  reset_quota?: boolean
  expires_at?: string // ISO 8601 (RFC3339); empty string clears expiration
  rate_limit_5h?: number
  rate_limit_1d?: number
  rate_limit_7d?: number
}

/**
 * Update an API key's admin-managed fields (name/status/quota/expiry/rate limits).
 * Backend requires exactly one mutation category per PUT call: fields | group_id | reset_rate_limit_usage.
 * Use `updateApiKeyGroup` for the group category and `resetApiKeyRateLimitUsage` for the reset category.
 * @param id - API Key ID
 * @param payload - Fields to update
 * @returns Updated API key with auto-grant info
 */
export async function updateApiKeyFields(
  id: number,
  payload: AdminUpdateApiKeyFieldsPayload
): Promise<UpdateApiKeyGroupResult> {
  const { data } = await apiClient.put<UpdateApiKeyGroupResult>(`/admin/api-keys/${id}`, payload)
  return data
}

/**
 * Reset an API key's rate-limit usage windows (5h/1d/7d). Its own mutation category.
 * @param id - API Key ID
 */
export async function resetApiKeyRateLimitUsage(id: number): Promise<UpdateApiKeyGroupResult> {
  const { data } = await apiClient.put<UpdateApiKeyGroupResult>(`/admin/api-keys/${id}`, {
    reset_rate_limit_usage: true
  })
  return data
}

/**
 * Delete an API key regardless of owner.
 * @param id - API Key ID
 */
export async function deleteApiKey(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/api-keys/${id}`)
  return data
}

export const apiKeysAPI = {
  updateApiKeyGroup,
  createApiKeyForUser,
	createQCanvasKeyPairForUser,
  updateApiKeyFields,
  resetApiKeyRateLimitUsage,
  deleteApiKey
}

export default apiKeysAPI
