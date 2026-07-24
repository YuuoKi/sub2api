<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 border-b border-gray-200 pb-4 dark:border-dark-700 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 class="ui-heading">员工与开卡</h1>
          <p class="ui-subheading mt-1">
            给员工开 API 卡：建账号、选分组、充值，一次办完。
          </p>
        </div>
        <div class="flex flex-wrap gap-2">
          <button class="btn btn-outline" type="button" :disabled="loading" @click="loadStaff">
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
            刷新
          </button>
          <button class="btn btn-primary" type="button" data-test="create-service-identity" @click="openCreateStaff">
            <Icon name="key" size="sm" />
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
          </div>
          <div class="text-xs text-gray-500 dark:text-gray-400">共 {{ filteredUsers.length }} 个员工</div>
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs uppercase text-gray-500 dark:bg-dark-700/40 dark:text-gray-400">
              <tr>
                <th class="px-5 py-3 font-medium">成员</th>
                <th class="px-5 py-3 font-medium">状态</th>
                <th class="px-5 py-3 font-medium">今日花费</th>
                <th class="px-5 py-3 font-medium">累计花费</th>
                <th class="px-5 py-3 text-right font-medium">操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <template v-for="user in filteredUsers" :key="user.id">
                <tr class="cursor-pointer hover:bg-gray-50 dark:hover:bg-dark-700/40" data-test="toggle-expand" @click="toggleExpand(user)">
                  <td class="px-5 py-3">
                    <div class="flex items-center gap-3">
                      <span
                        class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-gray-100 text-sm font-semibold text-gray-600 dark:bg-dark-700 dark:text-gray-300"
                      >
                        {{ staffDisplayName(user.username, user.email).slice(0, 1).toUpperCase() }}
                      </span>
                      <div class="min-w-0">
                        <div class="flex items-center gap-2 font-medium text-gray-900 dark:text-white">
                          {{ staffDisplayName(user.username, user.email) }}
                          <span v-if="user.role === 'admin'" class="text-xs text-gray-400 dark:text-gray-500">管理员</span>
                        </div>
                        <div class="truncate text-xs text-gray-500 dark:text-gray-400">{{ user.email }}</div>
                      </div>
                    </div>
                  </td>
                  <td class="px-5 py-3">
                    <span
                      class="inline-flex rounded-md px-2 py-1 text-xs font-medium"
                      :class="user.status === 'active' ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300' : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'"
                    >
                      {{ user.status === 'active' ? '在用' : '已停用' }}
                    </span>
                  </td>
                  <td class="px-5 py-3 tabular-nums text-gray-700 dark:text-gray-200">{{ formatMoney(usageMap[user.id]?.today_actual_cost, usdCnyRate) }}</td>
                  <td class="px-5 py-3 tabular-nums text-gray-900 dark:text-white">{{ formatMoney(usageMap[user.id]?.total_actual_cost, usdCnyRate) }}</td>
                  <td class="px-5 py-3">
                    <div class="flex justify-end gap-1.5" @click.stop>
                      <button class="btn btn-sm btn-outline" type="button" data-test="row-recharge" @click="openRecharge(user)">
                        充值
                      </button>
                      <RouterLink class="btn btn-sm btn-outline" :to="`/admin/console/ai-records?user_id=${user.id}`" data-test="row-view-spend">
                        查看花费
                      </RouterLink>
                      <button class="btn btn-sm btn-outline" type="button" @click="toggleExpand(user)">
                        {{ expandedUserId === user.id ? '收起' : '查看卡片' }}
                      </button>
                    </div>
                  </td>
                </tr>
                <!-- 展开：员工的 API 卡 -->
                <tr v-if="expandedUserId === user.id">
                  <td colspan="5" class="bg-gray-50/60 px-5 py-4 dark:bg-dark-900/40">
                    <div v-if="keysLoading" class="py-4 text-center text-sm text-gray-500 dark:text-gray-400">加载卡片中…</div>
                    <template v-else>
                      <div v-if="!userKeys.length" class="py-4 text-center text-sm text-gray-500 dark:text-gray-400">
                        这名员工还没有卡。点右上角「新增员工」，用同一邮箱提交即可补开。
                      </div>
                      <div v-else class="overflow-x-auto">
                      <table class="min-w-full text-sm">
                        <thead class="text-left text-xs uppercase text-gray-500 dark:text-gray-400">
                          <tr>
                            <th class="px-3 py-2 font-medium">卡名</th>
                            <th class="px-3 py-2 font-medium">Key</th>
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
                            <td class="px-3 py-2 font-mono text-xs text-gray-500 dark:text-gray-400">{{ maskKey() }}</td>
                            <td class="px-3 py-2">
                              <span class="inline-flex rounded-md px-2 py-0.5 text-xs font-medium" :class="keyStatusClass(key.status)">
                                {{ keyStatusLabel(key.status) }}
                              </span>
                            </td>
                            <td class="px-3 py-2 text-xs tabular-nums" :class="quotaWarningTextClass(quotaWarningLevel(key.quota_used, key.quota))">
                              <div>{{ key.quota > 0 ? `${formatAccountUsd(key.quota_used)} / ${formatAccountUsd(key.quota)}` : '不限额' }}</div>
                              <div
                                v-if="key.quota > 0"
                                class="mt-1 h-1.5 w-24 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600"
                              >
                                <div
                                  class="h-full rounded-full transition-all"
                                  :class="quotaWarningBarClass(quotaWarningLevel(key.quota_used, key.quota))"
                                  :style="{ width: `${Math.min(((key.quota_used || 0) / key.quota) * 100, 100)}%` }"
                                />
                              </div>
                            </td>
                            <td class="px-3 py-2 text-xs tabular-nums text-gray-900 dark:text-white">{{ formatMoney(keyUsageMap[key.id]?.total_actual_cost, usdCnyRate) }}</td>
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
                      </div>
                    </template>
                  </td>
                </tr>
              </template>
              <tr v-if="!loading && !filteredUsers.length">
                <td colspan="5" class="px-5 py-10 text-center text-sm text-gray-500 dark:text-gray-400">
                  还没有员工。点右上角「新增员工」开第一张卡。
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <!-- 新增员工：一个弹窗办完（建账号 → 开双 Key → 充值 → 明文展示一次） -->
      <div v-if="staffModalOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="closeStaffModal">
        <div class="w-full max-w-md rounded-lg border border-gray-200 bg-white p-6 shadow-xl dark:border-dark-700 dark:bg-dark-800">
          <template v-if="!issuedQCanvasPair">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">新增员工</h2>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              提交后自动建好账号、开出视频 / LLM 两把 Key 并按金额充值，明文 Key 只显示一次。
            </p>
            <form class="mt-5 space-y-4" data-test="service-identity-form" @submit.prevent="submitStaff">
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">员工姓名</label>
                <input v-model="staffForm.username" class="input" maxlength="50" placeholder="例如：张三" />
              </div>
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">邮箱</label>
                <input v-model="staffForm.email" class="input" data-test="service-identity-email" type="email" required placeholder="zhangsan@wujie.local（仅作唯一标识）" />
              </div>
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">所在分组</label>
                <select v-model.number="staffForm.groupId" class="input" data-test="wizard-group" required>
                  <option :value="0" disabled>选择分组，如「视频组、后期组、AI 组」</option>
                  <option v-for="group in eligibleGroups" :key="group.id" :value="group.id">
                    {{ group.name }}
                  </option>
                </select>
                <p v-if="groupsLoading" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  正在加载可用组…
                </p>
                <p v-else-if="!eligibleGroups.length" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  没有可选分组，先在下方新建一个：
                </p>
                <div class="mt-2 flex flex-wrap items-center gap-2">
                  <input
                    v-model="quickGroupName"
                    class="input !w-48 !py-1 text-xs"
                    maxlength="50"
                    placeholder="新组名，如「视频组、后期组、AI 组」"
                    data-test="staff-quick-group-name"
                  />
                  <button
                    class="btn btn-sm btn-primary"
                    type="button"
                    :disabled="creatingGroup || !quickGroupName.trim()"
                    data-test="staff-quick-group-create"
                    @click="quickCreateGroup(quickGroupName)"
                  >
                    {{ creatingGroup ? '创建中…' : '创建并选中' }}
                  </button>
                </div>
              </div>
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">充值金额（美元，可填 0）</label>
                <input
                  v-model.number="staffForm.rechargeAmount"
                  class="input"
                  data-test="wizard-amount"
                  type="number"
                  min="0"
                  step="0.01"
                  placeholder="例如 50"
                />
                <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  按 1 美元 ≈ ¥{{ usdCnyRate }} 入账到该员工余额；不充可填 0，之后随时在行内「充值」。
                </p>
              </div>
              <div class="flex justify-end gap-2 pt-2">
                <button class="btn btn-outline" type="button" @click="closeStaffModal">取消</button>
                <button class="btn btn-primary" type="submit" data-test="wizard-submit" :disabled="submitting || groupsLoading || staffForm.groupId <= 0">
                  {{ submitting ? '开通中…' : '开卡' }}
                </button>
              </div>
            </form>
          </template>

          <!-- 明文双 Key（仅这一次） -->
          <template v-else>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">开卡成功</h2>
            <p v-if="issuedQCanvasPair.video.key && issuedQCanvasPair.media.key" class="mt-1 text-xs text-amber-600 dark:text-amber-300">
              两把完整 Key 只显示这一次，请立刻复制并交给员工。关掉后只能看到脱敏元数据。
            </p>
            <p v-else class="mt-1 text-xs text-amber-600 dark:text-amber-300">
              该请求已重放，明文 Key 不再返回；请不要把重试当作重新开卡。
            </p>
            <div v-if="issuedQCanvasPair.video.key && issuedQCanvasPair.media.key" class="mt-4 space-y-3">
              <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-900">
                <div class="text-xs text-gray-500">视频 Key · {{ selectedGroupName(staffForm.groupId) }}</div>
                <div class="mt-1 break-all font-mono text-sm text-gray-900 dark:text-white" data-test="wizard-video-key">{{ issuedQCanvasPair.video.key }}</div>
              </div>
              <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-900">
                <div class="text-xs text-gray-500">LLM / 图片 Key · {{ selectedGroupName(staffForm.groupId) }}</div>
                <div class="mt-1 break-all font-mono text-sm text-gray-900 dark:text-white" data-test="wizard-media-key">{{ issuedQCanvasPair.media.key }}</div>
              </div>
            </div>
            <p v-if="rechargeResult" class="mt-3 text-xs text-gray-600 dark:text-gray-300" data-test="wizard-recharge-result">
              {{ rechargeResult }}
            </p>
            <div class="mt-4 flex justify-end gap-2">
              <button v-if="issuedQCanvasPair.video.key && issuedQCanvasPair.media.key" class="btn btn-primary" type="button" data-test="wizard-copy" @click="copyIssuedQCanvasPair">
                <Icon name="copy" size="sm" />
                {{ qcanvasPairCopied ? '已复制' : '复制两把 Key' }}
              </button>
              <button class="btn btn-outline" type="button" data-test="wizard-done" @click="closeStaffModal">完成</button>
            </div>
          </template>
        </div>
      </div>

      <!-- 行内充值弹窗 -->
      <div v-if="rechargeModalOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="rechargeModalOpen = false">
        <div class="w-full max-w-sm rounded-lg border border-gray-200 bg-white p-6 shadow-xl dark:border-dark-700 dark:bg-dark-800">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            给 {{ staffDisplayName(rechargeTarget?.username, rechargeTarget?.email) }} 充值
          </h2>
          <form class="mt-5 space-y-4" data-test="recharge-form" @submit.prevent="submitRecharge">
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">充值金额（美元）</label>
              <input v-model.number="rechargeAmount" class="input" data-test="recharge-amount" type="number" min="0.01" step="0.01" required placeholder="例如 50" />
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">按 1 美元 ≈ ¥{{ usdCnyRate }} 入账，立即生效。</p>
            </div>
            <div class="flex justify-end gap-2 pt-2">
              <button class="btn btn-outline" type="button" @click="rechargeModalOpen = false">取消</button>
              <button class="btn btn-primary" type="submit" data-test="recharge-submit" :disabled="recharging">
                {{ recharging ? '充值中…' : '确认充值' }}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { RouterLink } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type { AdminGroup, AdminUser, ApiKey } from '@/types'
