<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 border-b border-gray-200 pb-4 dark:border-dark-700 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 class="ui-heading">AI 调用记录</h1>
          <p class="ui-subheading mt-1">谁调了什么模型、提示词是什么、花了多少钱。</p>
        </div>
        <button class="btn btn-outline" type="button" data-test="reload" :disabled="loading" @click="reload">
          <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
          刷新
        </button>
      </div>

      <!-- 筛选 -->
      <section class="ui-panel p-4">
        <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
          <div ref="userSearchRef" class="relative">
            <label class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">成员</label>
            <input
              v-model="userKeyword"
              class="input pr-8"
              type="text"
              placeholder="输入邮箱搜索成员"
              data-test="filter-user-search"
              @input="debounceUserSearch"
              @focus="showUserDropdown = true"
            />
            <button
              v-if="filterUserId > 0"
              type="button"
              class="absolute right-2 top-8 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
              aria-label="清除成员筛选"
              @click="clearUserFilter"
            >
              ✕
            </button>
            <div
              v-if="showUserDropdown && (userResults.length > 0 || (userKeyword.trim() && !userSearchLoading))"
              class="absolute z-50 mt-1 max-h-60 w-full overflow-auto rounded-lg border border-gray-200 bg-white shadow-lg dark:border-dark-600 dark:bg-dark-800"
            >
              <button
                v-for="user in userResults"
                :key="user.id"
                type="button"
                class="w-full px-3 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-dark-700"
                @click="selectUser(user)"
              >
                <span>{{ user.email }}<span v-if="user.deleted" class="ml-1 text-xs text-gray-400">（已删除）</span></span>
                <span class="ml-2 text-xs text-gray-400">#{{ user.id }}</span>
              </button>
              <div
                v-if="userKeyword.trim() && !userResults.length && !userSearchLoading"
                class="px-3 py-2 text-xs text-gray-500 dark:text-gray-400"
              >
                没有匹配的成员
              </div>
            </div>
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">模型</label>
            <input v-model="filterModel" class="input" placeholder="例如 seedance / gpt" @keyup.enter="onFilterChanged" />
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">开始日期</label>
            <input v-model="filterStart" class="input" type="date" @change="onFilterChanged" />
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">结束日期</label>
            <input v-model="filterEnd" class="input" type="date" @change="onFilterChanged" />
          </div>
          <div class="flex items-end">
            <button class="btn btn-outline w-full" type="button" @click="clearFilters">清空筛选</button>
          </div>
        </div>
        <div v-if="stats" class="mt-4 grid grid-cols-2 gap-3 border-t border-gray-100 pt-4 dark:border-dark-700 sm:grid-cols-4">
          <div>
            <div class="text-xs text-gray-500 dark:text-gray-400">调用次数</div>
            <div class="mt-1 text-lg font-semibold tabular-nums text-gray-900 dark:text-white">{{ formatCount(stats.total_requests) }}</div>
          </div>
          <div>
            <div class="text-xs text-gray-500 dark:text-gray-400">实际花费</div>
            <div class="mt-1 text-lg font-semibold tabular-nums text-gray-900 dark:text-white">{{ formatMoney(stats.total_actual_cost, usdCnyRate) }}</div>
          </div>
          <div>
            <div class="text-xs text-gray-500 dark:text-gray-400">Tokens</div>
            <div class="mt-1 text-lg font-semibold tabular-nums text-gray-900 dark:text-white">{{ formatTokens(stats.total_tokens) }}</div>
          </div>
          <div>
            <div class="text-xs text-gray-500 dark:text-gray-400">平均耗时</div>
            <div class="mt-1 text-lg font-semibold tabular-nums text-gray-900 dark:text-white">{{ formatDuration(stats.average_duration_ms) }}</div>
          </div>
        </div>
      </section>

      <!-- 调用明细 / 提示词采集 -->
      <div class="inline-flex rounded-lg border border-gray-200 bg-gray-50 p-0.5 dark:border-dark-700 dark:bg-dark-800">
        <button
          v-for="tab in innerTabs"
          :key="tab.key"
          type="button"
          :data-test="`tab-${tab.key}`"
          class="rounded-md px-4 py-1.5 text-sm font-medium transition-colors"
          :class="innerTab === tab.key
            ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
            : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'"
          @click="innerTab = tab.key"
        >
          {{ tab.label }}
        </button>
      </div>

      <!-- 调用明细表 -->
      <section v-show="innerTab === 'logs'" class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs uppercase text-gray-500 dark:bg-dark-700/40 dark:text-gray-400">
              <tr>
                <th class="px-5 py-3 font-medium">时间</th>
                <th class="px-5 py-3 font-medium">成员</th>
                <th class="px-5 py-3 font-medium">模型</th>
                <th class="px-5 py-3 font-medium">Tokens</th>
                <th class="px-5 py-3 font-medium">花费</th>
                <th class="px-5 py-3 font-medium">耗时</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="log in logs" :key="log.id" class="hover:bg-gray-50 dark:hover:bg-dark-700/40">
                <td class="whitespace-nowrap px-5 py-3 text-xs text-gray-500 dark:text-gray-400">{{ formatDateTime(log.created_at) }}</td>
                <td class="px-5 py-3">
                  <div class="font-medium text-gray-900 dark:text-white">{{ staffDisplayName(log.user?.username, log.user?.email) }}</div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">{{ log.api_key?.name || '' }}</div>
                </td>
                <td class="px-5 py-3">
                  <div class="flex flex-wrap items-center gap-1.5">
                    <span class="inline-flex rounded-md bg-gray-100 px-2 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-200">
                      {{ log.model }}
                    </span>
                    <span
                      v-if="log.image_count > 0"
                      class="inline-flex rounded-md bg-gray-100 px-1.5 py-0.5 text-[10px] font-medium text-gray-600 dark:bg-dark-700 dark:text-gray-300"
                    >
                      图片 ×{{ Math.max(log.image_count, 1) }}{{ log.image_size ? ` · ${log.image_size}` : '' }}
                    </span>
                  </div>
                </td>
                <td class="px-5 py-3 text-xs tabular-nums text-gray-600 dark:text-gray-300">
                  {{ formatTokens(log.input_tokens + log.output_tokens) }}
                </td>
                <td class="px-5 py-3 tabular-nums text-gray-900 dark:text-white">{{ formatMoney(log.actual_cost, usdCnyRate) }}</td>
                <td class="px-5 py-3 text-xs tabular-nums text-gray-500 dark:text-gray-400">{{ formatDuration(log.duration_ms) }}</td>
              </tr>
              <tr v-if="!loading && !logs.length">
                <td colspan="6" class="px-5 py-10 text-center text-sm text-gray-500 dark:text-gray-400">
                  没有符合条件的调用记录。
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="flex items-center justify-between border-t border-gray-100 px-5 py-3 text-sm dark:border-dark-700">
          <span class="text-xs text-gray-500 dark:text-gray-400">共 {{ logsTotal }} 条</span>
          <div class="flex items-center gap-2">
            <button class="btn btn-sm btn-outline" type="button" :disabled="page <= 1 || loading" @click="setPage(page - 1)">上一页</button>
            <span class="text-xs tabular-nums text-gray-600 dark:text-gray-300">{{ page }} / {{ Math.max(1, totalPages) }}</span>
            <button class="btn btn-sm btn-outline" type="button" :disabled="page >= totalPages || loading" @click="setPage(page + 1)">下一页</button>
          </div>
        </div>
      </section>

      <!-- 提示词采集样本 -->
      <section v-show="innerTab === 'prompts'" class="space-y-3">
        <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-200 px-1 pb-3 text-xs text-gray-500 dark:border-dark-700 dark:text-gray-400">
          <span>提示词与结果已脱敏采集。可在此标注采纳/质量分；完整周报也可在生成内容看板查看。</span>
          <RouterLink class="font-medium underline hover:no-underline" to="/admin/generation-content">
            打开生成内容看板
          </RouterLink>
        </div>

        <div
          v-if="weeklySummary"
          class="ui-panel p-4"
          data-test="weekly-report"
        >
          <h2 class="text-sm font-semibold text-gray-900 dark:text-white">经验周报摘要</h2>
          <dl class="mt-3 grid gap-3 text-xs sm:grid-cols-3">
            <div>
              <dt class="text-gray-500 dark:text-gray-400">周期</dt>
              <dd class="mt-0.5 text-gray-700 dark:text-gray-200">{{ weeklySummary.period }}</dd>
            </div>
            <div>
              <dt class="text-gray-500 dark:text-gray-400">条目</dt>
              <dd class="mt-0.5 tabular-nums text-gray-700 dark:text-gray-200">{{ weeklySummary.entries }}</dd>
            </div>
            <div>
              <dt class="text-gray-500 dark:text-gray-400">视频任务</dt>
              <dd class="mt-0.5 tabular-nums text-gray-700 dark:text-gray-200">{{ weeklySummary.videoTasks }}</dd>
            </div>
            <div>
              <dt class="text-gray-500 dark:text-gray-400">成本估算</dt>
              <dd class="mt-0.5 tabular-nums text-gray-700 dark:text-gray-200">{{ weeklySummary.cost }}</dd>
            </div>
            <div>
              <dt class="text-gray-500 dark:text-gray-400">采纳情况</dt>
              <dd class="mt-0.5 text-gray-700 dark:text-gray-200">{{ weeklySummary.adoption }}</dd>
            </div>
            <div>
              <dt class="text-gray-500 dark:text-gray-400">异常</dt>
              <dd class="mt-0.5 text-gray-700 dark:text-gray-200">{{ weeklySummary.anomalies }}</dd>
            </div>
          </dl>
        </div>
        <div
          v-else-if="weeklyReportError"
          data-test="weekly-report-error"
          class="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-xs text-amber-700 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-300"
        >
          {{ weeklyReportError }}
        </div>

        <div
          v-if="samplesError"
          data-test="samples-error"
          class="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-xs text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300"
        >
          {{ samplesError }}
        </div>
        <ContentWall
          v-else
          :samples="samples"
          :is-live="samplesLive"
          :usd-cny-rate="usdCnyRate"
          @updated="onAdoptionUpdated"
        />
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import ContentWall from '@/components/admin/generation-content/ContentWall.vue'
import { adminAPI } from '@/api/admin'
import type { AdminUsageLog } from '@/types'
import type { AdminUsageStatsResponse, SimpleUser } from '@/api/admin/usage'
import type { GenerationContentWeeklyReport, GenerationSample } from '@/api/admin/generation_content'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { DEFAULT_USD_CNY_RATE } from '@/composables/useDisplayCurrency'
import {
  formatCount,
  formatDateTime,
  formatDuration,
  formatMoney,
  formatTokens,
  parseAiRecordsQuery,
  staffDisplayName,
} from './consoleUtils'

