/** Detect implicit-flow tokens in URL fragments (legacy, insecure). */
export function hasLegacyFragmentOAuthTokens(params: URLSearchParams): boolean {
  return Boolean(params.get('access_token')?.trim())
}

/** Remove hash fragment from the current URL without navigation. */
export function stripUrlFragment(): void {
  if (typeof window === 'undefined') return
  const url = new URL(window.location.href)
  if (!url.hash) return
  window.history.replaceState(null, '', `${url.pathname}${url.search}`)
}