import type { BatchApiKeyUsageStats, BatchUserUsageStats } from '@/api/admin/dashboard'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { requestConfirmation } from '@/composables/useAppDialog'
import { DEFAULT_USD_CNY_RATE } from '@/composables/useDisplayCurrency'
import {
  CONSOLE_ERROR_ZH,
  formatAccountUsd,
  formatDateTime,
  formatMoney,
  quotaWarningBarClass,
  quotaWarningLevel,
  quotaWarningTextClass,
  staffDisplayName,
} from './consoleUtils'

const appStore = useAppStore()

const loading = ref(false)
const search = ref('')
const users = ref<AdminUser[]>([])
const usageMap = ref<Record<number, BatchUserUsageStats>>({})
// 后端 /admin/dashboard/stats 与 /admin/dashboard/users-ranking 均未提供实时汇率字段；
// 使用系统默认汇率展示，不臆造动态汇率。
const usdCnyRate = ref(DEFAULT_USD_CNY_RATE)

const filteredUsers = computed(() => {
  return users.value.filter((user) => user.role !== 'admin' && user.member_type === 'tool')
})

// ---- 员工列表 ----

async function loadStaff() {
  loading.value = true
  try {
    const res = await adminAPI.users.list(1, 100, {
      search: search.value.trim() || undefined,
      include_subscriptions: false,
      sort_by: 'created_at',
      sort_order: 'asc',
    })
    users.value = (res.items || []).filter((user) => user.role !== 'admin' && user.member_type === 'tool')
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
    appStore.showError(extractApiErrorMessage(err, '加载员工列表失败', CONSOLE_ERROR_ZH))
  } finally {
    loading.value = false
  }
}