const appStore = useAppStore()
const route = useRoute()

type InnerTab = 'logs' | 'prompts'
const innerTabs: Array<{ key: InnerTab; label: string }> = [
  { key: 'logs', label: '调用明细' },
  { key: 'prompts', label: '提示词采集' },
]
const innerTab = ref<InnerTab>('logs')

const loading = ref(false)
const logs = ref<AdminUsageLog[]>([])
const logsTotal = ref(0)
const page = ref(1)
const pageSize = 20
const stats = ref<AdminUsageStatsResponse | null>(null)
const samples = ref<GenerationSample[]>([])
const samplesLive = ref(false)
const samplesError = ref('')
const weeklyReport = ref<GenerationContentWeeklyReport | null>(null)
const weeklyReportError = ref('')
const userKeyword = ref('')
const userResults = ref<SimpleUser[]>([])
const showUserDropdown = ref(false)
const userSearchLoading = ref(false)
const userSearchRef = ref<HTMLElement | null>(null)
// 默认系统汇率，采集样本响应里若带有真实 usd_cny_rate 会覆盖此值（见 loadSamples）。
const usdCnyRate = ref(DEFAULT_USD_CNY_RATE)

// reload()/setPage() 共用一个 controller：新请求发出前先取消上一次同族请求，
// 组件卸载时统一取消，避免竞态更新已卸载组件的状态。
let reloadAbortController: AbortController | null = null
let userSearchAbortController: AbortController | null = null
let userSearchTimeout: ReturnType<typeof setTimeout> | null = null

