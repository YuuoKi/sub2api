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
                <th class="px-5 py-3 font-medium">分组</th>
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
                <td class="px-5 py-3">
                  <AccountGroupsCell :groups="account.groups" />
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
                <td colspan="8" class="px-5 py-10 text-center text-sm text-gray-500 dark:text-gray-400">
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
              <GroupSelector
                v-model="accountForm.groupIds"
                :groups="groups"
                :platform="accountForm.platform"
                data-test="account-groups"
              />
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                必选，至少一个分组。账号的调用按分组计费与路由（例如作图走 media 组）。
              </p>
              <div
                v-if="!platformGroups.length"
                class="mt-2 rounded-md border border-dashed border-gray-300 p-3 text-xs dark:border-dark-600"
                data-test="group-quick-create"
              >
                <p class="text-gray-600 dark:text-gray-300">当前平台还没有分组，先建一个（也可到「模型分组」页精细调整）：</p>
                <div class="mt-2 flex flex-wrap items-center gap-2">
                  <button class="btn btn-sm btn-outline" type="button" :disabled="creatingGroup" data-test="quick-create-media" @click="quickCreateGroup('media')">
                    作图组 media
                  </button>
                  <button class="btn btn-sm btn-outline" type="button" :disabled="creatingGroup" data-test="quick-create-video" @click="quickCreateGroup('video')">
                    视频组 video
                  </button>
                  <input
                    v-model="quickGroupName"
                    class="input !w-40 !py-1 text-xs"
                    maxlength="50"
                    placeholder="或输入分组名"
                    data-test="quick-create-name"
                  />
                  <button
                    class="btn btn-sm btn-primary"
                    type="button"
                    :disabled="creatingGroup || !quickGroupName.trim()"
                    data-test="quick-create-custom"
                    @click="quickCreateGroup(quickGroupName)"
                  >
                    {{ creatingGroup ? '创建中…' : '创建并选中' }}
                  </button>
                </div>
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
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import GroupSelector from '@/components/common/GroupSelector.vue'
import AccountGroupsCell from '@/components/account/AccountGroupsCell.vue'
import { adminAPI } from '@/api/admin'
import type { Account, AccountPlatform, AdminGroup } from '@/types'
import type { VideoProviderAccount } from '@/api/admin/video'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { requestConfirmation } from '@/composables/useAppDialog'
import { CONSOLE_ERROR_ZH, formatDateTime } from './consoleUtils'

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
const groups = ref<AdminGroup[]>([])

// ---- 通用 AI 账号 ----

const accountModalOpen = ref(false)
const editingAccount = ref<Account | null>(null)
const savingAccount = ref(false)

const accountForm = reactive({
  platform: 'anthropic' as AccountPlatform,
  name: '',
  apiKey: '',
  baseUrl: '',
  notes: '',
  groupIds: [] as number[],
})

// 分组必选：当前平台可选的分组；切平台时清掉不再匹配的已选 id
const platformGroups = computed(() => groups.value.filter((g) => g.platform === accountForm.platform))
const creatingGroup = ref(false)
const quickGroupName = ref('')

watch(
  () => accountForm.platform,
  () => {
    const valid = new Set(platformGroups.value.map((g) => g.id))
    accountForm.groupIds = accountForm.groupIds.filter((id) => valid.has(id))
  },
)