// ---- 新增员工（一个弹窗：建账号 → 开双 Key → 充值 → 明文展示一次） ----

const staffModalOpen = ref(false)
const submitting = ref(false)
const wizardUser = ref<AdminUser | null>(null)
const rechargeResult = ref('')
const staffForm = reactive({
  username: '',
  email: '',
  groupId: 0,
  rechargeAmount: 0,
})

type IssuedQCanvasKeyPair = { video: ApiKey; media: ApiKey }
const issuedQCanvasPair = ref<IssuedQCanvasKeyPair | null>(null)
const qcanvasPairCopied = ref(false)

const activeGroups = ref<AdminGroup[]>([])
const groupsLoading = ref(true)
const creatingGroup = ref(false)
const quickGroupName = ref('')

// 新员工还没有任何专属授权：专属组与订阅组一律不可选（内部 member_type='tool' 照传，界面只用「员工」说法）
const eligibleGroups = computed(() => {
  return activeGroups.value.filter(
    (group) => group.status === 'active' && group.subscription_type !== 'subscription' && !group.is_exclusive,
  )
})

function selectedGroupName(groupID: number): string {
  return activeGroups.value.find((group) => group.id === groupID)?.name ?? `组 #${groupID}`
}

async function loadActiveGroups() {
  groupsLoading.value = true
  try {
    activeGroups.value = await adminAPI.groups.getAll()
  } catch (err) {
    activeGroups.value = []
    appStore.showError(extractApiErrorMessage(err, '加载分组失败', CONSOLE_ERROR_ZH))
  } finally {
    groupsLoading.value = false
  }
}