function isAbortError(e: unknown): boolean {
  return (
    (e as { name?: string })?.name === 'AbortError' ||
    (e as { code?: string })?.code === 'ERR_CANCELED'
  )
}

const filterUserId = ref(0)
const filterModel = ref('')
const filterStart = ref('')
const filterEnd = ref('')

const totalPages = computed(() => Math.ceil(logsTotal.value / pageSize))

function applyRouteQuery() {
  const parsed = parseAiRecordsQuery(route.query as Record<string, unknown>)
  filterUserId.value = parsed.userId
  filterModel.value = parsed.model
  innerTab.value = parsed.tab
}

function buildFilterParams() {
  return {
    ...(filterUserId.value > 0 ? { user_id: filterUserId.value } : {}),
    ...(filterModel.value.trim() ? { model: filterModel.value.trim() } : {}),
    ...(filterStart.value ? { start_date: filterStart.value } : {}),
    ...(filterEnd.value ? { end_date: filterEnd.value } : {}),
  }
}

async function loadLogs(signal?: AbortSignal) {
  const res = await adminAPI.usage.list(
    {
      page: page.value,
      page_size: pageSize,
      ...buildFilterParams(),
    },
    { signal }
  )
  logs.value = res.items || []
  logsTotal.value = res.total || 0
}

