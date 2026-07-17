import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import {
  IN_APP_ADMIN_COMPLIANCE_PATH,
  NEUTRAL_ACK_PHRASE_EN
} from '@/utils/complianceBrand'
import AdminComplianceDialog from '../AdminComplianceDialog.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/i18n', () => ({
  getLocale: () => 'en'
}))

vi.mock('@/stores', () => ({
  useAdminComplianceStore: () => ({
    shouldShow: true,
    expectedPhrase: NEUTRAL_ACK_PHRASE_EN,
    documentUrl: IN_APP_ADMIN_COMPLIANCE_PATH,
    status: {
      version: 'v2026.06.10',
      ack_phrase_en: NEUTRAL_ACK_PHRASE_EN,
      document_url_en: IN_APP_ADMIN_COMPLIANCE_PATH
    },
    submitting: false,
    accept: vi.fn()
  }),
  useAuthStore: () => ({
    isAuthenticated: true,
    isAdmin: true,
    logout: vi.fn()
  }),
  useAppStore: () => ({
    showSuccess: vi.fn(),
    showError: vi.fn()
  })
}))

vi.mock('@/content/admin-compliance.zh.md?raw', () => ({ default: '# zh' }))
vi.mock('@/content/admin-compliance.en.md?raw', () => ({ default: '# en' }))

describe('AdminComplianceDialog brand fail-closed', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('shows neutralized phrase and never navigates to github.com/Wei-Shaw', () => {
    const wrapper = mount(AdminComplianceDialog, {
      global: {
        stubs: {
          BaseDialog: {
            template: '<div><slot /><slot name="footer" /></div>',
            props: ['show', 'title']
          },
          Input: true,
          Icon: true,
          RouterLink: {
            props: ['to'],
            template:
              '<a :href="typeof to === \'string\' ? to : to?.path" data-testid="admin-compliance-document-link"><slot /></a>'
          }
        }
      }
    })

    expect(wrapper.text()).toContain(NEUTRAL_ACK_PHRASE_EN)
    expect(wrapper.text()).not.toMatch(/Sub2API/i)
    expect(wrapper.text()).not.toContain('github.com/Wei-Shaw')
    expect(wrapper.html()).not.toMatch(/Sub2API/i)

    const link = wrapper.get('[data-testid="admin-compliance-document-link"]')
    const href = String(link.attributes('href') || '')
    expect(href).toBe(IN_APP_ADMIN_COMPLIANCE_PATH)
    expect(href).not.toContain('github.com')
    expect(href).not.toContain('Wei-Shaw')
  })
})