function openCreateStaff() {
  staffForm.username = ''
  staffForm.email = ''
  staffForm.groupId = eligibleGroups.value[0]?.id ?? 0
  staffForm.rechargeAmount = 0
  wizardUser.value = null
  rechargeResult.value = ''
  issuedQCanvasPair.value = null
  qcanvasPairCopied.value = false
  quickGroupName.value = ''
  staffModalOpen.value = true
}

function closeStaffModal() {
  staffModalOpen.value = false
  wizardUser.value = null
  rechargeResult.value = ''
  issuedQCanvasPair.value = null
  qcanvasPairCopied.value = false
}

async function quickCreateGroup(name: string) {
  const trimmed = name.trim()
  if (!trimmed || creatingGroup.value) return
  creatingGroup.value = true
  try {
    // 后端建组契约要求全量字段（缺失会 500）；与密钥库内联建组同一模板
    const created = await adminAPI.groups.create({
      name: trimmed,
      description: '',
      platform: 'openai',
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
    await loadActiveGroups()
    staffForm.groupId = created.id
    quickGroupName.value = ''
    appStore.showSuccess(`分组「${created.name}」已创建并选中`)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '创建分组失败', CONSOLE_ERROR_ZH))
  } finally {
    creatingGroup.value = false
  }
}

