import { createIdempotencyKey } from '@/utils/idempotencyKey'

export interface AccountCreateIdempotencySession {
  keyFor: (scope: string) => string
  reset: () => void
}

/**
 * Keeps one stable key per logical account for the lifetime of an open create
 * modal. A lost response can therefore be retried without creating a second
 * account, while batch entries still receive distinct keys.
 */
export function createAccountCreateIdempotencySession(): AccountCreateIdempotencySession {
  const keys = new Map<string, string>()

  return {
    keyFor(scope: string): string {
      const normalized = scope.trim() || 'single'
      const existing = keys.get(normalized)
      if (existing) return existing
      const created = createIdempotencyKey()
      keys.set(normalized, created)
      return created
    },
    reset(): void {
      keys.clear()
    },
  }
}
