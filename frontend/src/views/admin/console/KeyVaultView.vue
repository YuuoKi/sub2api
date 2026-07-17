<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 border-b border-gray-200 pb-4 dark:border-dark-700 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 class="ui-heading">密钥库</h1>
          <p class="ui-subheading mt-1">
            把老板手上的上游密钥统一收进来：文字与作图账号、视频通道，各归各位。密钥加密保存，前端只显示脱敏状态。
          </p>
        </div>
        <div class="flex flex-wrap gap-2">
          <button class="btn btn-outline" type="button" :disabled="loading" @click="reload">
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
            刷新
          </button>
          <button v-if="activeTab === 'accounts'" class="btn btn-primary" type="button" data-test="open-create-account" @click="openCreate">
            <Icon name="plus" size="sm" />
            录入 AI 账号
          </button>
          <RouterLink v-else class="btn btn-primary" to="/admin/video/providers">
            <Icon name="plus" size="sm" />
            管理视频通道
          </RouterLink>
        </div>
      </div>

      <!-- Tab 切换 -->
      <div class="inline-flex rounded-lg border border-gray-200 bg-gray-50 p-0.5 dark:border-dark-700 dark:bg-dark-800">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          type="button"
          class="rounded-md px-4 py-1.5 text-sm font-medium transition-colors"
          :class="activeTab === tab.key
            ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
            : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'"
          @click="activeTab = tab.key"
        >
          {{ tab.label }}
        </button>
      </div>

      <!-- 通用 AI 账号 -->
      <section v-show="activeTab === 'accounts'" class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs uppercase text-gray-500 dark:bg-dark-700/40 dark:text-gray-400">
              <tr>
                <th class="px-5 py-3 font-medium">名称</th>
                <th class="px-5 py-3 font-medium">平台</th>
                <th class="px-5 py-3 font-medium">接入方式</th>
                <th class="px-5 py-3 font-medium">状态</th>
                <th class="px-5 py-3 font-medium">最近使用</th>
                <th class="px-5 py-3 font-medium">备注</th>
                <th class="px-5 py-3 text-right font-medium">操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="account in accounts" :key="account.id" class="hover:bg-gray-50 dark:hover:bg-dark-700/40">
                <td class="px-5 py-3 font-medium text-gray-900 dark:text-white">{{ account.name }}</td>
                <td class="px-5 py-3">
                  <span class="inline-flex rounded-md px-2 py-1 text-xs font-medium" :class="platformBadgeClass(account.platform)">
                    {{ platformLabel(account.platform) }}
                  </span>
                </td>
                <td class="px-5 py-3 text-gray-600 dark:text-gray-300">{{ accountTypeLabel(account.type) }}</td>
                <td class="px-5 py-3">
                  <span class="inline-flex rounded-md px-2 py-1 text-xs font-medium" :class="accountStatusClass(account)">
                    {{ accountStatusLabel(account) }}
                  </span>
                </td>
                <td class="px-5 py-3 text-xs text-gray-500 dark:text-gray-400">{{ formatDateTime(account.last_used_at) }}</td>
                <td class="max-w-[200px] truncate px-5 py-3 text-xs text-gray-500 dark:text-gray-400" :title="account.notes || ''">
                  {{ account.notes || '—' }}
                </td>
                <td class="px-5 py-3">
                  <div class="flex justify-end gap-1.5">
                    <button
                      class="btn btn-sm btn-outline"
                      type="button"
                      :disabled="testingAccountId === account.id"
                      @click="testAccount(account)"
                    >
                      <Icon name="beaker" size="sm" :class="{ 'animate-pulse': testingAccountId === account.id }" />
                      测试
                    </button>
                    <button class="btn btn-sm btn-outline" type="button" @click="openEditAccount(account)">
                      <Icon name="edit" size="sm" />
                      编辑
                    </button>
                    <button
                      class="btn btn-sm btn-outline"
                      type="button"
                      @click="toggleAccountStatus(account)"
                    >
                      {{ account.status === 'active' ? '停用' : '启用' }}
                    </button>
                    <button class="btn btn-sm btn-outline !text-red-600 dark:!text-red-400" type="button" @click="removeAccount(account)">
                      <Icon name="trash" size="sm" />
                    </button>
                  </div>
                </td>
              </tr>
              <tr v-if="!loading && !accounts.length">
                <td colspan="7" class="px-5 py-10 text-center text-sm text-gray-500 dark:text-gray-400">
                  还没有录入任何 AI 账号。点右上角「录入 AI 账号」，把老板手上的密钥收进来。
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-if="accountsTotal > accounts.length" class="border-t border-gray-100 px-5 py-3 text-xs text-gray-500 dark:border-dark-700 dark:text-gray-400">
          共 {{ accountsTotal }} 个账号，当前展示前 {{ accounts.length }} 个。
        </div>
      </section>

      <!-- 视频通道：只读摘要，管理入口在 /admin/video/providers（含一次性真实调用授权门禁，不在此处复制） -->
      <section v-show="activeTab === 'video'" class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs uppercase text-gray-500 dark:bg-dark-700/40 dark:text-gray-400">
              <tr>
                <th class="px-5 py-3 font-medium">通道名称</th>
                <th class="px-5 py-3 font-medium">员工组</th>
                <th class="px-5 py-3 font-medium">默认模型</th>
                <th class="px-5 py-3 font-medium">凭证状态</th>
                <th class="px-5 py-3 font-medium">是否启用</th>
                <th class="px-5 py-3 text-right font-medium">操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="provider in providers" :key="provider.id" class="hover:bg-gray-50 dark:hover:bg-dark-700/40">
                <td class="px-5 py-3 font-medium text-gray-900 dark:text-white">
                  {{ provider.display_name }}
                </td>
                <td class="px-5 py-3 text-gray-600 dark:text-gray-300">{{ provider.group_name || '—' }}</td>
                <td class="px-5 py-3 text-gray-600 dark:text-gray-300">{{ provider.default_model || '—' }}</td>
                <td class="px-5 py-3">
                  <span
                    class="inline-flex rounded-md px-2 py-1 text-xs font-medium"
                    :class="provider.api_key_configured ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300' : 'bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'"
                  >
                    {{ provider.api_key_configured ? `已配置 ${provider.masked_key || ''}` : '未配置凭证' }}
                  </span>
                </td>
                <td class="px-5 py-3">
                  <span
                    class="inline-flex rounded-md px-2 py-1 text-xs font-medium"
                    :class="provider.enabled ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300' : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'"
                  >
                    {{ provider.enabled ? '启用中' : '未启用' }}
                  </span>
                </td>
                <td class="px-5 py-3 text-right">
                  <RouterLink class="btn btn-sm btn-outline" to="/admin/video/providers">
                    <Icon name="edit" size="sm" />
                    去管理
                  </RouterLink>
                </td>
              </tr>
              <tr v-if="!loading && !providers.length">
                <td colspan="6" class="px-5 py-10 text-center text-sm text-gray-500 dark:text-gray-400">
                  暂无视频通道。点右上角「管理视频通道」接入 Seedance。
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <!-- 账号 新增/编辑 弹窗 -->
      <div v-if="accountModalOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="closeAccountModal">
        <div class="w-full max-w-lg rounded-lg border border-gray-200 bg-white p-6 shadow-xl dark:border-dark-700 dark:bg-dark-800">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ editingAccount ? '编辑 AI 账号' : '录入 AI 账号' }}
          </h2>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            只需要填平台、名称和密钥。密钥保存后不再回显，留空表示保留当前密钥。
          </p>
          <form class="mt-5 space-y-4" data-test="account-form" @submit.prevent="saveAccount">
            <div class="grid gap-4 sm:grid-cols-2">
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">平台</label>
                <select v-model="accountForm.platform" class="input" :disabled="!!editingAccount">
                  <option value="anthropic">Claude（Anthropic）</option>
                  <option value="openai">OpenAI</option>
                  <option value="gemini">Gemini</option>
                  <option value="antigravity">Antigravity</option>
                </select>
              </div>
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">名称</label>
                <input v-model="accountForm.name" class="input" maxlength="100" placeholder="例如：老板的 Claude 主账号" required />
              </div>
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">API Key</label>
              <input
                v-model="accountForm.apiKey"
                class="input"
                type="password"
                autocomplete="off"
                maxlength="4000"
                data-test="account-api-key"
                :placeholder="editingAccount ? '留空表示保留当前密钥' : 'sk-...'"
                :required="!editingAccount"
              />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">接口地址（可选）</label>
              <input v-model="accountForm.baseUrl" class="input" maxlength="500" placeholder="留空使用官方默认地址" />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">备注（可选）</label>
              <input v-model="accountForm.notes" class="input" maxlength="200" placeholder="例如：包月订阅，到期 8 月底" />
            </div>
            <div class="flex justify-end gap-2 pt-2">
              <button class="btn btn-outline" type="button" data-test="cancel-account" @click="closeAccountModal">取消</button>
              <button class="btn btn-primary" type="submit" data-test="save-account" :disabled="savingAccount">
                <Icon name="check" size="sm" />
                保存
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type { Account, AccountPlatform } from '@/types'
import type { VideoProviderAccount } from '@/api/admin/video'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { requestConfirmation } from '@/composables/useAppDialog'
import { formatDateTime } from './consoleUtils'