// 顺序执行：建账号（已建过则跳过，允许失败后原地重试）→ 开双 Key（同组，后端已放开）→ 充值（金额 > 0 时）
// 邮箱已存在（409）时复用既有员工继续签发：这是「上次签发失败」/「卡被删光」后的唯一补开路径
async function findStaffByEmail(email: string): Promise<AdminUser | null> {
  const res = await adminAPI.users.list(1, 100, {
    search: email,
    include_subscriptions: false,
    sort_by: 'created_at',
    sort_order: 'asc',
  })
  const normalized = email.toLowerCase()
  return (
    (res.items || []).find(
      (user) => user.email.toLowerCase() === normalized && user.member_type === 'tool' && user.role !== 'admin',
    ) ?? null
  )
}

async function submitStaff() {
  if (!eligibleGroups.value.some((group) => group.id === staffForm.groupId)) {
    appStore.showError('请先选择所在分组；没有可用分组时先新建一个')
    return
  }
  submitting.value = true
  rechargeResult.value = ''
  try {
    if (!wizardUser.value) {
      try {
        const res = await adminAPI.users.create({
          email: staffForm.email.trim(),
          username: staffForm.username.trim() || undefined,
          member_type: 'tool',
          role: 'user',
        })
        wizardUser.value = res.user
      } catch (createErr) {
        if ((createErr as { response?: { status?: number } })?.response?.status !== 409) throw createErr
        const existing = await findStaffByEmail(staffForm.email.trim())
        if (!existing) {
          appStore.showError('该邮箱已被占用，且不是可补开的员工账号，请换一个邮箱')
          return
        }
        wizardUser.value = existing
      }
    }
    const owner = wizardUser.value
    issuedQCanvasPair.value = await adminAPI.apiKeys.createQCanvasKeyPairForUser(
      owner.id,
      { video_group_id: staffForm.groupId, media_group_id: staffForm.groupId },
      crypto.randomUUID(),
    )
    const amount = Number(staffForm.rechargeAmount) || 0
    if (amount > 0) {
      try {
        const updated = await adminAPI.users.updateBalance(owner.id, amount, 'add', '开卡充值')
        rechargeResult.value = `已充值 $${amount.toFixed(2)}，当前余额 $${Number(updated.balance ?? 0).toFixed(2)}。`
      } catch (rechargeErr) {
        rechargeResult.value = `充值失败：${extractApiErrorMessage(rechargeErr, '请稍后在行内「充值」重试', CONSOLE_ERROR_ZH)}。Key 已签发，请先复制。`
        appStore.showError('双 Key 已开通，但充值失败，可在列表行内重试')
      }
    }
    if (expandedUserId.value === owner.id) await loadUserKeys(owner.id)
    await loadStaff()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '开卡失败', CONSOLE_ERROR_ZH))
  } finally {
    submitting.value = false
  }
}

