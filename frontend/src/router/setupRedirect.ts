export const ADMIN_CONSOLE_HOME_PATH = '/admin/console/overview'
export const USER_HOME_PATH = '/dashboard'

export function resolveAuthenticatedLandingPath(isAdmin: boolean): string {
  return isAdmin ? ADMIN_CONSOLE_HOME_PATH : USER_HOME_PATH
}

export function resolvePostAuthRedirect(redirect: unknown, isAdmin: boolean): string {
  if (typeof redirect === 'string' && redirect.startsWith('/') && !redirect.startsWith('//')) {
    return redirect
  }

  return resolveAuthenticatedLandingPath(isAdmin)
}

export function resolveCompletedSetupRedirectPath(isAuthenticated: boolean, isAdmin: boolean): string {
  if (!isAuthenticated) {
    return '/login'
  }

  return resolveAuthenticatedLandingPath(isAdmin)
}
