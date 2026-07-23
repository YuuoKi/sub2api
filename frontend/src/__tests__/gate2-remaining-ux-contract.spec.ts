import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { getAdminSteps } from '@/components/Guide/steps'

const source = (path: string) => readFileSync(resolve(process.cwd(), path), 'utf8')

describe('Gate 2 remaining UX contract', () => {
  it('does not render registration before the public runtime setting is loaded and enabled', () => {
    const login = source('src/views/auth/LoginView.vue')
    expect(login).toContain('publicSettingsLoaded && registrationEnabled && !backendModeEnabled')
    expect(login).toContain('registrationEnabled.value = settings.registration_enabled === true')
    expect(login).toContain('const registrationEnabled = ref<boolean>(false)')
  })

  it('sends password and 2FA admin logins to the unified console overview by default', () => {
    const login = source('src/views/auth/LoginView.vue')
    expect(login).toContain("import { resolvePostAuthRedirect } from '@/router/setupRedirect'")
    expect(login.match(/resolvePostAuthRedirect\(/g)).toHaveLength(2)
  })

  it('hard-cuts onboarding to four Wujie operator steps with an explicit skip path', () => {
    const steps = getAdminSteps((key) => key)
    expect(steps).toHaveLength(4)
    expect(steps.map((step) => step.element)).toEqual([
      '#sidebar-user-manage',
      '#sidebar-group-manage',
      '#sidebar-channel-manage',
      '#sidebar-global-usage'
    ])
    for (const step of steps) {
      expect(step.popover?.showButtons).toContain('close')
    }

    for (const locale of ['src/i18n/locales/zh/misc.ts', 'src/i18n/locales/en/misc.ts']) {
      const contents = source(locale)
      const onboarding = contents.slice(contents.indexOf('  onboarding:'), contents.indexOf('  payment:'))
      expect(onboarding).not.toContain('Sub2API')
      expect(onboarding).not.toContain('<div')
      expect(onboarding).not.toContain('<ul')
      expect(onboarding).not.toMatch(/[🚀🎉👋💡🎯🔗🔑📊]/u)
    }
  })

  it('uses role-specific usage names in Chinese and English', () => {
    const zhCommon = source('src/i18n/locales/zh/common.ts')
    const enCommon = source('src/i18n/locales/en/common.ts')
    const zhUsage = source('src/i18n/locales/zh/dashboard.ts')
    const enUsage = source('src/i18n/locales/en/dashboard.ts')
    const zhAdmin = source('src/i18n/locales/zh/admin/resources.ts')
    const enAdmin = source('src/i18n/locales/en/admin/resources.ts')
    const roleNav = source('src/components/layout/roleAwareNavigation.ts')

    expect(zhCommon).toContain("myUsage: '我的用量'")
    expect(zhCommon).toContain("globalUsage: '全局用量'")
    expect(enCommon).toContain("myUsage: 'My usage'")
    expect(enCommon).toContain("globalUsage: 'Global usage'")
    expect(zhUsage).toContain("title: '我的用量'")
    expect(enUsage).toContain("title: 'My usage'")
    expect(zhAdmin).toContain("title: '全局用量'")
    expect(enAdmin).toContain("title: 'Global usage'")
    // Role-aware IA uses brand Chinese labels; employee spend + admin global usage stay reachable.
    expect(roleNav).toContain("label: '我的花费'")
    expect(roleNav).toContain("label: '用量与成本'")
  })

  it('adds prerequisite-aware chained empty states without creating data automatically', () => {
    const groups = source('src/views/admin/GroupsView.vue')
    const accounts = source('src/views/admin/AccountsView.vue')
    const keys = source('src/views/user/KeysView.vue')

    expect(groups).toContain("action-to=\"/admin/users\"")
    expect(groups).toContain("admin.groups.emptyPrerequisite")
    expect(accounts).toContain("admin.accounts.emptyPrerequisite")
    expect(accounts).toContain("groups.length === 0")
    expect(accounts).toContain("'/admin/groups'")
    expect(keys).toContain("keys.emptyPrerequisite")
    expect(keys).toContain("groups.length === 0")
    expect(keys).toContain("'/available-channels'")
  })

  it('keeps AppHeader as the only level-one heading owner for video pages', () => {
    for (const path of [
      'src/views/admin/video/VideoProvidersView.vue',
      'src/views/admin/video/VideoTasksView.vue',
      'src/views/admin/video/VideoTaskDetailView.vue',
      'src/views/admin/video/VideoSystemCheckView.vue'
    ]) {
      const view = source(path)
      expect(view).not.toContain('<h1')
      expect(view).toContain('<h2')
    }
  })

  it('keeps the LAN overview read-only and uses only the formal backup surface', () => {
    const overview = source('src/views/admin/console/BossOverviewView.vue')
    expect(overview).not.toContain('updateMonthlyBudget')
    expect(overview).not.toContain('dataManagement')
    expect(overview).not.toContain('listBackupJobs')
  })

  it('uses the staff capability only for non-interactive service identities and unlimited cards', () => {
    const staff = source('src/views/admin/console/StaffView.vue')
    expect(staff).toContain('data-test="create-service-identity"')
    expect(staff).toContain("member_type: 'tool'")
    expect(staff).toContain("role: 'user'")
    expect(staff).toContain('quota: 0')
    expect(staff).not.toContain('InitialCredentialDialog')
    expect(staff).not.toContain('issueForm.quota')
  })

  it('removes local provider-account quota controls from the LAN administrator surface', () => {
    const createAccount = source('src/components/account/CreateAccountModal.vue')
    const editAccount = source('src/components/account/EditAccountModal.vue')
    const actionMenu = source('src/components/admin/account/AccountActionMenu.vue')

    expect(createAccount).toContain("!appStore.lanAdminModeEnabled && form.platform === 'anthropic'")
    expect(createAccount).toContain("!appStore.lanAdminModeEnabled && (form.type === 'apikey' || form.type === 'bedrock')")
    expect(editAccount).toContain("!appStore.lanAdminModeEnabled && account?.platform === 'anthropic'")
    expect(editAccount).toContain("!appStore.lanAdminModeEnabled && (account?.type === 'apikey' || account?.type === 'bedrock')")
    expect(actionMenu).toContain('!appStore.lanAdminModeEnabled &&')

    // LAN admin hides per-account quota controls on create/edit/action surfaces above.
    // OpsSettingsDialog keeps ops-level openai quota auto-pause defaults (not account-local),
    // and SettingsView gates broader admin tabs via local `lanAdminModeEnabled` computed.
    const settingsView = source('src/views/admin/SettingsView.vue')
    expect(settingsView).toContain('const lanAdminModeEnabled = computed(() =>')
    expect(settingsView).toContain('v-if="!lanAdminModeEnabled"')
  })
})
