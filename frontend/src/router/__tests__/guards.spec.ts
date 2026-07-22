import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import {
  resolveAuthenticatedLandingPath,
  resolveCompletedSetupRedirectPath,
  resolvePostAuthRedirect,
} from '@/router/setupRedirect'
import { isLanAdminWebRouteAllowed } from '@/router'

// Mock 导航加载状态
vi.mock('@/composables/useNavigationLoading', () => {
  const mockStart = vi.fn()
  const mockEnd = vi.fn()
  return {
    useNavigationLoadingState: () => ({
      startNavigation: mockStart,
      endNavigation: mockEnd,
      isLoading: { value: false },
    }),
    useNavigationLoading: () => ({
      startNavigation: mockStart,
      endNavigation: mockEnd,
      isLoading: { value: false },
    }),
  }
})

// Mock 路由预加载
vi.mock('@/composables/useRoutePrefetch', () => ({
  useRoutePrefetch: () => ({
    triggerPrefetch: vi.fn(),
    cancelPendingPrefetch: vi.fn(),
    resetPrefetchState: vi.fn(),
  }),
}))

// Mock API 相关模块
vi.mock('@/api', () => ({
  authAPI: {
    getCurrentUser: vi.fn().mockResolvedValue({ data: {} }),
    logout: vi.fn(),
  },
  isTotp2FARequired: () => false,
}))

vi.mock('@/api/admin/system', () => ({
  checkUpdates: vi.fn(),
}))

vi.mock('@/api/auth', () => ({
  getPublicSettings: vi.fn(),
}))


// 用于测试的 auth 状态
interface MockAuthState {
  isAuthenticated: boolean
  isAdmin: boolean
  isSimpleMode: boolean
  backendModeEnabled: boolean
  hasPendingAuthSession: boolean
  setupNeedsSetup?: boolean
  mustChangePassword?: boolean
}

interface SimulateGuardHooks {
  /** Mirrors router/index.ts adminComplianceStore.fetchStatus when requiresAdmin passes. */
  onAdminComplianceFetch?: () => void
  /** Stand-in for BossOverviewView adminAPI.dashboard.getStats after a successful guard. */
  onAdminOverviewFetch?: () => void
}

/**
 * 将 router/index.ts 中 beforeEach 守卫的核心逻辑提取为可测试的函数
 */
function simulateGuard(
  toPath: string,
  toMeta: Record<string, any>,
  authState: MockAuthState,
  hooks?: SimulateGuardHooks,
): string | null {
  const requiresAuth = toMeta.requiresAuth !== false
  const requiresAdmin = toMeta.requiresAdmin === true

  if (toPath === '/setup' && authState.setupNeedsSetup === false) {
    return resolveCompletedSetupRedirectPath(authState.isAuthenticated, authState.isAdmin)
  }

  if (
    authState.backendModeEnabled &&
    !isLanAdminWebRouteAllowed(toPath, authState.isAuthenticated, authState.isAdmin)
  ) {
    return authState.isAuthenticated && authState.isAdmin
      ? '/admin/console/overview'
      : '/login'
  }

  // 不需要认证的路由
  if (!requiresAuth) {
    if (
      authState.isAuthenticated &&
      (toPath === '/login' || toPath === '/register')
    ) {
      if (authState.backendModeEnabled && !authState.isAdmin) {
        return null
      }
      return resolvePostAuthRedirect(toMeta.redirect, authState.isAdmin)
    }
    return null // 允许通过
  }

  // 需要认证但未登录
  if (!authState.isAuthenticated) {
    return '/login'
  }

  if (authState.mustChangePassword && toPath !== '/change-temporary-password') {
    return '/change-temporary-password'
  }
  if (!authState.mustChangePassword && toPath === '/change-temporary-password') {
    return resolveAuthenticatedLandingPath(authState.isAdmin)
  }

  // 需要管理员但不是管理员 — early return before adminCompliance / overview fetches
  if (requiresAdmin && !authState.isAdmin) {
    return '/dashboard'
  }

  // Mirrors router/index.ts: adminCompliance fetch only after requiresAdmin passes
  if (requiresAdmin && authState.isAdmin) {
    hooks?.onAdminComplianceFetch?.()
  }

  // 简易模式限制
  if (authState.isSimpleMode) {
    const restrictedPaths = [
      '/admin/groups',
      '/admin/subscriptions',
      '/admin/redeem',
      '/subscriptions',
      '/redeem',
    ]
    if (restrictedPaths.some((path) => toPath.startsWith(path))) {
      return resolveAuthenticatedLandingPath(authState.isAdmin)
    }
  }

  if (toPath === '/admin/console/overview' && authState.isAdmin) {
    hooks?.onAdminOverviewFetch?.()
  }

  return null // 允许通过
}

