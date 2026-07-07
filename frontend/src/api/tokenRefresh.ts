/**
 * Unified access-token refresh with singleflight deduplication.
 * Used by the Axios 401 interceptor and the auth store proactive timer.
 */

import axios from 'axios'
import type { ApiResponse } from '@/types'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1'
const AUTH_TOKEN_KEY = 'auth_token'
const REFRESH_TOKEN_KEY = 'refresh_token'
const TOKEN_EXPIRES_AT_KEY = 'token_expires_at'

export interface TokenRefreshResult {
  access_token: string
  refresh_token: string
  expires_in: number
  expires_at: number
}

let inFlightRefresh: Promise<TokenRefreshResult> | null = null

function persistRefreshedTokens(result: TokenRefreshResult): void {
  localStorage.setItem(AUTH_TOKEN_KEY, result.access_token)
  localStorage.setItem(REFRESH_TOKEN_KEY, result.refresh_token)
  localStorage.setItem(TOKEN_EXPIRES_AT_KEY, String(result.expires_at))
}

async function syncAuthStoreAfterRefresh(result: TokenRefreshResult): Promise<void> {
  try {
    const { useAuthStore } = await import('@/stores/auth')
    useAuthStore().syncTokenRefreshResult(result)
  } catch {
    // Pinia may not be active (isolated unit tests).
  }
}

async function executeRefresh(): Promise<TokenRefreshResult> {
  const refreshToken = localStorage.getItem(REFRESH_TOKEN_KEY)
  if (!refreshToken) {
    throw new Error('No refresh token available')
  }

  const response = await axios.post(
    `${API_BASE_URL}/auth/refresh`,
    { refresh_token: refreshToken },
    { headers: { 'Content-Type': 'application/json' } }
  )

  const refreshData = response.data as ApiResponse<{
    access_token: string
    refresh_token: string
    expires_in: number
  }>

  if (refreshData.code !== 0 || !refreshData.data) {
    throw new Error('Token refresh failed')
  }

  const { access_token, refresh_token, expires_in } = refreshData.data
  const result: TokenRefreshResult = {
    access_token,
    refresh_token,
    expires_in,
    expires_at: Date.now() + expires_in * 1000,
  }

  persistRefreshedTokens(result)
  await syncAuthStoreAfterRefresh(result)
  return result
}

/**
 * Refresh the access token at most once per in-flight window.
 * Concurrent callers share the same promise.
 */
export function refreshAccessTokenOnce(): Promise<TokenRefreshResult> {
  if (!inFlightRefresh) {
    inFlightRefresh = executeRefresh().finally(() => {
      inFlightRefresh = null
    })
  }
  return inFlightRefresh
}

/** Test-only reset for module-level singleflight state. */
export function resetTokenRefreshStateForTests(): void {
  inFlightRefresh = null
}