// ---- 行内充值 ----

const rechargeModalOpen = ref(false)
const rechargeTarget = ref<AdminUser | null>(null)
const rechargeAmount = ref<number>(0)
const recharging = ref(false)

function openRecharge(user: AdminUser) {
  rechargeTarget.value = user
  rechargeAmount.value = 0
  rechargeModalOpen.value = true
}

async function submitRecharge() {
  const amount = Number(rechargeAmount.value)
  if (!rechargeTarget.value || !(amount > 0)) {
    appStore.showError('请输入大于 0 的充值金额')
    return
  }
  recharging.value = true
  try {
    const updated = await adminAPI.users.updateBalance(rechargeTarget.value.id, amount, 'add', '行内充值')
    appStore.showSuccess(`已充值 $${amount.toFixed(2)}，当前余额 $${Number(updated.balance ?? 0).toFixed(2)}`)
    rechargeModalOpen.value = false
    rechargeTarget.value = null
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '充值失败', CONSOLE_ERROR_ZH))
  } finally {
    recharging.value = false
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
    appStore.showError(extractApiErrorMessage(err, '加载员工卡片失败', CONSOLE_ERROR_ZH))
  } finally {
    keysLoading.value = false
  }
}

// 管理员列表/详情接口约定不会回显明文或部分明文（apiKeyDTOWithoutSecret 直接清空 key），
// 但这里绝不能"失败开放"：即便后端某天意外携带了非空 key，也只显示统一占位符，
// 不对其做任何 slice/展示。完整 Key 只能来自开卡响应弹窗（issuedQCanvasPair）。
function maskKey(): string {
  return '已发放·不再显示'
}

function keyStatusLabel(status: ApiKey['status']): string {
  const labels: Record<string, string> = {
    active: '在用',
    inactive: '已停用',
    disabled: '已停用',
    quota_exhausted: '额度用完',
    expired: '已过期',
  }
  return labels[status] ?? status
}

function keyStatusClass(status: ApiKey['status']): string {
  const classes: Record<string, string> = {
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
  // admin 契约只接受 active/disabled（不要发 inactive，后端会 400）
  const next = key.status === 'active' ? 'disabled' : 'active'
  if (next === 'disabled') {
    const confirmed = await requestConfirmation({
      message: `确定停用卡「${key.name || '未命名'}」？停用后该卡调用会立即失败。`,
      danger: true,
    })
    if (!confirmed) return
  }
  keyActionBusy.value = true
  try {
    await adminAPI.apiKeys.updateApiKeyFields(key.id, { status: next })
    appStore.showSuccess(next === 'active' ? '卡已启用' : '卡已停用')
    if (expandedUserId.value) await loadUserKeys(expandedUserId.value)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '切换卡状态失败', CONSOLE_ERROR_ZH))
  } finally {
    keyActionBusy.value = false
  }
}

async function removeKey(key: ApiKey) {
  const confirmed = await requestConfirmation({
    message: `确定删除卡「${key.name || '未命名'}」？删除后该卡立刻失效。`,
    danger: true,
  })
  if (!confirmed) return
  keyActionBusy.value = true
  try {
    await adminAPI.apiKeys.deleteApiKey(key.id)
    appStore.showSuccess('卡已删除')
    if (expandedUserId.value) await loadUserKeys(expandedUserId.value)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '删除卡失败', CONSOLE_ERROR_ZH))
  } finally {
    keyActionBusy.value = false
  }
}

async function copyIssuedQCanvasPair() {
  if (!issuedQCanvasPair.value?.video.key || !issuedQCanvasPair.value.media.key) return
  try {
    await navigator.clipboard.writeText(`video=${issuedQCanvasPair.value.video.key}\nmedia=${issuedQCanvasPair.value.media.key}`)
    qcanvasPairCopied.value = true
  } catch {
    appStore.showError('复制失败，请手动复制两把 Key')
  }
}

onMounted(() => {
  void loadStaff()
  void loadActiveGroups()
})
</script>
