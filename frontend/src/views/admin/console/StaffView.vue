<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 border-b border-gray-200 pb-4 dark:border-dark-700 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">成员与开卡</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            员工和外部工具（n8n、脚本、批量出图器）都在这里建账号、开卡。卡强绑定到成员：谁调了什么、花了多少钱，互不污染。
          </p>
        </div>
        <div class="flex flex-wrap gap-2">
          <button class="btn btn-outline" type="button" :disabled="loading" @click="loadStaff">
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
            刷新
          </button>
          <button class="btn btn-outline" type="button" @click="openCreateStaff('tool')">
            <Icon name="key" size="sm" />
            新增工具
          </button>
          <button class="btn btn-primary" type="button" @click="openCreateStaff('human')">
            <Icon name="userPlus" size="sm" />
            新增员工
          </button>
        </div>
      </div>

      <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="flex flex-col gap-3 border-b border-gray-200 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex flex-1 flex-wrap items-center gap-2">
            <input
              v-model="search"
              class="input sm:max-w-xs"
              placeholder="按姓名或邮箱搜索"
              @keyup.enter="loadStaff"
            />
            <div class="flex rounded-lg border border-gray-200 p-0.5 text-xs dark:border-dark-700">
              <button
                v-for="opt in memberFilters"
                :key="opt.value"
                type="button"
                class="rounded-md px-2.5 py-1 font-medium transition-colors"
                :class="memberFilter === opt.value ? 'bg-teal-600 text-white' : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700'"
                @click="memberFilter = opt.value"
              >
                {{ opt.label }}
              </button>
            </div>
          </div>
          <div class="text-xs text-gray-500 dark:text-gray-400">共 {{ filteredUsers.length }} 名成员（含管理员）</div>
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs uppercase text-gray-500 dark:bg-dark-700/40 dark:text-gray-400">
              <tr>
                <th class="px-5 py-3 font-medium">员工</th>
                <th class="px-5 py-3 font-medium">备注</th>
                <th class="px-5 py-3 font-medium">状态</th>
                <th class="px-5 py-3 font-medium">今日花费</th>
                <th class="px-5 py-3 font-medium">累计花费</th>
                <th class="px-5 py-3 text-right font-medium">操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <template v-for="user in filteredUsers" :key="user.id">
                <tr class="cursor-pointer hover:bg-gray-50 dark:hover:bg-dark-700/40" @click="toggleExpand(user)">
                  <td class="px-5 py-3">
                    <div class="flex items-center gap-3">
                      <span
                        class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-sm font-semibold"
                        :class="user.member_type === 'tool' ? 'bg-sky-100 text-sky-700 dark:bg-sky-500/20 dark:text-sky-200' : 'bg-teal-100 text-teal-700 dark:bg-teal-500/20 dark:text-teal-200'"
                      >
                        {{ staffDisplayName(user.username, user.email).slice(0, 1).toUpperCase() }}
                      </span>
                      <div class="min-w-0">
                        <div class="flex items-center gap-2 font-medium text-gray-900 dark:text-white">
                          {{ staffDisplayName(user.username, user.email) }}
                          <span v-if="user.member_type === 'tool'" class="rounded bg-sky-100 px-1.5 py-0.5 text-[10px] font-medium text-sky-700 dark:bg-sky-500/20 dark:text-sky-200">工具</span>
                          <span v-if="user.role === 'admin'" class="rounded bg-violet-100 px-1.5 py-0.5 text-[10px] font-medium text-violet-700 dark:bg-violet-500/20 dark:text-violet-200">管理员</span>
                        </div>
                        <div class="truncate text-xs text-gray-500 dark:text-gray-400">{{ user.email }}</div>
                      </div>
                    </div>
                  </td>
                  <td class="max-w-[180px] truncate px-5 py-3 text-xs text-gray-500 dark:text-gray-400" :title="user.notes || ''">{{ user.notes || '—' }}</td>
                  <td class="px-5 py-3">
                    <span
                      class="inline-flex rounded-md px-2 py-1 text-xs font-medium"
                      :class="user.status === 'active' ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300' : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'"
                    >
                      {{ user.status === 'active' ? '在用' : '已停用' }}
                    </span>
                  </td>
                  <td class="px-5 py-3 tabular-nums text-gray-700 dark:text-gray-200">{{ formatMoney(usageMap[user.id]?.today_actual_cost, usdCnyRate) }}</td>
                  <td class="px-5 py-3 tabular-nums font-medium text-teal-600 dark:text-teal-300">{{ formatMoney(usageMap[user.id]?.total_actual_cost, usdCnyRate) }}</td>
                  <td class="px-5 py-3">
                    <div class="flex justify-end gap-1.5" @click.stop>
                      <button class="btn btn-sm btn-primary" type="button" @click="openIssueCard(user)">
                        <Icon name="key" size="sm" />
                        开卡
                      </button>
                      <button class="btn btn-sm btn-outline" type="button" @click="toggleExpand(user)">
                        {{ expandedUserId === user.id ? '收起' : `查看卡片（${expandedUserId === user.id ? userKeys.length : '…'}）` }}
                      </button>
                    </div>
                  </td>
                </tr>
                <!-- 展开：员工的 API 卡 -->
                <tr v-if="expandedUserId === user.id">
                  <td colspan="6" class="bg-gray-50/60 px-5 py-4 dark:bg-dark-900/40">
                    <div v-if="keysLoading" class="py-4 text-center text-sm text-gray-500 dark:text-gray-400">加载卡片中…</div>
                    <template v-else>
                      <div v-if="!userKeys.length" class="py-4 text-center text-sm text-gray-500 dark:text-gray-400">
                        这名员工还没有卡。点上面的「开卡」发一张。
                      </div>
                      <table v-else class="min-w-full text-sm">
                        <thead class="text-left text-xs uppercase text-gray-500 dark:text-gray-400">
                          <tr>
                            <th class="px-3 py-2 font-medium">卡名</th>
                            <th class="px-3 py-2 font-medium">Key（脱敏）</th>
                            <th class="px-3 py-2 font-medium">状态</th>
                            <th class="px-3 py-2 font-medium">额度</th>
                            <th class="px-3 py-2 font-medium">卡上花费</th>
                            <th class="px-3 py-2 font-medium">最近使用</th>
                            <th class="px-3 py-2 text-right font-medium">操作</th>
                          </tr>
                        </thead>
                        <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                          <tr v-for="key in userKeys" :key="key.id">
                            <td class="px-3 py-2 font-medium text-gray-900 dark:text-white">{{ key.name || '未命名' }}</td>
                            <td class="px-3 py-2 font-mono text-xs text-gray-500 dark:text-gray-400">{{ maskKey(key.key) }}</td>
                            <td class="px-3 py-2">
                              <span class="inline-flex rounded-md px-2 py-0.5 text-xs font-medium" :class="keyStatusClass(key.status)">
                                {{ keyStatusLabel(key.status) }}
                              </span>
                            </td>
                            <td class="px-3 py-2 text-xs tabular-nums text-gray-600 dark:text-gray-300">
                              {{ key.quota > 0 ? `${formatAccountUsd(key.quota_used)} / ${formatAccountUsd(key.quota)}` : '不限额' }}
                            </td>
                            <td class="px-3 py-2 text-xs tabular-nums text-teal-600 dark:text-teal-300">{{ formatMoney(keyUsageMap[key.id]?.total_actual_cost, usdCnyRate) }}</td>
                            <td class="px-3 py-2 text-xs text-gray-500 dark:text-gray-400">{{ formatDateTime(key.last_used_at) }}</td>
                            <td class="px-3 py-2">
                              <div class="flex justify-end gap-1.5">
                                <button
                                  class="btn btn-sm btn-outline"
                                  type="button"
                                  :disabled="keyActionBusy"
                                  @click="toggleKeyStatus(key)"
                                >
                                  {{ key.status === 'active' ? '停用' : '启用' }}
                                </button>
                                <button
                                  class="btn btn-sm btn-outline !text-red-600 dark:!text-red-400"
                                  type="button"
                                  :disabled="keyActionBusy"
                                  @click="removeKey(key)"
                                >
                                  <Icon name="trash" size="sm" />
                                </button>
                              </div>
                            </td>
                          </tr>
                        </tbody>
                      </table>
                    </template>
                  </td>
                </tr>
              </template>
              <tr v-if="!loading && !filteredUsers.length">
                <td colspan="6" class="px-5 py-10 text-center text-sm text-gray-500 dark:text-gray-400">
                  {{ memberFilter === 'tool' ? '还没有工具账号。点右上角「新增工具」给 n8n、脚本等建档。' : '还没有成员。点右上角「新增员工」开始建档。' }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <!-- 新增成员弹窗 -->
      <div v-if="staffModalOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="staffModalOpen = false">
        <div class="w-full max-w-md rounded-lg border border-gray-200 bg-white p-6 shadow-xl dark:border-dark-700 dark:bg-dark-800">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ staffForm.memberType === 'tool' ? '新增工具' : '新增员工' }}</h2>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ staffForm.memberType === 'tool'
              ? '给 n8n、剪辑脚本、批量出图器等工具建一个独立账号，消费不会算到任何人头上。'
              : '建好档案后再给员工开卡。登录邮箱和初始密码交给员工本人。' }}
          </p>
          <form class="mt-5 space-y-4" @submit.prevent="createStaff">
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">类型</label>
              <div class="flex rounded-lg border border-gray-200 p-0.5 text-sm dark:border-dark-700">
                <button
                  type="button"
                  class="flex-1 rounded-md px-3 py-1.5 font-medium transition-colors"
                  :class="staffForm.memberType === 'human' ? 'bg-teal-600 text-white' : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700'"
                  @click="staffForm.memberType = 'human'"
                >
                  员工
                </button>
                <button
                  type="button"
                  class="flex-1 rounded-md px-3 py-1.5 font-medium transition-colors"
                  :class="staffForm.memberType === 'tool' ? 'bg-sky-600 text-white' : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700'"
                  @click="staffForm.memberType = 'tool'"
                >
                  外部工具
                </button>
              </div>
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ staffForm.memberType === 'tool' ? '工具名称' : '姓名' }}</label>
              <input v-model="staffForm.username" class="input" maxlength="50" :placeholder="staffForm.memberType === 'tool' ? '例如：n8n 自动出图' : '例如：张三'" />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">登录邮箱</label>
              <input v-model="staffForm.email" class="input" type="email" required :placeholder="staffForm.memberType === 'tool' ? 'n8n@wujie.local（占位邮箱即可）' : 'zhangsan@company.com'" />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">初始密码</label>
              <input v-model="staffForm.password" class="input" type="password" autocomplete="new-password" required minlength="6" placeholder="至少 6 位" />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">备注（可选）</label>
              <input v-model="staffForm.notes" class="input" maxlength="200" :placeholder="staffForm.memberType === 'tool' ? '例如：夜间批量分镜流程' : '例如：短剧组剪辑'" />
            </div>
            <div class="flex justify-end gap-2 pt-2">
              <button class="btn btn-outline" type="button" @click="staffModalOpen = false">取消</button>
              <button class="btn btn-primary" type="submit" :disabled="creatingStaff">
                <Icon name="check" size="sm" />
                创建
              </button>
            </div>
          </form>
        </div>
      </div>

      <!-- 开卡弹窗 -->
      <div v-if="issueModalOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="closeIssueModal">
        <div class="w-full max-w-md rounded-lg border border-gray-200 bg-white p-6 shadow-xl dark:border-dark-700 dark:bg-dark-800">
          <template v-if="!issuedKey">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              给 {{ staffDisplayName(issueTarget?.username, issueTarget?.email) }} 开卡
            </h2>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">这张卡强绑定该员工，卡上的每次调用都会记到这个人头上。</p>
            <form class="mt-5 space-y-4" @submit.prevent="issueCard">
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">卡名</label>
                <input v-model="issueForm.name" class="input" maxlength="100" required placeholder="例如：张三-短剧生产卡" />
              </div>
              <div class="grid gap-4 sm:grid-cols-2">
                <div>
                  <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">额度（美元）</label>
                  <input v-model.number="issueForm.quota" class="input" type="number" min="0" step="0.01" placeholder="0 = 不限额" />
                  <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">约 {{ formatMoney(issueForm.quota, usdCnyRate) }}；实际额度按 USD 存储</p>
                </div>
                <div>
                  <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">有效期（天）</label>
                  <input v-model.number="issueForm.expiresInDays" class="input" type="number" min="0" step="1" placeholder="0 = 长期有效" />
                </div>
              </div>
              <div class="flex justify-end gap-2 pt-2">
                <button class="btn btn-outline" type="button" @click="closeIssueModal">取消</button>
                <button class="btn btn-primary" type="submit" :disabled="issuing">
                  <Icon name="key" size="sm" />
                  开卡
                </button>
              </div>
            </form>
          </template>
          <template v-else>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">开卡成功</h2>
            <p class="mt-1 text-xs text-amber-600 dark:text-amber-300">
              完整 Key 只显示这一次，请立即复制并交给员工保管。关掉这个窗口后只能看到脱敏后的 Key。
            </p>
            <div class="mt-4 rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-900">
              <div class="break-all font-mono text-sm text-gray-900 dark:text-white">{{ issuedKey.key }}</div>
            </div>
            <div class="mt-4 flex justify-end gap-2">
              <button class="btn btn-primary" type="button" @click="copyIssuedKey">
                <Icon name="copy" size="sm" />
                {{ copied ? '已复制' : '复制 Key' }}
              </button>
              <button class="btn btn-outline" type="button" @click="closeIssueModal">完成</button>
            </div>
          </template>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type { AdminUser, ApiKey } from '@/types'
