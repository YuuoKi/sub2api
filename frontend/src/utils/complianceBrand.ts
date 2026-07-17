/** In-app legal route for admin compliance (never upstream GitHub / unserved /docs). */
export const IN_APP_ADMIN_COMPLIANCE_PATH = '/legal/admin-compliance'

export const NEUTRAL_ACK_PHRASE_ZH =
  '我已阅读、理解并同意无界 · 企业 AI 中台的部署与运营合规承诺'
export const NEUTRAL_ACK_PHRASE_EN =
  'I have read, understood, and agree to the 无界 · 企业 AI 中台 Deployment and Operation Compliance Commitment'

/**
 * Backend-expected ack phrases (admin_compliance.go AdminComplianceAckPhraseZH/EN).
 * Display always uses neutralized phrases; these are only for silent submit remap
 * when the API omitted ack_phrase_* (last-resort fallback).
 */
export const UPSTREAM_ACK_PHRASE_ZH =
  '我已阅读、理解并同意 Sub2API 部署与运营合规承诺'
export const UPSTREAM_ACK_PHRASE_EN =
  'I have read, understood, and agree to the Sub2API Deployment and Operation Compliance Commitment'

const UPSTREAM_IDENTITY = /sub2api|wei-?shaw/i
const UPSTREAM_DOC_HOST =
  /github\.com\/wei-shaw|sub2api\.(io|org)|raw\.githubusercontent\.com\/wei-shaw/i

export function containsUpstreamIdentity(value?: string | null): boolean {
  return Boolean(value && UPSTREAM_IDENTITY.test(value))
}

export function isUpstreamComplianceDocumentUrl(url?: string | null): boolean {
  if (!url?.trim()) return true
  const trimmed = url.trim()
  if (UPSTREAM_DOC_HOST.test(trimmed)) return true
  if (/^https?:\/\//i.test(trimmed)) return true
  if (trimmed.includes('/docs/')) return true
  return false
}

export function sanitizeComplianceAckPhrase(
  phrase: string | undefined | null,
  locale: 'zh' | 'en'
): string {
  const fallback = locale === 'zh' ? NEUTRAL_ACK_PHRASE_ZH : NEUTRAL_ACK_PHRASE_EN
  if (!phrase?.trim() || containsUpstreamIdentity(phrase)) {
    return fallback
  }
  return phrase.trim()
}

export function sanitizeComplianceDocumentUrl(url?: string | null): string {
  if (isUpstreamComplianceDocumentUrl(url)) {
    return IN_APP_ADMIN_COMPLIANCE_PATH
  }
  const trimmed = url!.trim()
  if (trimmed.startsWith('/legal/')) {
    return trimmed
  }
  return IN_APP_ADMIN_COMPLIANCE_PATH
}
