import { beforeEach, describe, expect, it, vi } from 'vitest'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))

vi.mock('@/api/client', () => ({ apiClient: { post } }))

import {
  buildQCanvasAssetHandoffTargetURL,
  buildQCanvasAssetHandoffWindowName,
  createAssetHandoff,
  startQCanvasAssetHandoffTransfer
} from '@/api/admin/video'

describe('admin video asset handoff', () => {
  beforeEach(() => post.mockReset())

  it('creates a short-lived ticket without putting the source asset URL in the request', async () => {
    const response = {
      ticket: 'opaque-ticket',
      source_task_id: 501,
      asset_kind: 'video',
      expires_at: '2026-07-16T12:05:00Z'
    }
    post.mockResolvedValue({ data: response })

    const result = await createAssetHandoff(501, 'video')

    expect(post).toHaveBeenCalledWith('/admin/video/tasks/501/asset-handoffs', { asset_kind: 'video' })
    expect(result).toEqual(response)
  })

  it('keeps the bearer ticket out of the target URL and window name', () => {
    const url = buildQCanvasAssetHandoffTargetURL('http://127.0.0.1:5173/')
    const name = buildQCanvasAssetHandoffWindowName('http://127.0.0.1:8080', 'nonce_1234567890123456')

    expect(url).toBe('http://127.0.0.1:5173/asset-handoff')
    expect(new URL(url).search).toBe('')
    expect(name).toContain('nonce_1234567890123456')
    expect(name).not.toContain('opaque-ticket')
  })

  it('fails explicitly when the QCanvas base URL is not configured', () => {
    expect(() => buildQCanvasAssetHandoffTargetURL('')).toThrow('QCanvas')
  })

  it('refuses to send the bearer ticket to a non-loopback QCanvas origin', () => {
    expect(() => buildQCanvasAssetHandoffTargetURL('https://qcanvas.example.com')).toThrow('loopback')
    expect(() => buildQCanvasAssetHandoffTargetURL('http://user:pass@127.0.0.1:5173')).toThrow(/用户名或密码/)
    expect(() => buildQCanvasAssetHandoffTargetURL('http://127.0.0.1:5173/path?token=secret')).toThrow(/纯 origin/)
  })

  it('sends the ticket only through an exact-origin nonce handshake', () => {
    let messageListener: ((event: MessageEvent<unknown>) => void) | undefined
    const replace = vi.fn()
    const targetPostMessage = vi.fn()
    const targetWindow = {
      name: '',
      location: { replace },
      postMessage: targetPostMessage
    } as unknown as Window
    const sourceWindow = {
      location: { origin: 'http://127.0.0.1:8080' },
      crypto: { randomUUID: () => 'nonce_1234567890123456' },
      addEventListener: (_type: string, listener: EventListenerOrEventListenerObject) => {
        messageListener = listener as (event: MessageEvent<unknown>) => void
      },
      removeEventListener: vi.fn(),
      setTimeout: vi.fn(() => 1),
      clearTimeout: vi.fn()
    } as unknown as Window

    startQCanvasAssetHandoffTransfer(
      targetWindow,
      'opaque_ticket_12345678901234567890',
      'http://127.0.0.1:5173',
      sourceWindow
    )

    expect(replace).toHaveBeenCalledWith('http://127.0.0.1:5173/asset-handoff')
    expect(targetWindow.name).not.toContain('opaque_ticket')
    expect(messageListener).toBeTypeOf('function')
    messageListener?.({
      source: targetWindow,
      origin: 'http://127.0.0.1:5173',
      data: { type: 'qcanvas-asset-handoff-ready', nonce: 'nonce_1234567890123456' }
    } as MessageEvent<unknown>)
    expect(targetPostMessage).toHaveBeenCalledWith({
      type: 'sub2api-asset-handoff-ticket',
      nonce: 'nonce_1234567890123456',
      ticket: 'opaque_ticket_12345678901234567890'
    }, 'http://127.0.0.1:5173')
  })
})