import type { BatchApiKeyUsageStats, BatchUserUsageStats } from '@/api/admin/dashboard'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatAccountUsd, formatDateTime, formatMoney, staffDisplayName } from './consoleUtils'

const appStore = useAppStore()

const loading = ref(false)
const search = ref('')
const users = ref<AdminUser[]>([])
const total = ref(0)
const usageMap = ref<Record<number, BatchUserUsageStats>>({})
const usdCnyRate = ref(7.2)

// ---- 成员类型筛选 ----

type MemberFilter = 'all' | 'human' | 'tool'
const memberFilter = ref<MemberFilter>('all')
const memberFilters: Array<{ value: MemberFilter; label: string }> = [
  { value: 'all', label: '全部' },
  { value: 'human', label: '员工' },
  { value: 'tool', label: '工具' },
]

const filteredUsers = computed(() => {
  if (memberFilter.value === 'all') return users.value
  return users.value.filter((user) => (user.member_type ?? 'human') === memberFilter.value)
})

// ---- 员工列表 ----

async function loadStaff() {
  loading.value = true
  try {
    const [res, rateStats] = await Promise.all([
      adminAPI.users.list(1, 100, {
        search: search.value.trim() || undefined,
        sort_by: 'created_at',
        sort_order: 'asc',
      }),
      adminAPI.dashboard.getStats().catch(() => null),
    ])
    usdCnyRate.value = Number(rateStats?.usd_cny_rate || 7.2)
    users.value = res.items || []
    total.value = res.total || users.value.length
    if (users.value.length) {
      const usage = await adminAPI.dashboard.getBatchUsersUsage(users.value.map((user) => user.id))
      const map: Record<number, BatchUserUsageStats> = {}
      for (const [id, stats] of Object.entries(usage.stats || {})) {
        map[Number(id)] = stats
      }
      usageMap.value = map
    } else {
      usageMap.value = {}
    }
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载员工列表失败'))
  } finally {
    loading.value = false
  }
}

