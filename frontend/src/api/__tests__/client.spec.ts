import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import axios from 'axios'
import type { AxiosInstance } from 'axios'
import { setActivePinia, createPinia } from 'pinia'

// 需要在导入 client 之前设置 mock
vi.mock('@/i18n', () => ({
  getLocale: () => 'zh-CN',
}))

describe('API Client', () => {
  let apiClient: AxiosInstance

  beforeEach(async () => {
    localStorage.clear()
    // 每次测试重新导入以获取干净的模块状态
    vi.resetModules()
    const mod = await import('@/api/client')
    apiClient = mod.apiClient
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  // --- 请求拦截器 ---

  describe('请求拦截器', () => {
    it('自动附加 Authorization 头', async () => {
      localStorage.setItem('auth_token', 'my-jwt-token')

      // 拦截实际请求
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 0, data: {} },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      await apiClient.get('/test')

      const config = adapter.mock.calls[0][0]
      expect(config.headers.get('Authorization')).toBe('Bearer my-jwt-token')
    })

    it('不覆盖调用方显式传入的 Authorization 头', async () => {
      localStorage.setItem('auth_token', 'stored-token')

      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 0, data: {} },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      await apiClient.post('/test', {}, {
        headers: { Authorization: 'Bearer explicit-token' },
      })

      const config = adapter.mock.calls[0][0]
      expect(config.headers.get('Authorization')).toBe('Bearer explicit-token')
    })

    it('无 token 时不附加 Authorization 头', async () => {
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 0, data: {} },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      await apiClient.get('/test')

      const config = adapter.mock.calls[0][0]
      expect(config.headers.get('Authorization')).toBeFalsy()
    })

    it('GET 请求自动附加 timezone 参数', async () => {
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 0, data: {} },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      await apiClient.get('/test')

      const config = adapter.mock.calls[0][0]
      expect(config.params).toHaveProperty('timezone')
    })

    it('POST 请求不附加 timezone 参数', async () => {
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 0, data: {} },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      await apiClient.post('/test', { foo: 'bar' })

      const config = adapter.mock.calls[0][0]
      expect(config.params?.timezone).toBeUndefined()
    })

    it('请求默认带 withCredentials 以支持跨域 cookie', async () => {
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 0, data: {} },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      await apiClient.post('/auth/oauth/bind-token')

      const config = adapter.mock.calls[0][0]
      expect(config.withCredentials).toBe(true)
    })
  })

  // --- 响应拦截器 ---

  describe('响应拦截器', () => {
    it('code=0 时解包 data 字段', async () => {
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 0, data: { name: 'test' }, message: 'ok' },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      const response = await apiClient.get('/test')
      expect(response.data).toEqual({ name: 'test' })
    })

    it('code!=0 时拒绝并返回结构化错误', async () => {
      const adapter = vi.fn().mockResolvedValue({
        status: 200,
        data: { code: 1001, message: '参数错误', data: null },
        headers: {},
        config: {},
        statusText: 'OK',
      })
      apiClient.defaults.adapter = adapter

      await expect(apiClient.get('/test')).rejects.toEqual(
        expect.objectContaining({
          code: 1001,
          message: '参数错误',
        })
      )
    })
  })

  // --- 401 Token 刷新 ---

  describe('401 Token 刷新', () => {
    it('无 refresh_token 时 401 清除 localStorage', async () => {
      localStorage.setItem('auth_token', 'expired-token')
      // 不设置 refresh_token

      // Mock window.location
      const originalLocation = window.location
      Object.defineProperty(window, 'location', {
        value: { ...originalLocation, pathname: '/dashboard', href: '/dashboard' },
        writable: true,
      })

      const adapter = vi.fn().mockRejectedValue({
        response: {
          status: 401,
          data: { code: 'TOKEN_EXPIRED', message: 'Token expired' },
        },
        config: {
          url: '/test',
          headers: { Authorization: 'Bearer expired-token' },
        },
        code: 'ERR_BAD_REQUEST',
      })
      apiClient.defaults.adapter = adapter

      await expect(apiClient.get('/test')).rejects.toBeDefined()

      expect(localStorage.getItem('auth_token')).toBeNull()

      // 恢复 location
      Object.defineProperty(window, 'location', {
        value: originalLocation,
        writable: true,
      })
    })

    it('401 刷新成功后同步 Pinia token 与 localStorage', async () => {
      localStorage.setItem('auth_token', 'expired-token')
      localStorage.setItem('refresh_token', 'refresh-token')

      setActivePinia(createPinia())
      const { useAuthStore } = await import('@/stores/auth')
      const store = useAuthStore()
      store.$patch({ token: 'expired-token' })

      const refreshSpy = vi.spyOn(axios, 'post').mockResolvedValue({
        data: {
          code: 0,
          data: {
            access_token: 'fresh-token',
            refresh_token: 'fresh-refresh',
            expires_in: 3600,
          },
        },
      })

      let callCount = 0
      const adapter = vi.fn().mockImplementation(async (config: { url?: string }) => {
        callCount += 1
        if (callCount === 1) {
          return Promise.reject({
            response: {
              status: 401,
              data: { code: 'TOKEN_EXPIRED', message: 'Token expired' },
            },
            config: {
              url: '/test',
              headers: { Authorization: 'Bearer expired-token' },
            },
            code: 'ERR_BAD_REQUEST',
          })
        }

        return {
          status: 200,
          data: { code: 0, data: { ok: true } },
          headers: {},
          config,
          statusText: 'OK',
        }
      })
      apiClient.defaults.adapter = adapter

      const response = await apiClient.get('/test')

      expect(refreshSpy).toHaveBeenCalledTimes(1)
      expect(response.data).toEqual({ ok: true })
      expect(localStorage.getItem('auth_token')).toBe('fresh-token')
      expect(store.token).toBe('fresh-token')
    })

    it('并发 401 重试只触发一次 refresh', async () => {
      localStorage.setItem('auth_token', 'expired-token')
      localStorage.setItem('refresh_token', 'refresh-token')

      const refreshSpy = vi.spyOn(axios, 'post').mockImplementation(async () => {
        await new Promise((resolve) => setTimeout(resolve, 20))
        return {
          data: {
            code: 0,
            data: {
              access_token: 'fresh-token',
              refresh_token: 'fresh-refresh',
              expires_in: 3600,
            },
          },
        }
      })

      let callCount = 0
      const adapter = vi.fn().mockImplementation(async (config: { url?: string }) => {
        callCount += 1
        if (callCount <= 2) {
          return Promise.reject({
            response: {
              status: 401,
              data: { code: 'TOKEN_EXPIRED', message: 'Token expired' },
            },
            config: {
              url: config.url,
              headers: { Authorization: 'Bearer expired-token' },
            },
            code: 'ERR_BAD_REQUEST',
          })
        }

        return {
          status: 200,
          data: { code: 0, data: { ok: true } },
          headers: {},
          config,
          statusText: 'OK',
        }
      })
      apiClient.defaults.adapter = adapter

      await Promise.all([apiClient.get('/one'), apiClient.get('/two')])

      expect(refreshSpy).toHaveBeenCalledTimes(1)
    })
  })

  // --- 网络错误 ---

  describe('网络错误', () => {
    it('网络错误返回 status 0 的错误', async () => {
      const adapter = vi.fn().mockRejectedValue({
        code: 'ERR_NETWORK',
        message: 'Network Error',
        config: { url: '/test' },
        // 没有 response
      })
      apiClient.defaults.adapter = adapter

      await expect(apiClient.get('/test')).rejects.toEqual(
        expect.objectContaining({
          status: 0,
          message: 'Network error. Please check your connection.',
        })
      )
    })
  })

  // --- 请求取消 ---

  describe('请求取消', () => {
    it('取消的请求保持原始取消错误', async () => {
      const source = axios.CancelToken.source()

      const adapter = vi.fn().mockRejectedValue(
        new axios.Cancel('Operation canceled')
      )
      apiClient.defaults.adapter = adapter

      await expect(
        apiClient.get('/test', { cancelToken: source.token })
      ).rejects.toBeDefined()
    })
  })
})