async function loadStats(signal?: AbortSignal) {
  stats.value = await adminAPI.usage.getStats(buildFilterParams(), { signal })
}

async function loadSamples(signal?: AbortSignal) {
  const res = await adminAPI.generationContent.getSamples({ signal })
  samples.value = res.samples || []
  samplesLive.value = Boolean(res.is_live)
  if (typeof res.usd_cny_rate === 'number' && res.usd_cny_rate > 0) {
    usdCnyRate.value = res.usd_cny_rate
  }
}

async function loadWeeklyReport(signal?: AbortSignal) {
  weeklyReport.value = await adminAPI.generationContent.getWeeklyReport({ signal })
}

// 经验周报摘要：不再直接渲染英文 ledger markdown，改为结构化中文字段；
// 条目为 0 且成本为 0 时整块不显示（空态已有「暂无采集数据」卡），
// 但有失败任务信号时即使零数据也显示（失败信号不能吞）。
function formatPeriodDate(value: string): string {
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  return `${d.getFullYear()}/${d.getMonth() + 1}/${d.getDate()}`
}

const weeklySummary = computed(() => {
  const report = weeklyReport.value
  if (!report) return null
  if (!report.entries && !report.total_cost_estimate && !report.anomalies?.failed_tasks) return null
  const anomalies = report.anomalies
  const anomalyParts: string[] = []
  if (anomalies?.failed_tasks) anomalyParts.push(`失败任务 ${anomalies.failed_tasks}`)
  if (anomalies?.missing_task_joins) anomalyParts.push(`缺关联 ${anomalies.missing_task_joins}`)
  if (anomalies?.truncated_rows) anomalyParts.push(`截断行 ${anomalies.truncated_rows}`)
  return {
    period: `${formatPeriodDate(report.period_start)} ~ ${formatPeriodDate(report.period_end)}`,
    entries: report.entries,
    videoTasks: report.video_tasks,
    cost: formatMoney(report.total_cost_estimate, report.usd_cny_rate ?? usdCnyRate.value),
    adoption: `采纳率 ${Math.round((report.adoption_rate || 0) * 100)}% · 采纳 ${report.adopted_count} · 驳回 ${report.rejected_count} · 待审 ${(report.pending_count || 0) + (report.unreviewed_count || 0)}`,
    anomalies: anomalyParts.length ? anomalyParts.join(' · ') : '无',
  }
})

const selectedUser = ref<SimpleUser | null>(null)

function debounceUserSearch() {
  if (userSearchTimeout) clearTimeout(userSearchTimeout)
  userSearchTimeout = setTimeout(() => {
    void runUserSearch()
  }, 300)
}

async function runUserSearch() {
  const keyword = userKeyword.value.trim()
  if (filterUserId.value > 0 && selectedUser.value && keyword !== selectedUser.value.email) {
    filterUserId.value = 0
    selectedUser.value = null
  }
  if (!keyword) {
    userResults.value = []
    return
  }
  userSearchAbortController?.abort()
  const c = new AbortController()
  userSearchAbortController = c
  userSearchLoading.value = true
  try {
    userResults.value = await adminAPI.usage.searchUsers(keyword)
  } catch (err) {
    if (!isAbortError(err)) userResults.value = []
  } finally {
    if (userSearchAbortController === c) userSearchLoading.value = false
  }
}

function selectUser(user: SimpleUser) {
  selectedUser.value = user
  userKeyword.value = user.email
  showUserDropdown.value = false
  filterUserId.value = user.id
  onFilterChanged()
}

function clearUserFilter() {
  selectedUser.value = null
  userKeyword.value = ''
  userResults.value = []
  showUserDropdown.value = false
  filterUserId.value = 0
  onFilterChanged()
}

