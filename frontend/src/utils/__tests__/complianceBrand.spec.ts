import { describe, expect, it } from 'vitest'
import {
  IN_APP_ADMIN_COMPLIANCE_PATH,
  NEUTRAL_ACK_PHRASE_EN,
  NEUTRAL_ACK_PHRASE_ZH,
  UPSTREAM_ACK_PHRASE_EN,
  UPSTREAM_ACK_PHRASE_ZH,
  containsUpstreamIdentity,
  sanitizeComplianceAckPhrase,
  sanitizeComplianceDocumentUrl
} from '../complianceBrand'

describe('complianceBrand sanitization', () => {
  it('detects Sub2API / Wei-Shaw identity in ack phrases', () => {
    expect(containsUpstreamIdentity('I agree to Sub2API compliance')).toBe(true)
    expect(containsUpstreamIdentity('同意 Wei-Shaw 合规承诺')).toBe(true)
    expect(containsUpstreamIdentity('同意无界 · 企业 AI 中台合规承诺')).toBe(false)
  })

  it('force-neutralizes upstream ack phrases', () => {
    expect(sanitizeComplianceAckPhrase('I have read Sub2API terms', 'en')).toBe(
      NEUTRAL_ACK_PHRASE_EN
    )
    expect(sanitizeComplianceAckPhrase('我已同意 Sub2API 合规承诺', 'zh')).toBe(
      NEUTRAL_ACK_PHRASE_ZH
    )
    expect(sanitizeComplianceAckPhrase(UPSTREAM_ACK_PHRASE_EN, 'en')).toBe(
      NEUTRAL_ACK_PHRASE_EN
    )
    expect(sanitizeComplianceAckPhrase(UPSTREAM_ACK_PHRASE_ZH, 'zh')).toBe(
      NEUTRAL_ACK_PHRASE_ZH
    )
  })

  it('keeps upstream submit fallbacks aligned with backend constants', () => {
    expect(UPSTREAM_ACK_PHRASE_ZH).toBe(
      '我已阅读、理解并同意 Sub2API 部署与运营合规承诺'
    )
    expect(UPSTREAM_ACK_PHRASE_EN).toBe(
      'I have read, understood, and agree to the Sub2API Deployment and Operation Compliance Commitment'
    )
    expect(NEUTRAL_ACK_PHRASE_EN).not.toMatch(/Sub2API/i)
    expect(NEUTRAL_ACK_PHRASE_ZH).not.toMatch(/Sub2API/i)
  })

  it('keeps non-upstream custom phrases', () => {
    expect(sanitizeComplianceAckPhrase('I agree to Acme Corp terms', 'en')).toBe(
      'I agree to Acme Corp terms'
    )
  })

  it('rewrites Wei-Shaw GitHub and unserved /docs URLs to in-app legal route', () => {
    expect(
      sanitizeComplianceDocumentUrl(
        'https://github.com/Wei-Shaw/sub2api/blob/main/docs/legal/admin-compliance.en.md'
      )
    ).toBe(IN_APP_ADMIN_COMPLIANCE_PATH)
    expect(sanitizeComplianceDocumentUrl('/docs/legal/admin-compliance.zh.md')).toBe(
      IN_APP_ADMIN_COMPLIANCE_PATH
    )
    expect(sanitizeComplianceDocumentUrl('/legal/admin-compliance')).toBe(
      IN_APP_ADMIN_COMPLIANCE_PATH
    )
  })
})
