import { beforeEach, describe, expect, it, vi } from 'vitest'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const { get, post } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: { get, post },
}))

import {
  cancelSimulationTask,
  createSimulationTask,
  getSimulationContract,
  getSimulationTask,
  listSimulationTasks,
  downloadSimulationResult,
} from '@/api/user/video_simulation'

const clientSource = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), '../user/video_simulation.ts'),
  'utf8',
)

describe('user video simulation API client', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
  })

  it('calls only /user/video/simulation/* paths', () => {
    expect(clientSource).toContain("'/user/video/simulation/contract'")
    expect(clientSource).toContain("'/user/video/simulation/tasks'")
    expect(clientSource).toContain('`/user/video/simulation/tasks/${id}`')
    expect(clientSource).toContain('`/user/video/simulation/tasks/${id}/cancel`')
    expect(clientSource).toContain('`/user/video/simulation/tasks/${id}/result`')
    expect(clientSource).not.toContain('/admin/video')
  })

  it('loads contract, creates with 202 payload shape, lists, gets, cancels, and downloads blob', async () => {
    get.mockResolvedValueOnce({
      data: {
        provider: 'mock',
        model: 'mock-video-v1',
        label: '模拟视频结果',
        media_kind: 'image',
        duration_seconds: 4,
        resolution: '720p',
        currency: 'USD',
        pricing_source: 'internal_simulation',
        pricing_version: 'simulation-v1',
        cost_amount: 0,
        network: false,
        billing: false,
      },
    })
    post.mockResolvedValueOnce({
      data: {
        id: 12,
        provider: 'mock',
        model: 'mock-video-v1',
        status: 'queued',
        prompt: 'hello',
        duration: 4,
        resolution: '720p',
        cost: 0,
        currency: 'USD',
        pricing_source: 'internal_simulation',
        pricing_version: 'simulation-v1',
        error: '',
        version: 1,
        created_at: '2026-07-18T00:00:00Z',
        updated_at: '2026-07-18T00:00:00Z',
        completed_at: null,
      },
    })
    get.mockResolvedValueOnce({ data: { items: [{ id: 12, status: 'running' }] } })
    get.mockResolvedValueOnce({ data: { id: 12, status: 'succeeded' } })
    post.mockResolvedValueOnce({ data: { id: 12, status: 'cancelled' } })
    get.mockResolvedValueOnce({ data: new Blob(['<svg/>'], { type: 'image/svg+xml' }) })

    await expect(getSimulationContract()).resolves.toMatchObject({
      provider: 'mock',
      cost_amount: 0,
      media_kind: 'image',
      billing: false,
      network: false,
    })
    await expect(
      createSimulationTask({ api_key_id: 3, prompt: 'hello', creation_key: 'ck-1' }),
    ).resolves.toMatchObject({ id: 12, status: 'queued' })
    await expect(listSimulationTasks()).resolves.toEqual({ items: [{ id: 12, status: 'running' }] })
    await expect(getSimulationTask(12)).resolves.toMatchObject({ id: 12, status: 'succeeded' })
    await expect(cancelSimulationTask(12)).resolves.toMatchObject({ id: 12, status: 'cancelled' })
    await expect(downloadSimulationResult(12)).resolves.toBeInstanceOf(Blob)

    expect(get).toHaveBeenNthCalledWith(1, '/user/video/simulation/contract')
    expect(post).toHaveBeenNthCalledWith(1, '/user/video/simulation/tasks', {
      api_key_id: 3,
      prompt: 'hello',
      creation_key: 'ck-1',
    })
    expect(get).toHaveBeenNthCalledWith(2, '/user/video/simulation/tasks')
    expect(get).toHaveBeenNthCalledWith(3, '/user/video/simulation/tasks/12')
    expect(post).toHaveBeenNthCalledWith(2, '/user/video/simulation/tasks/12/cancel')
    expect(get).toHaveBeenNthCalledWith(4, '/user/video/simulation/tasks/12/result', {
      responseType: 'blob',
    })
  })

  it('forwards optional AbortSignal on list/get/cancel/result', async () => {
    const signal = new AbortController().signal
    get.mockResolvedValue({ data: { items: [] } })
    post.mockResolvedValue({ data: { id: 1, status: 'cancelled' } })

    await listSimulationTasks({ signal })
    await getSimulationTask(7, { signal })
    await cancelSimulationTask(7, { signal })
    get.mockResolvedValue({ data: new Blob(['x']) })
    await downloadSimulationResult(7, { signal })

    expect(get).toHaveBeenCalledWith('/user/video/simulation/tasks', { signal })
    expect(get).toHaveBeenCalledWith('/user/video/simulation/tasks/7', { signal })
    expect(post).toHaveBeenCalledWith(
      '/user/video/simulation/tasks/7/cancel',
      null,
      { signal },
    )
    expect(get).toHaveBeenCalledWith('/user/video/simulation/tasks/7/result', {
      responseType: 'blob',
      signal,
    })
  })
})