// ---- 新增员工 ----

const staffModalOpen = ref(false)
const creatingStaff = ref(false)
const staffForm = reactive({
  username: '',
  email: '',
  password: '',
  notes: '',
  memberType: 'human' as 'human' | 'tool',
})

function openCreateStaff(memberType: 'human' | 'tool' = 'human') {
  staffForm.username = ''
  staffForm.email = ''
  staffForm.password = ''
  staffForm.notes = ''
  staffForm.memberType = memberType
  staffModalOpen.value = true
}

async function createStaff() {
  creatingStaff.value = true
  try {
    await adminAPI.users.create({
      email: staffForm.email.trim(),
      password: staffForm.password,
      username: staffForm.username.trim() || undefined,
      notes: staffForm.notes.trim() || undefined,
      member_type: staffForm.memberType,
    })
    appStore.showSuccess(staffForm.memberType === 'tool' ? '工具账号已创建，接下来可以开卡' : '员工已创建，接下来可以开卡')
    staffModalOpen.value = false
    await loadStaff()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, staffForm.memberType === 'tool' ? '创建工具账号失败' : '创建员工失败'))
  } finally {
    creatingStaff.value = false
  }
}

// ---- 展开员工的卡 ----

const expandedUserId = ref<number | null>(null)
const userKeys = ref<ApiKey[]>([])
const keysLoading = ref(false)
const keyUsageMap = ref<Record<number, BatchApiKeyUsageStats>>({})