const appStore = useAppStore()
const route = useRoute()

type TabKey = 'accounts' | 'video'
const tabs: Array<{ key: TabKey; label: string }> = [
  { key: 'accounts', label: '通用 AI 账号（文字 / 作图）' },
  { key: 'video', label: '视频通道' },
]
const activeTab = ref<TabKey>('accounts')

const loading = ref(false)
const accounts = ref<Account[]>([])
const accountsTotal = ref(0)
const providers = ref<VideoProviderAccount[]>([])

// ---- 通用 AI 账号 ----

const accountModalOpen = ref(false)
const editingAccount = ref<Account | null>(null)
const savingAccount = ref(false)
const testingAccountId = ref<number | null>(null)

const accountForm = reactive({
  platform: 'anthropic' as AccountPlatform,
  name: '',
  apiKey: '',
  baseUrl: '',
  notes: '',
})

const platformDefaults: Record<string, string> = {
  anthropic: 'https://api.anthropic.com',
  openai: 'https://api.openai.com',
  gemini: 'https://generativelanguage.googleapis.com',
  antigravity: '',
}

function platformLabel(platform: string): string {
  const labels: Record<string, string> = {
    anthropic: 'Claude',
    openai: 'OpenAI',
    gemini: 'Gemini',
    antigravity: 'Antigravity',
  }
  return labels[platform] || platform
}