async function quickCreateGroup(name: string) {
  const trimmed = name.trim()
  if (!trimmed || creatingGroup.value) return
  creatingGroup.value = true
  try {
    // 后端建组契约要求全量字段（缺失会 500）；与 GroupsView 默认表单对齐，按模板名预置媒体开关
    const created = await adminAPI.groups.create({
      name: trimmed,
      description: '',
      platform: accountForm.platform,
      rate_multiplier: 1,
      is_exclusive: false,
      subscription_type: 'standard',
      daily_limit_usd: null,
      weekly_limit_usd: null,
      monthly_limit_usd: null,
      allow_image_generation: trimmed === 'media',
      allow_batch_image_generation: false,
      image_rate_independent: false,
      image_rate_multiplier: 1,
      batch_image_discount_multiplier: 0.5,
      batch_image_hold_multiplier: 0.6,
      image_price_1k: null,
      image_price_2k: null,
      image_price_4k: null,
      video_rate_independent: false,
      video_rate_multiplier: 1,
      video_price_480p: null,
      video_price_720p: null,
      video_price_1080p: null,
      peak_rate_enabled: false,
      peak_start: '',
      peak_end: '',
      peak_rate_multiplier: 1,
      claude_code_only: false,
      fallback_group_id: null,
      fallback_group_id_on_invalid_request: null,
      allow_messages_dispatch: false,
      require_oauth_only: false,
      require_privacy_set: false,
      model_routing_enabled: false,
      supported_model_scopes: ['claude', 'gemini_text', 'gemini_image'],
      mcp_xml_inject: true,
      copy_accounts_from_group_ids: [],
      rpm_limit: 0,
    })
    await loadGroups()
    if (!accountForm.groupIds.includes(created.id)) {
      accountForm.groupIds = [...accountForm.groupIds, created.id]
    }
    quickGroupName.value = ''
    appStore.showSuccess(`分组「${created.name}」已创建并选中`)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '创建分组失败', CONSOLE_ERROR_ZH))
  } finally {
    creatingGroup.value = false
  }
}

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
  accountForm.groupIds = []
  quickGroupName.value = ''
  accountModalOpen.value = true
}

function openEditAccount(account: Account) {
  editingAccount.value = account
  accountForm.platform = account.platform
  accountForm.name = account.name
  accountForm.apiKey = ''
  accountForm.baseUrl = String(account.credentials?.base_url ?? '')
  accountForm.notes = account.notes || ''
  // 回填已绑定分组；老数据若没有 group_ids 字段则用预加载的 groups 兜底
  accountForm.groupIds = [...(account.group_ids ?? account.groups?.map((g) => g.id) ?? [])]
  quickGroupName.value = ''
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
  // 分组必选：未选组直接拦截并显式报错，避免账号落成「无组孤儿」（计费/路由都走不通）
  if (accountForm.groupIds.length === 0) {
    appStore.showError('请至少选择一个分组；没有可用分组时先点下方「作图组 media / 视频组 video」快速创建')
    return
  }
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
        group_ids: [...accountForm.groupIds],
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
        group_ids: [...accountForm.groupIds],
      })
      appStore.showSuccess('账号已录入密钥库')
    }
    closeAccountModal()
    await loadAccounts()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '保存账号失败', CONSOLE_ERROR_ZH))
  } finally {
    savingAccount.value = false
  }
}

async function toggleAccountStatus(account: Account) {
  const next = account.status === 'active' ? 'inactive' : 'active'
  if (next === 'inactive') {
    const confirmed = await requestConfirmation({
      message: `确定停用账号「${account.name}」？停用后走该账号的调用会立即失败。`,
      danger: true,
    })
    if (!confirmed) return
  }
  try {
    await adminAPI.accounts.toggleStatus(account.id, next)
    appStore.showSuccess(next === 'active' ? '账号已启用' : '账号已停用')
    await loadAccounts()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '切换账号状态失败', CONSOLE_ERROR_ZH))
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
    appStore.showError(extractApiErrorMessage(err, '删除账号失败', CONSOLE_ERROR_ZH))
  }
}

async function loadAccounts() {
  const res = await adminAPI.accounts.list(1, 100)
  accounts.value = res.items || []
  accountsTotal.value = res.total || accounts.value.length
}

async function loadGroups() {
  groups.value = await adminAPI.groups.getAll()
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
    await Promise.all([loadAccounts(), loadProviders(), loadGroups()])
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载密钥库失败', CONSOLE_ERROR_ZH))
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
