import { describe, expect, it, vi } from 'vitest'

vi.mock('@/utils/productMode', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/utils/productMode')>()
  return { ...actual, isVideoGatewayDemoMode: false }
})

import { routingStrategyLabel, taskPhaseLabel } from '../videoUtils'

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

describe('taskPhaseLabel', () => {
  it('maps real task status to a plain-language phase without inventing progress', () => {
    expect(taskPhaseLabel({ status: 'queued' })).toBe('排队中')
    expect(taskPhaseLabel({ status: 'submitted' })).toBe('已提交')
    expect(taskPhaseLabel({ status: 'running' })).toBe('生成中')
  })

  it('prefers the archive next_action as the near-completion phase', () => {
    expect(taskPhaseLabel({ status: 'running', next_action: 'archive' })).toBe('即将完成')
  })

  it('falls back to a neutral in-progress label for unknown status', () => {
    expect(taskPhaseLabel({ status: 'something-else' })).toBe('处理中')
  })
})