function platformBadgeClass(platform: string): string {
  const classes: Record<string, string> = {
    anthropic: 'bg-orange-50 text-orange-700 dark:bg-orange-500/10 dark:text-orange-300',
    openai: 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300',
    gemini: 'bg-blue-50 text-blue-700 dark:bg-blue-500/10 dark:text-blue-300',
    antigravity: 'bg-violet-50 text-violet-700 dark:bg-violet-500/10 dark:text-violet-300',
  }
  return classes[platform] || 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-200'
}

function accountTypeLabel(type: string): string {
  const labels: Record<string, string> = {
    apikey: 'API Key',
    oauth: 'OAuth 登录',
    'setup-token': 'Setup Token',
    upstream: '上游中转',
    bedrock: 'AWS Bedrock',
    service_account: '服务账号',
  }
  return labels[type] || type
}

function accountStatusLabel(account: Account): string {
  if (account.status === 'error') return '异常'
  if (account.status === 'inactive') return '已停用'
  return '正常'
}

function accountStatusClass(account: Account): string {
  if (account.status === 'error') return 'bg-red-50 text-red-700 dark:bg-red-500/10 dark:text-red-300'
  if (account.status === 'inactive') return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
  return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
}

function openCreate() {
  editingAccount.value = null
  accountForm.platform = 'anthropic'
  accountForm.name = ''
  accountForm.apiKey = ''
  accountForm.baseUrl = ''
  accountForm.notes = ''
  accountModalOpen.value = true
}

