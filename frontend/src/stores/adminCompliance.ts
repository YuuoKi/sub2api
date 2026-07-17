import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import adminComplianceAPI, { type AdminComplianceStatus } from '@/api/admin/compliance'
import { getLocale } from '@/i18n'
import {
  IN_APP_ADMIN_COMPLIANCE_PATH,
  NEUTRAL_ACK_PHRASE_EN,
  NEUTRAL_ACK_PHRASE_ZH,
  UPSTREAM_ACK_PHRASE_EN,
  UPSTREAM_ACK_PHRASE_ZH,
  sanitizeComplianceAckPhrase,
  sanitizeComplianceDocumentUrl
} from '@/utils/complianceBrand'

function sanitizeStatus(status: AdminComplianceStatus): AdminComplianceStatus {
  return {
    ...status,
    ack_phrase_zh: sanitizeComplianceAckPhrase(status.ack_phrase_zh, 'zh'),
    ack_phrase_en: sanitizeComplianceAckPhrase(status.ack_phrase_en, 'en'),
    document_url_zh: sanitizeComplianceDocumentUrl(status.document_url_zh),
    document_url_en: sanitizeComplianceDocumentUrl(status.document_url_en)
  }
}

export const useAdminComplianceStore = defineStore('adminCompliance', () => {
  const status = ref<AdminComplianceStatus | null>(null)
  const loading = ref(false)
  const submitting = ref(false)
  const initialized = ref(false)
  const forceVisible = ref(false)

  /** Raw backend ack phrases for POST remap — never exposed to UI. */
  let submitPhraseZh: string | null = null
  let submitPhraseEn: string | null = null

  const required = computed(() => status.value?.required === true)
  const shouldShow = computed(() => required.value || forceVisible.value)
  const currentLocale = computed(() => getLocale())
  const expectedPhrase = computed(() => {
    if (currentLocale.value === 'zh') {
      return status.value?.ack_phrase_zh || NEUTRAL_ACK_PHRASE_ZH
    }
    return status.value?.ack_phrase_en || NEUTRAL_ACK_PHRASE_EN
  })
  const documentUrl = computed(() => {
    if (currentLocale.value === 'zh') {
      return status.value?.document_url_zh || IN_APP_ADMIN_COMPLIANCE_PATH
    }
    return status.value?.document_url_en || IN_APP_ADMIN_COMPLIANCE_PATH
  })

  function retainSubmitPhrases(raw: {
    ack_phrase_zh?: string | null
    ack_phrase_en?: string | null
  }): void {
    const zh = raw.ack_phrase_zh?.trim()
    const en = raw.ack_phrase_en?.trim()
    if (zh) {
      submitPhraseZh = zh
    } else if (!submitPhraseZh) {
      submitPhraseZh = UPSTREAM_ACK_PHRASE_ZH
    }
    if (en) {
      submitPhraseEn = en
    } else if (!submitPhraseEn) {
      submitPhraseEn = UPSTREAM_ACK_PHRASE_EN
    }
  }

  function resolveSubmitPhrase(typedPhrase: string): string | null {
    const trimmed = typedPhrase.trim()
    if (trimmed !== expectedPhrase.value) {
      return null
    }
    if (currentLocale.value === 'zh') {
      return submitPhraseZh || UPSTREAM_ACK_PHRASE_ZH
    }
    return submitPhraseEn || UPSTREAM_ACK_PHRASE_EN
  }

  async function fetchStatus(): Promise<AdminComplianceStatus> {
    loading.value = true
    try {
      const raw = await adminComplianceAPI.getStatus()
      retainSubmitPhrases(raw)
      const nextStatus = sanitizeStatus(raw)
      status.value = nextStatus
      initialized.value = true
      forceVisible.value = nextStatus.required
      return nextStatus
    } finally {
      loading.value = false
    }
  }

  async function accept(phrase: string): Promise<AdminComplianceStatus> {
    const submitPhrase = resolveSubmitPhrase(phrase)
    if (!submitPhrase) {
      throw new Error('confirmation phrase does not match')
    }

    submitting.value = true
    try {
      const raw = await adminComplianceAPI.accept({
        phrase: submitPhrase,
        language: currentLocale.value
      })
      retainSubmitPhrases(raw)
      const nextStatus = sanitizeStatus(raw)
      status.value = nextStatus
      forceVisible.value = nextStatus.required
      return nextStatus
    } finally {
      submitting.value = false
    }
  }

  function requireAcknowledgement(partialStatus?: Partial<AdminComplianceStatus>): void {
    retainSubmitPhrases({
      ack_phrase_zh: partialStatus?.ack_phrase_zh,
      ack_phrase_en: partialStatus?.ack_phrase_en
    })
    status.value = sanitizeStatus({
      required: true,
      version: partialStatus?.version || status.value?.version || 'v2026.06.10',
      document_path_zh:
        partialStatus?.document_path_zh ||
        status.value?.document_path_zh ||
        'docs/legal/admin-compliance.zh.md',
      document_path_en:
        partialStatus?.document_path_en ||
        status.value?.document_path_en ||
        'docs/legal/admin-compliance.en.md',
      document_url_zh:
        partialStatus?.document_url_zh ||
        status.value?.document_url_zh ||
        IN_APP_ADMIN_COMPLIANCE_PATH,
      document_url_en:
        partialStatus?.document_url_en ||
        status.value?.document_url_en ||
        IN_APP_ADMIN_COMPLIANCE_PATH,
      ack_phrase_zh:
        partialStatus?.ack_phrase_zh || status.value?.ack_phrase_zh || NEUTRAL_ACK_PHRASE_ZH,
      ack_phrase_en:
        partialStatus?.ack_phrase_en || status.value?.ack_phrase_en || NEUTRAL_ACK_PHRASE_EN,
      acknowledgement: status.value?.acknowledgement
    })
    initialized.value = true
    forceVisible.value = true
  }

  function reset(): void {
    status.value = null
    loading.value = false
    submitting.value = false
    initialized.value = false
    forceVisible.value = false
    submitPhraseZh = null
    submitPhraseEn = null
  }

  return {
    status,
    loading,
    submitting,
    initialized,
    required,
    shouldShow,
    expectedPhrase,
    documentUrl,
    fetchStatus,
    accept,
    requireAcknowledgement,
    reset
  }
})