async function toggleExpand(user: AdminUser) {
  if (expandedUserId.value === user.id) {
    expandedUserId.value = null
    return
  }
  expandedUserId.value = user.id
  await loadUserKeys(user.id)
}

async function loadUserKeys(userId: number) {
  keysLoading.value = true
  try {
    const res = await adminAPI.users.getUserApiKeys(userId)
    userKeys.value = res.items || []
    if (userKeys.value.length) {
      const usage = await adminAPI.dashboard.getBatchApiKeysUsage(userKeys.value.map((key) => key.id))
      const map: Record<number, BatchApiKeyUsageStats> = {}
      for (const [id, stats] of Object.entries(usage.stats || {})) {
        map[Number(id)] = stats
      }
      keyUsageMap.value = map
    } else {
      keyUsageMap.value = {}
    }
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载员工卡片失败'))
  } finally {
    keysLoading.value = false
  }
}

function maskKey(key: string): string {
  if (!key) return '—'
  if (key.length <= 12) return `${key.slice(0, 4)}…`
  return `${key.slice(0, 8)}…${key.slice(-4)}`
}

function keyStatusLabel(status: ApiKey['status']): string {
  const labels: Record<ApiKey['status'], string> = {
    active: '在用',
    inactive: '已停用',
    disabled: '已停用',
    quota_exhausted: '额度用完',
    expired: '已过期',
  }
  return labels[status] ?? status
}

