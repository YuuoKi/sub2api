import { describe, expect, it, vi } from 'vitest'

vi.mock('@/utils/productMode', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/utils/productMode')>()
  return { ...actual, isVideoGatewayDemoMode: false }
})

import { routingStrategyLabel } from '../videoUtils'

describe('routingStrategyLabel', () => {
  it.each([
    ['mock_least_inflight', '试跑模式 · 处理中最少'],
    ['review_real_least_inflight', '一次真实复核 · 处理中最少'],
    ['internal_real_least_inflight', '内部真实调用 · 处理中最少'],
  ])('renders the current execution-mode-prefixed strategy %s', (strategy, expected) => {
    expect(routingStrategyLabel(strategy)).toBe(expected)
  })

  it('does not treat the retired unprefixed strategy as a current contract value', () => {
    expect(routingStrategyLabel('least_inflight')).toBe('least_inflight')
  })
})
