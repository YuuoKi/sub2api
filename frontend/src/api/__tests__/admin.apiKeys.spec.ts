import { beforeEach, describe, expect, it, vi } from 'vitest'

const { post, put, del } = vi.hoisted(() => ({
  post: vi.fn(),
  put: vi.fn(),
  del: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: { post, put, delete: del },
}))

import {
  apiKeysAPI,
  createApiKeyForUser,
  deleteApiKey,
  resetApiKeyRateLimitUsage,
  updateApiKeyFields,
  updateApiKeyGroup,
} from '@/api/admin/apiKeys'

describe('admin api-keys api adapter', () => {
  beforeEach(() => {
    post.mockReset()
    put.mockReset()
    del.mockReset()
  })

  it('issues a card for a member via POST /admin/users/:id/api-keys with an Idempotency-Key header', async () => {
    const created = { id: 1, key: 'sk-full-value-once', name: 'card' }
    post.mockResolvedValue({ data: created })

    const result = await createApiKeyForUser(9, { name: 'card', quota: 0 }, 'idem-key-123')

    expect(post).toHaveBeenCalledWith(
      '/admin/users/9/api-keys',
      { name: 'card', quota: 0 },
      { headers: { 'Idempotency-Key': 'idem-key-123' } },
    )
    expect(result).toEqual(created)
  })

  it('generates a fresh Idempotency-Key per call when one is not supplied', async () => {
    post.mockResolvedValue({ data: { id: 1, key: 'sk-value' } })

    await createApiKeyForUser(9, { name: 'card' })
    await createApiKeyForUser(9, { name: 'card' })

    const firstKey = post.mock.calls[0][2].headers['Idempotency-Key']
    const secondKey = post.mock.calls[1][2].headers['Idempotency-Key']
    expect(firstKey).toBeTruthy()
    expect(secondKey).toBeTruthy()
    expect(firstKey).not.toBe(secondKey)
  })

  it('updates field-category mutations only (name/status/quota/etc, no group_id)', async () => {
    put.mockResolvedValue({ data: { api_key: { id: 1 }, auto_granted_group_access: false } })

    await updateApiKeyFields(1, { status: 'disabled' })

    expect(put).toHaveBeenCalledWith('/admin/api-keys/1', { status: 'disabled' })
  })

  it('updates the group-category mutation via the existing updateApiKeyGroup helper', async () => {
    put.mockResolvedValue({ data: { api_key: { id: 1 }, auto_granted_group_access: false } })

    await updateApiKeyGroup(1, 5)

    expect(put).toHaveBeenCalledWith('/admin/api-keys/1', { group_id: 5 })
  })

  it('resets rate-limit usage as its own mutation category', async () => {
    put.mockResolvedValue({ data: { api_key: { id: 1 }, auto_granted_group_access: false } })

    await resetApiKeyRateLimitUsage(1)

    expect(put).toHaveBeenCalledWith('/admin/api-keys/1', { reset_rate_limit_usage: true })
  })

  it('deletes an API key regardless of owner via DELETE /admin/api-keys/:id', async () => {
    del.mockResolvedValue({ data: { message: 'API key deleted successfully' } })

    const result = await deleteApiKey(7)

    expect(del).toHaveBeenCalledWith('/admin/api-keys/7')
    expect(result).toEqual({ message: 'API key deleted successfully' })
  })

  it('exposes a barrel object with the new functions', () => {
    expect(apiKeysAPI.createApiKeyForUser).toBe(createApiKeyForUser)
    expect(apiKeysAPI.updateApiKeyFields).toBe(updateApiKeyFields)
    expect(apiKeysAPI.resetApiKeyRateLimitUsage).toBe(resetApiKeyRateLimitUsage)
    expect(apiKeysAPI.deleteApiKey).toBe(deleteApiKey)
    expect(apiKeysAPI.updateApiKeyGroup).toBe(updateApiKeyGroup)
  })
})
