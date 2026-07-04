import { beforeEach, describe, expect, it, vi } from 'vitest'
import { subscribeQPS } from '@/api/admin/ops'

const { apiPostMock } = vi.hoisted(() => ({
  apiPostMock: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    post: (...args: any[]) => apiPostMock(...args),
  },
}))

class MockWebSocket {
  static instances: MockWebSocket[] = []

  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3

  readonly url: string
  readonly protocols?: string | string[]
  readyState = MockWebSocket.OPEN
  onopen: ((event: Event) => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  onerror: ((event: Event) => void) | null = null
  onclose: ((event: CloseEvent) => void) | null = null

  constructor(url: string, protocols?: string | string[]) {
    this.url = url
    this.protocols = protocols
    MockWebSocket.instances.push(this)
  }

  close(): void {
    this.readyState = MockWebSocket.CLOSED
    this.onclose?.({ code: 1000 } as CloseEvent)
  }
}

describe('ops realtime websocket client', () => {
  beforeEach(() => {
    apiPostMock.mockReset()
    MockWebSocket.instances = []
    window.localStorage.clear()
    Object.defineProperty(window, 'WebSocket', {
      configurable: true,
      value: MockWebSocket,
    })
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: {
        protocol: 'https:',
        host: 'app.example.com',
      },
    })
  })

  it('requests a one-use ticket and does not put JWT in websocket protocols', async () => {
    window.localStorage.setItem('auth_token', 'stored-token')
    apiPostMock.mockResolvedValue({ data: { ticket: 'ticket-1' } })

    const unsubscribe = subscribeQPS(vi.fn(), {
      token: 'explicit-token',
      wsBaseUrl: 'api.example.com',
    })
    await vi.waitFor(() => expect(MockWebSocket.instances).toHaveLength(1))

    expect(apiPostMock).toHaveBeenCalledWith('/admin/ops/ws/ticket', {}, {
      headers: { Authorization: 'Bearer explicit-token' },
    })
    const ws = MockWebSocket.instances[0]
    expect(ws.protocols).toEqual(['sub2api-admin'])
    expect(ws.url).toBe('wss://api.example.com/api/v1/admin/ops/ws/qps?ticket=ticket-1')
    expect(ws.url).not.toContain('jwt.')

    unsubscribe()
  })

  it('does not create a websocket after unsubscribe while ticket request is in flight', async () => {
    let resolveTicket!: (value: { data: { ticket: string } }) => void
    apiPostMock.mockReturnValue(new Promise((resolve) => {
      resolveTicket = resolve
    }))

    const unsubscribe = subscribeQPS(vi.fn(), {
      token: 'explicit-token',
      wsBaseUrl: 'api.example.com',
    })
    await vi.waitFor(() => expect(apiPostMock).toHaveBeenCalledTimes(1))

    unsubscribe()
    resolveTicket({ data: { ticket: 'late-ticket' } })
    await vi.dynamicImportSettled()

    expect(MockWebSocket.instances).toHaveLength(0)
  })
})