function openEditAccount(account: Account) {
  editingAccount.value = account
  accountForm.platform = account.platform
  accountForm.name = account.name
  accountForm.apiKey = ''
  accountForm.baseUrl = String(account.credentials?.base_url ?? '')
  accountForm.notes = account.notes || ''
  accountModalOpen.value = true
}

function clearAccountSecrets() {
  // 密钥永远不落地在可回显的 reactive 状态里：取消或保存后都要立即清空，
  // 不依赖弹窗卸载时机（v-if 卸载是异步的，明文会在这段时间内留在内存/响应式图里）。
  accountForm.apiKey = ''
}

function closeAccountModal() {
  accountModalOpen.value = false
  editingAccount.value = null
  clearAccountSecrets()
}

async function saveAccount() {
  savingAccount.value = true
  try {
    if (editingAccount.value) {
      const credentials: Record<string, unknown> = { ...(editingAccount.value.credentials || {}) }
      if (accountForm.apiKey.trim()) credentials.api_key = accountForm.apiKey.trim()
      if (accountForm.baseUrl.trim()) {
        credentials.base_url = accountForm.baseUrl.trim()
      } else if (platformDefaults[accountForm.platform]) {
        credentials.base_url = credentials.base_url || platformDefaults[accountForm.platform]
      }
      await adminAPI.accounts.update(editingAccount.value.id, {
        name: accountForm.name.trim(),
        notes: accountForm.notes.trim() || null,
        credentials,
      })
      appStore.showSuccess('账号已更新')
    } else {
      const baseUrl = accountForm.baseUrl.trim() || platformDefaults[accountForm.platform]
      const credentials: Record<string, unknown> = { api_key: accountForm.apiKey.trim() }
      if (baseUrl) credentials.base_url = baseUrl
      await adminAPI.accounts.create({
        name: accountForm.name.trim(),
        notes: accountForm.notes.trim() || null,
        platform: accountForm.platform,
        type: 'apikey',
        credentials,
      })
      appStore.showSuccess('账号已录入密钥库')
    }
    closeAccountModal()
    await loadAccounts()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '保存账号失败'))
  } finally {
    savingAccount.value = false
  }
}

async function toggleAccountStatus(account: Account) {
  try {
    const next = account.status === 'active' ? 'inactive' : 'active'
    await adminAPI.accounts.toggleStatus(account.id, next)
    appStore.showSuccess(next === 'active' ? '账号已启用' : '账号已停用')
    await loadAccounts()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '切换账号状态失败'))
  }
}

async function removeAccount(account: Account) {
  const confirmed = await requestConfirmation({
    message: `确定删除账号「${account.name}」？删除后走该账号的调用会失败。`,
    danger: true,
  })
  if (!confirmed) return
  try {
    await adminAPI.accounts.delete(account.id)
    appStore.showSuccess('账号已删除')
    await loadAccounts()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '删除账号失败'))
  }
}

async function testAccount(account: Account) {
  testingAccountId.value = account.id
  try {
    const result = await adminAPI.accounts.testAccount(account.id)
    if (result.success) {
      appStore.showSuccess(`「${account.name}」连通正常${result.latency_ms ? `（${result.latency_ms}ms）` : ''}`)
    } else {
      appStore.showError(`「${account.name}」测试失败：${result.message}`)
    }
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '账号连通性测试失败'))
  } finally {
    testingAccountId.value = null
  }
}

async function loadAccounts() {
  const res = await adminAPI.accounts.list(1, 100)
  accounts.value = res.items || []
  accountsTotal.value = res.total || accounts.value.length
}

// ---- 视频通道（只读摘要；创建/编辑/授权统一在 /admin/video/providers 完成） ----

async function loadProviders() {
  const res = await adminAPI.video.listProviders()
  providers.value = res.items || []
}

// ---- 汇总加载 ----

async function reload() {
  loading.value = true
  try {
    await Promise.all([loadAccounts(), loadProviders()])
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载密钥库失败'))
  } finally {
    loading.value = false
  }
}

function syncTabFromRoute() {
  if (route.query.tab === 'video') {
    activeTab.value = 'video'
  }
}

watch(() => route.query.tab, syncTabFromRoute)

onMounted(() => {
  syncTabFromRoute()
  void reload()
})
</script>