function keyStatusClass(status: ApiKey['status']): string {
  const classes: Record<ApiKey['status'], string> = {
    active: 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300',
    inactive: 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300',
    disabled: 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300',
    quota_exhausted: 'bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300',
    expired: 'bg-red-50 text-red-700 dark:bg-red-500/10 dark:text-red-300',
  }
  return classes[status] ?? classes.disabled
}

// ---- 卡操作 ----

const keyActionBusy = ref(false)

async function toggleKeyStatus(key: ApiKey) {
  keyActionBusy.value = true
  try {
    // admin 契约只接受 active/disabled（不要发 inactive，后端会 400）
    const next = key.status === 'active' ? 'disabled' : 'active'
    await adminAPI.apiKeys.updateApiKey(key.id, { status: next })
    appStore.showSuccess(next === 'active' ? '卡已启用' : '卡已停用')
    if (expandedUserId.value) await loadUserKeys(expandedUserId.value)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '切换卡状态失败'))
  } finally {
    keyActionBusy.value = false
  }
}

async function removeKey(key: ApiKey) {
  if (!window.confirm(`确定删除卡「${key.name || maskKey(key.key)}」？删除后该卡立刻失效。`)) return
  keyActionBusy.value = true
  try {
    await adminAPI.apiKeys.deleteApiKey(key.id)
    appStore.showSuccess('卡已删除')
    if (expandedUserId.value) await loadUserKeys(expandedUserId.value)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '删除卡失败'))
  } finally {
    keyActionBusy.value = false
  }
}

// ---- 开卡 ----

const issueModalOpen = ref(false)
const issueTarget = ref<AdminUser | null>(null)
const issuing = ref(false)
const issuedKey = ref<ApiKey | null>(null)
const copied = ref(false)
const issueForm = reactive({ name: '', quota: 0, expiresInDays: 0 })

function openIssueCard(user: AdminUser) {
  issueTarget.value = user
  issuedKey.value = null
  copied.value = false
  issueForm.name = `${staffDisplayName(user.username, user.email)}-生产卡`
  issueForm.quota = 0
  issueForm.expiresInDays = 0
  issueModalOpen.value = true
}

function closeIssueModal() {
  issueModalOpen.value = false
  issueTarget.value = null
  issuedKey.value = null
}

async function issueCard() {
  if (!issueTarget.value) return
  issuing.value = true
  try {
    issuedKey.value = await adminAPI.apiKeys.createApiKeyForUser(issueTarget.value.id, {
      name: issueForm.name.trim(),
      quota: issueForm.quota > 0 ? issueForm.quota : 0,
      ...(issueForm.expiresInDays > 0 ? { expires_in_days: issueForm.expiresInDays } : {}),
    })
    appStore.showSuccess('开卡成功')
    if (expandedUserId.value === issueTarget.value.id) {
      await loadUserKeys(issueTarget.value.id)
    }
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '开卡失败'))
  } finally {
    issuing.value = false
  }
}

async function copyIssuedKey() {
  if (!issuedKey.value) return
  try {
    await navigator.clipboard.writeText(issuedKey.value.key)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch {
    appStore.showError('复制失败，请手动选中复制')
  }
}

onMounted(loadStaff)
</script>