describe('路由守卫逻辑', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  // --- 未认证用户 ---

  describe('未认证用户', () => {
    const authState: MockAuthState = {
      isAuthenticated: false,
      isAdmin: false,
      isSimpleMode: false,
      backendModeEnabled: false,
      hasPendingAuthSession: false,
    }

    it('访问需要认证的页面重定向到 /login', () => {
      const redirect = simulateGuard('/dashboard', {}, authState)
      expect(redirect).toBe('/login')
    })

    it('访问管理页面重定向到 /login', () => {
      const redirect = simulateGuard('/admin/dashboard', { requiresAdmin: true }, authState)
      expect(redirect).toBe('/login')
    })

    it('访问公开页面允许通过', () => {
      const redirect = simulateGuard('/login', { requiresAuth: false }, authState)
      expect(redirect).toBeNull()
    })

    it('访问 /home 公开页面允许通过', () => {
      const redirect = simulateGuard('/home', { requiresAuth: false }, authState)
      expect(redirect).toBeNull()
    })
  })

  // --- 已认证普通用户 ---

  describe('已认证普通用户', () => {
    const authState: MockAuthState = {
      isAuthenticated: true,
      isAdmin: false,
      isSimpleMode: false,
      backendModeEnabled: false,
      hasPendingAuthSession: false,
    }

    it('访问 /login 重定向到 /dashboard', () => {
      const redirect = simulateGuard('/login', { requiresAuth: false }, authState)
      expect(redirect).toBe('/dashboard')
    })

    it('访问 /register 重定向到 /dashboard', () => {
      const redirect = simulateGuard('/register', { requiresAuth: false }, authState)
      expect(redirect).toBe('/dashboard')
    })

    it('访问 /dashboard 允许通过', () => {
      const redirect = simulateGuard('/dashboard', {}, authState)
      expect(redirect).toBeNull()
    })

    it('访问管理页面被拒绝，重定向到 /dashboard', () => {
      const redirect = simulateGuard('/admin/dashboard', { requiresAdmin: true }, authState)
      expect(redirect).toBe('/dashboard')
    })

    it('访问 /admin/users 被拒绝', () => {
      const redirect = simulateGuard('/admin/users', { requiresAdmin: true }, authState)
      expect(redirect).toBe('/dashboard')
    })

    it('普通用户直接访问 /admin/console/* 被拒绝并重定向到 /dashboard', () => {
      for (const path of [
        '/admin/console/overview',
        '/admin/console/key-vault',
        '/admin/console/staff',
        '/admin/console/ai-records',
      ]) {
        const redirect = simulateGuard(path, { requiresAdmin: true }, authState)
        expect(redirect, path).toBe('/dashboard')
      }
    })

    it('simulateGuard: non-admin /admin/console/overview redirects to /dashboard and does not invoke adminCompliance fetch or overview getStats', () => {
      const onAdminComplianceFetch = vi.fn()
      const onAdminOverviewFetch = vi.fn()
      const redirect = simulateGuard(
        '/admin/console/overview',
        { requiresAdmin: true },
        authState,
        { onAdminComplianceFetch, onAdminOverviewFetch },
      )
      expect(redirect).toBe('/dashboard')
      expect(onAdminComplianceFetch).not.toHaveBeenCalled()
      expect(onAdminOverviewFetch).not.toHaveBeenCalled()
    })
  })

  // --- 已认证管理员 ---

  describe('已认证管理员', () => {
    const authState: MockAuthState = {
      isAuthenticated: true,
      isAdmin: true,
      isSimpleMode: false,
      backendModeEnabled: false,
      hasPendingAuthSession: false,
    }

    it('访问 /login 重定向到统一控制台总览', () => {
      const redirect = simulateGuard('/login', { requiresAuth: false }, authState)
      expect(redirect).toBe('/admin/console/overview')
    })

    it('访问 /login 时优先保留显式站内 redirect', () => {
      const redirect = simulateGuard(
        '/login',
        { requiresAuth: false, redirect: '/admin/dashboard' },
        authState,
      )
      expect(redirect).toBe('/admin/dashboard')
    })

    it('访问 /login 时拒绝外部 redirect 并回落统一控制台总览', () => {
      for (const unsafeRedirect of ['https://example.com', '//example.com']) {
        const redirect = simulateGuard(
          '/login',
          { requiresAuth: false, redirect: unsafeRedirect },
          authState,
        )
        expect(redirect).toBe('/admin/console/overview')
      }
    })

    it('访问管理页面允许通过', () => {
      const redirect = simulateGuard('/admin/dashboard', { requiresAdmin: true }, authState)
      expect(redirect).toBeNull()
    })

    it('访问用户页面允许通过', () => {
      const redirect = simulateGuard('/dashboard', {}, authState)
      expect(redirect).toBeNull()
    })

    it('admin simulateGuard for /admin/console/overview allows pass and runs adminCompliance + overview fetch hooks', () => {
      const onAdminComplianceFetch = vi.fn()
      const onAdminOverviewFetch = vi.fn()
      const redirect = simulateGuard(
        '/admin/console/overview',
        { requiresAdmin: true },
        authState,
        { onAdminComplianceFetch, onAdminOverviewFetch },
      )
      expect(redirect).toBeNull()
      expect(onAdminComplianceFetch).toHaveBeenCalledTimes(1)
      expect(onAdminOverviewFetch).toHaveBeenCalledTimes(1)
    })
  })

  // --- 简易模式 ---

  describe('简易模式受限路由', () => {
    it('普通用户简易模式访问 /subscriptions 重定向到 /dashboard', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: false,
        isSimpleMode: true,
        backendModeEnabled: false,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/subscriptions', {}, authState)
      expect(redirect).toBe('/dashboard')
    })

    it('普通用户简易模式访问 /redeem 重定向到 /dashboard', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: false,
        isSimpleMode: true,
        backendModeEnabled: false,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/redeem', {}, authState)
      expect(redirect).toBe('/dashboard')
    })

    it('管理员简易模式访问 /admin/groups 重定向到控制台首页', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: true,
        isSimpleMode: true,
        backendModeEnabled: false,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/admin/groups', { requiresAdmin: true }, authState)
      expect(redirect).toBe('/admin/console/overview')
    })

    it('管理员简易模式访问 /admin/subscriptions 重定向', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: true,
        isSimpleMode: true,
        backendModeEnabled: false,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard(
        '/admin/subscriptions',
        { requiresAdmin: true },
        authState
      )
      expect(redirect).toBe('/admin/console/overview')
    })

    it('简易模式下非受限页面正常访问', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: false,
        isSimpleMode: true,
        backendModeEnabled: false,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/dashboard', {}, authState)
      expect(redirect).toBeNull()
    })

    it('简易模式下 /keys 正常访问', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: false,
        isSimpleMode: true,
        backendModeEnabled: false,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/keys', {}, authState)
      expect(redirect).toBeNull()
    })
  })

  describe('Backend Mode', () => {
    it('unauthenticated: /home redirects to /login', () => {
      const authState: MockAuthState = {
        isAuthenticated: false,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/home', { requiresAuth: false }, authState)
      expect(redirect).toBe('/login')
    })

    it('unauthenticated: /login is allowed', () => {
      const authState: MockAuthState = {
        isAuthenticated: false,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/login', { requiresAuth: false }, authState)
      expect(redirect).toBeNull()
    })

    it('unauthenticated: /key-usage is blocked', () => {
      const authState: MockAuthState = {
        isAuthenticated: false,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/key-usage', { requiresAuth: false }, authState)
      expect(redirect).toBe('/login')
    })

    it('unauthenticated: /setup is allowed', () => {
      const authState: MockAuthState = {
        isAuthenticated: false,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/setup', { requiresAuth: false }, authState)
      expect(redirect).toBeNull()
    })

    it('unauthenticated: initialized /setup redirects to /login', () => {
      const authState: MockAuthState = {
        isAuthenticated: false,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
        setupNeedsSetup: false,
      }
      const redirect = simulateGuard('/setup', { requiresAuth: false }, authState)
      expect(redirect).toBe('/login')
    })

    it('admin: initialized /setup redirects to the unified console overview', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: true,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
        setupNeedsSetup: false,
      }
      const redirect = simulateGuard('/setup', { requiresAuth: false }, authState)
      expect(redirect).toBe('/admin/console/overview')
    })

    it('admin: legacy /admin/dashboard redirects to the unified overview', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: true,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/admin/dashboard', { requiresAdmin: true }, authState)
      expect(redirect).toBe('/admin/console/overview')
    })

    it('admin: /login redirects to the unified console overview', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: true,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/login', { requiresAuth: false }, authState)
      expect(redirect).toBe('/admin/console/overview')
    })

    it('admin with a temporary password is forced through the password-change route', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: true,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
        mustChangePassword: true,
      }

      expect(simulateGuard('/admin/console/overview', { requiresAdmin: true }, authState))
        .toBe('/change-temporary-password')
      expect(simulateGuard('/change-temporary-password', {}, authState)).toBeNull()
    })

    it('admin with a completed password change cannot loop back to the temporary-password route', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: true,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
        mustChangePassword: false,
      }

      expect(simulateGuard('/change-temporary-password', {}, authState))
        .toBe('/admin/console/overview')
    })

    it('non-admin authenticated: /dashboard redirects to /login', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/dashboard', {}, authState)
      expect(redirect).toBe('/login')
    })

    it('non-admin authenticated: /login is allowed (no redirect loop)', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/login', { requiresAuth: false }, authState)
      expect(redirect).toBeNull()
    })

    it('non-admin authenticated: /key-usage is blocked', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/key-usage', { requiresAuth: false }, authState)
      expect(redirect).toBe('/login')
    })

    it('unauthenticated: callback routes are blocked', () => {
      const authState: MockAuthState = {
        isAuthenticated: false,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/auth/wechat/callback', { requiresAuth: false }, authState)
      expect(redirect).toBe('/login')
    })

    it('unauthenticated: WeChat payment callback route is blocked', () => {
      const authState: MockAuthState = {
        isAuthenticated: false,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/auth/wechat/payment/callback', { requiresAuth: false }, authState)
      expect(redirect).toBe('/login')
    })

    it('unauthenticated: /payment/result is blocked', () => {
      const authState: MockAuthState = {
        isAuthenticated: false,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/payment/result', { requiresAuth: false }, authState)
      expect(redirect).toBe('/login')
    })

    it('unauthenticated: /register stays blocked even with a pending auth session', () => {
      const authState: MockAuthState = {
        isAuthenticated: false,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: true,
      }
      const redirect = simulateGuard('/register', { requiresAuth: false }, authState)
      expect(redirect).toBe('/login')
    })

    it('unauthenticated: /email-verify is blocked without a pending auth session', () => {
      const authState: MockAuthState = {
        isAuthenticated: false,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/email-verify', { requiresAuth: false }, authState)
      expect(redirect).toBe('/login')
    })
  })
})
