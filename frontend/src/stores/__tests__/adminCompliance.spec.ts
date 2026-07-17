import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import {
  IN_APP_ADMIN_COMPLIANCE_PATH,
  NEUTRAL_ACK_PHRASE_EN,
  NEUTRAL_ACK_PHRASE_ZH,
  UPSTREAM_ACK_PHRASE_EN,
  UPSTREAM_ACK_PHRASE_ZH
} from '@/utils/complianceBrand'

const getStatus = vi.fn()
const accept = vi.fn()
const localeState = vi.hoisted(() => ({ value: 'en' }))

vi.mock('@/api/admin/compliance', () => ({
  default: {
    getStatus: (...args: unknown[]) => getStatus(...args),
    accept: (...args: unknown[]) => accept(...args)
  }
}))

vi.mock('@/i18n', () => ({
  getLocale: () => localeState.value
}))

const upstreamApiStatus = {
  required: true,
  version: 'v2026.06.10',
  document_path_zh: 'docs/legal/admin-compliance.zh.md',
  document_path_en: 'docs/legal/admin-compliance.en.md',
  document_url_zh:
    'https://github.com/Wei-Shaw/sub2api/blob/main/docs/legal/admin-compliance.zh.md',
  document_url_en:
    'https://github.com/Wei-Shaw/sub2api/blob/main/docs/legal/admin-compliance.en.md',
  ack_phrase_zh: UPSTREAM_ACK_PHRASE_ZH,
  ack_phrase_en: UPSTREAM_ACK_PHRASE_EN
}

describe('adminCompliance store brand sanitization', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    getStatus.mockReset()
    accept.mockReset()
    localeState.value = 'en'
  })

  it('neutralizes Sub2API phrases and Wei-Shaw GitHub URLs from API status', async () => {
    getStatus.mockResolvedValueOnce({ ...upstreamApiStatus })

    const { useAdminComplianceStore } = await import('../adminCompliance')
    const store = useAdminComplianceStore()
    await store.fetchStatus()

    expect(store.expectedPhrase).toBe(NEUTRAL_ACK_PHRASE_EN)
    expect(store.expectedPhrase).not.toMatch(/Sub2API/i)
    expect(store.expectedPhrase).not.toContain('github.com/Wei-Shaw')
    expect(store.documentUrl).toBe(IN_APP_ADMIN_COMPLIANCE_PATH)
    expect(store.status?.ack_phrase_zh).toBe(NEUTRAL_ACK_PHRASE_ZH)
    expect(store.status?.ack_phrase_en).toBe(NEUTRAL_ACK_PHRASE_EN)
    expect(store.status?.document_url_en).not.toContain('github.com/Wei-Shaw')
    expect(store.status?.document_url_en).toBe(IN_APP_ADMIN_COMPLIANCE_PATH)
  })

  it('requireAcknowledgement also sanitizes upstream identity payloads', async () => {
    const { useAdminComplianceStore } = await import('../adminCompliance')
    const store = useAdminComplianceStore()
    store.requireAcknowledgement({
      ack_phrase_en: 'Agree to Sub2API terms',
      document_url_en: 'https://github.com/Wei-Shaw/sub2api/blob/main/docs/legal/x.md'
    })

    expect(store.expectedPhrase).toBe(NEUTRAL_ACK_PHRASE_EN)
    expect(store.expectedPhrase).not.toMatch(/Sub2API/i)
    expect(store.documentUrl).toBe(IN_APP_ADMIN_COMPLIANCE_PATH)
  })

  it('accept remaps neutralized typed phrase to retained backend Sub2API phrase', async () => {
    getStatus.mockResolvedValueOnce({ ...upstreamApiStatus })
    accept.mockResolvedValueOnce({
      ...upstreamApiStatus,
      required: false,
      acknowledgement: {
        version: 'v2026.06.10',
        document_zh: 'docs/legal/admin-compliance.zh.md',
        document_en: 'docs/legal/admin-compliance.en.md',
        admin_user_id: 1,
        accepted_at: '2026-07-18T00:00:00Z'
      }
    })

    const { useAdminComplianceStore } = await import('../adminCompliance')
    const store = useAdminComplianceStore()
    await store.fetchStatus()

    expect(store.expectedPhrase).toBe(NEUTRAL_ACK_PHRASE_EN)
    await store.accept(NEUTRAL_ACK_PHRASE_EN)

    expect(accept).toHaveBeenCalledTimes(1)
    expect(accept).toHaveBeenCalledWith({
      phrase: UPSTREAM_ACK_PHRASE_EN,
      language: 'en'
    })
    expect(accept.mock.calls[0][0].phrase).toBe(
      'I have read, understood, and agree to the Sub2API Deployment and Operation Compliance Commitment'
    )
  })

  it('accept remaps the neutral Chinese phrase to the retained backend phrase', async () => {
    localeState.value = 'zh'
    getStatus.mockResolvedValueOnce({ ...upstreamApiStatus })
    accept.mockResolvedValueOnce({ ...upstreamApiStatus, required: false })

    const { useAdminComplianceStore } = await import('../adminCompliance')
    const store = useAdminComplianceStore()
    await store.fetchStatus()

    expect(store.expectedPhrase).toBe(NEUTRAL_ACK_PHRASE_ZH)
    await store.accept(NEUTRAL_ACK_PHRASE_ZH)

    expect(accept).toHaveBeenCalledWith({
      phrase: UPSTREAM_ACK_PHRASE_ZH,
      language: 'zh'
    })
  })

  it('accept does not submit when typed text does not match displayed phrase', async () => {
    getStatus.mockResolvedValueOnce({ ...upstreamApiStatus })

    const { useAdminComplianceStore } = await import('../adminCompliance')
    const store = useAdminComplianceStore()
    await store.fetchStatus()

    await expect(store.accept('wrong phrase')).rejects.toThrow(
      'confirmation phrase does not match'
    )
    expect(accept).not.toHaveBeenCalled()
  })

  it('accept falls back to upstream constants when API omitted ack phrases', async () => {
    getStatus.mockResolvedValueOnce({
      required: true,
      version: 'v2026.06.10',
      document_path_zh: 'docs/legal/admin-compliance.zh.md',
      document_path_en: 'docs/legal/admin-compliance.en.md',
      document_url_zh: '/legal/admin-compliance',
      document_url_en: '/legal/admin-compliance',
      ack_phrase_zh: '',
      ack_phrase_en: ''
    })
    accept.mockResolvedValueOnce({
      required: false,
      version: 'v2026.06.10',
      document_path_zh: 'docs/legal/admin-compliance.zh.md',
      document_path_en: 'docs/legal/admin-compliance.en.md',
      document_url_zh: '/legal/admin-compliance',
      document_url_en: '/legal/admin-compliance',
      ack_phrase_zh: '',
      ack_phrase_en: ''
    })

    const { useAdminComplianceStore } = await import('../adminCompliance')
    const store = useAdminComplianceStore()
    await store.fetchStatus()

    expect(store.expectedPhrase).toBe(NEUTRAL_ACK_PHRASE_EN)
    await store.accept(NEUTRAL_ACK_PHRASE_EN)

    expect(accept).toHaveBeenCalledWith({
      phrase: UPSTREAM_ACK_PHRASE_EN,
      language: 'en'
    })
  })
})
