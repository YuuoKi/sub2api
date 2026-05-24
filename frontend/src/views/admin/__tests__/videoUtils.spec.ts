import { describe, expect, it } from 'vitest'
import {
  candidateToTaskPayload,
  extractRoutingTrace,
  promptAssetCandidates,
  statusLabel,
} from '../video/videoUtils'

describe('video gateway Phase 4B utilities', () => {
  it('keeps prompt asset candidates as reusable task drafts', () => {
    expect(promptAssetCandidates).toHaveLength(5)

    const payload = candidateToTaskPayload(promptAssetCandidates[0])

    expect(payload.provider_account_id).toBe(0)
    expect(payload.prompt).toBe(promptAssetCandidates[0].prompt)
    expect(payload.task_type).toBe(promptAssetCandidates[0].task_type)
    expect(payload.prompt).toContain('短剧')
    expect(payload.duration).toBeGreaterThan(0)
  })

  it('uses boss-readable task status labels', () => {
    expect(statusLabel('queued')).toBe('排队中')
    expect(statusLabel('running')).toBe('生成中')
    expect(statusLabel('succeeded')).toBe('已完成')
    expect(statusLabel('failed')).toBe('失败，需要查看原因')
  })

  it('extracts a safe routing trace without copying unknown secret-like payload fields', () => {
    const trace = extractRoutingTrace({
      provider: 'mock',
      provider_account_id: 12,
      provider_account_name: '演示通道 A - 正常可用',
      routing_strategy: '',
      routing_reason: '',
      events: [
        {
          id: 1,
          video_task_id: 99,
          event_type: 'routed',
          message: 'video task routed',
          created_at: '2026-05-24T00:00:00Z',
          payload_json: {
            strategy: 'least_inflight',
            reason: '选择当前处理中 0、今日失败 0、优先级 10 的可用账号',
            selected_account_id: 12,
            selected_account_name: '演示通道 A - 正常可用',
            provider: 'mock',
            api_key: 'placeholder-key-should-not-copy',
            authorization: 'placeholder-authorization-should-not-copy',
            skipped_accounts: [
              {
                id: 21,
                display_name: 'Seedance 2.0 - 未配置真实密钥',
                provider: 'seedance',
                reason: '真实密钥未配置',
                masked_key: 'sdnc***demo',
              },
            ],
          },
        },
      ],
    })

    expect(trace?.strategy).toBe('least_inflight')
    expect(trace?.selected_account_name).toBe('演示通道 A - 正常可用')
    expect(trace?.skippedAccounts).toEqual([
      {
        id: 21,
        display_name: 'Seedance 2.0 - 未配置真实密钥',
        provider: 'seedance',
        reason: '真实密钥未配置',
      },
    ])
    expect(JSON.stringify(trace)).not.toContain('placeholder-key-should-not-copy')
    expect(JSON.stringify(trace)).not.toContain('placeholder-authorization-should-not-copy')
    expect(JSON.stringify(trace)).not.toContain('sdnc***demo')
  })
})