async function resolveUserFromFilter() {
  if (filterUserId.value <= 0) {
    selectedUser.value = null
    if (!userKeyword.value) userKeyword.value = ''
    return
  }
  if (selectedUser.value?.id === filterUserId.value) {
    userKeyword.value = selectedUser.value.email
    return
  }
  try {
    const results = await adminAPI.usage.searchUsers(String(filterUserId.value))
    const matched = results.find((user) => user.id === filterUserId.value)
    if (matched) {
      selectedUser.value = matched
      userKeyword.value = matched.email
    }
  } catch {
    // 保留 filterUserId，仅无法回显邮箱
  }
}

function onDocumentClick(event: MouseEvent) {
  const target = event.target as Node | null
  if (userSearchRef.value && target && !userSearchRef.value.contains(target)) {
    showUserDropdown.value = false
  }
}

async function reload() {
  reloadAbortController?.abort()
  const c = new AbortController()
  reloadAbortController = c
  loading.value = true
  samplesError.value = ''
  weeklyReportError.value = ''

  const [logsResult, statsResult, samplesResult, weeklyResult] = await Promise.allSettled([
    loadLogs(c.signal),
    loadStats(c.signal),
    loadSamples(c.signal),
    loadWeeklyReport(c.signal),
  ])

  if (c.signal.aborted) return

  // 明细/统计失败：沿用既有 toast 提示（不是本次修复范围内的"静默吞错"）。
  if (logsResult.status === 'rejected' && !isAbortError(logsResult.reason)) {
    appStore.showError(extractApiErrorMessage(logsResult.reason, '加载调用记录失败'))
  }
  if (statsResult.status === 'rejected' && !isAbortError(statsResult.reason)) {
    appStore.showError(extractApiErrorMessage(statsResult.reason, '加载统计数据失败'))
  }
  // 采集样本/经验周报失败：曾经被 catch 吞掉后渲染成"诚实空态"，
  // 现在改为就地错误提示，绝不能让失败看起来像"这里本来就没数据"。
  if (samplesResult.status === 'rejected' && !isAbortError(samplesResult.reason)) {
    samplesError.value = extractApiErrorMessage(samplesResult.reason, '加载采集样本失败')
  }
  if (weeklyResult.status === 'rejected' && !isAbortError(weeklyResult.reason)) {
    weeklyReportError.value = extractApiErrorMessage(weeklyResult.reason, '加载经验周报失败')
  }

  if (reloadAbortController === c) loading.value = false
}

function onAdoptionUpdated() {
  loadSamples()
    .then(() => {
      samplesError.value = ''
    })
    .catch((err) => {
      if (!isAbortError(err)) samplesError.value = extractApiErrorMessage(err, '加载采集样本失败')
    })
  loadWeeklyReport()
    .then(() => {
      weeklyReportError.value = ''
    })
    .catch((err) => {
      if (!isAbortError(err)) weeklyReportError.value = extractApiErrorMessage(err, '加载经验周报失败')
    })
}

function onFilterChanged() {
  page.value = 1
  void reload()
}

function clearFilters() {
  selectedUser.value = null
  userKeyword.value = ''
  userResults.value = []
  showUserDropdown.value = false
  filterUserId.value = 0
  filterModel.value = ''
  filterStart.value = ''
  filterEnd.value = ''
  onFilterChanged()
}

async function setPage(next: number) {
  page.value = Math.max(1, Math.min(next, totalPages.value))
  reloadAbortController?.abort()
  const c = new AbortController()
  reloadAbortController = c
  loading.value = true
  try {
    await loadLogs(c.signal)
  } catch (err) {
    if (!isAbortError(err)) {
      appStore.showError(extractApiErrorMessage(err, '加载调用记录失败'))
    }
  } finally {
    if (reloadAbortController === c) loading.value = false
  }
}

onMounted(() => {
  applyRouteQuery()
  void resolveUserFromFilter()
  void reload()
  document.addEventListener('click', onDocumentClick)
})

// Same-route query drill-downs (e.g. BossOverview → ai-records?user_id=) reuse the component;
// re-apply filters whenever the query changes.
watch(
  () => route.query,
  () => {
    applyRouteQuery()
    void resolveUserFromFilter()
    page.value = 1
    void reload()
  }
)

onUnmounted(() => {
  reloadAbortController?.abort()
  userSearchAbortController?.abort()
  if (userSearchTimeout) clearTimeout(userSearchTimeout)
  document.removeEventListener('click', onDocumentClick)
})
</script>
