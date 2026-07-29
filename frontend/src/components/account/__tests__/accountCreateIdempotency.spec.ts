import { describe, expect, it } from 'vitest'
import { createAccountCreateIdempotencySession } from '../accountCreateIdempotency'

describe('account create idempotency session', () => {
  it('reuses one key after a lost response and isolates batch entries', () => {
    const session = createAccountCreateIdempotencySession()

    const firstAttempt = session.keyFor('single')
    expect(session.keyFor('single')).toBe(firstAttempt)
    expect(session.keyFor('batch:0')).not.toBe(firstAttempt)
    expect(session.keyFor('batch:1')).not.toBe(session.keyFor('batch:0'))
  })

  it('starts a new logical session after the modal is reset', () => {
    const session = createAccountCreateIdempotencySession()
    const beforeReset = session.keyFor('single')

    session.reset()

    expect(session.keyFor('single')).not.toBe(beforeReset)
  })
})
