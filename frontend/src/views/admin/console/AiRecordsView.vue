<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 border-b border-gray-200 pb-4 dark:border-dark-700 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">任务记录</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">谁调了什么模型、提示词是什么、花了多少钱。</p>
        </div>
        <button class="btn btn-outline" type="button" :disabled="loading" @click="reload">
          <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
          刷新
        </button>
      </div>

      <!-- 与视频任务页共用的顶部切换 -->
      <div class="inline-flex rounded-lg border border-gray-200 bg-gray-50 p-0.5 dark:border-dark-700 dark:bg-dark-800">
        <RouterLink
          to="/admin/video/tasks"
          class="rounded-md px-4 py-1.5 text-sm font-medium text-gray-500 transition-colors hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
        >
          视频任务
        </RouterLink>
        <span class="rounded-md bg-white px-4 py-1.5 text-sm font-medium text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white">
          AI 调用记录
        </span>
      </div>

      <!-- 筛选 -->
      <section class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
        <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">员工</label>
            <select v-model="filterUserId" class="input" @change="onFilterChanged">
              <option :value="0">全部员工</option>
              <option v-for="user in staffOptions" :key="user.id" :value="user.id">
                {{ staffDisplayName(user.username, user.email) }}（{{ user.email }}）
              </option>
            </select>
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
            <div class="mt-1 text-lg font-semibold tabular-nums text-teal-600 dark:text-teal-300">{{ formatMoney(stats.total_actual_cost, usdCnyRate) }}</div>
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
                <th class="px-5 py-3 font-medium">员工</th>
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
                      v-if="log.media_type === 'image' || log.image_count > 0"
                      class="inline-flex rounded-md bg-purple-50 px-1.5 py-0.5 text-[10px] font-medium text-purple-700 dark:bg-purple-500/10 dark:text-purple-300"
                    >
                      图片 ×{{ Math.max(log.image_count, 1) }}{{ log.image_size ? ` · ${log.image_size}` : '' }}
                    </span>
                  </div>
                </td>
                <td class="px-5 py-3 text-xs tabular-nums text-gray-600 dark:text-gray-300">
                  {{ formatTokens(log.input_tokens + log.output_tokens) }}
                </td>
                <td class="px-5 py-3 tabular-nums font-medium text-teal-600 dark:text-teal-300">{{ formatMoney(log.actual_cost, usdCnyRate) }}</td>
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
        <div class="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-teal-200 bg-teal-50 px-4 py-3 text-xs text-teal-800 dark:border-teal-500/20 dark:bg-teal-500/10 dark:text-teal-100">
          <span>提示词与结果已脱敏采集。可在此标注采纳/质量分；完整周报也可在生成内容看板查看。</span>
          <RouterLink class="font-medium underline hover:no-underline" to="/admin/generation-content">
            打开生成内容看板
          </RouterLink>
        </div>

        <div
          v-if="weeklyReport"
          class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800"
        >
          <div class="flex flex-wrap items-end justify-between gap-3">
            <div>
              <h2 class="text-sm font-semibold text-gray-900 dark:text-white">经验周报摘要</h2>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                采纳率 {{ Math.round((weeklyReport.adoption_rate || 0) * 100) }}% ·
                采纳 {{ weeklyReport.adopted_count }} /
                待审 {{ (weeklyReport.pending_count || 0) + (weeklyReport.unreviewed_count || 0) }}
              </p>
            </div>
            <div class="text-right text-xs text-gray-500 dark:text-gray-400">
              条目 {{ weeklyReport.entries }} · 视频任务 {{ weeklyReport.video_tasks }}
            </div>
          </div>
          <pre
            v-if="weeklyReport.markdown"
            class="mt-3 max-h-48 overflow-auto whitespace-pre-wrap rounded-md bg-gray-50 p-3 text-xs text-gray-700 dark:bg-dark-900 dark:text-gray-200"
          >{{ weeklyReport.markdown }}</pre>
        </div>

        <ContentWall
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
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import ContentWall from '@/components/admin/generation-content/ContentWall.vue'
import { adminAPI } from '@/api/admin'
import type { AdminUsageLog, AdminUser } from '@/types'
import type { AdminUsageStatsResponse } from '@/api/admin/usage'
import type { GenerationContentWeeklyReport, GenerationSample } from '@/api/admin/generation_content'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
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
const weeklyReport = ref<GenerationContentWeeklyReport | null>(null)
const staffOptions = ref<AdminUser[]>([])
const usdCnyRate = ref(7.2)

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

async function loadLogs() {
  const res = await adminAPI.usage.list({
    page: page.value,
    page_size: pageSize,
    ...buildFilterParams(),
  })
  logs.value = res.items || []
  logsTotal.value = res.total || 0
}

async function loadStats() {
  stats.value = await adminAPI.usage.getStats(buildFilterParams())
}

async function loadSamples() {
  try {
    const res = await adminAPI.generationContent.getSamples()
    samples.value = res.samples || []
    samplesLive.value = Boolean(res.is_live)
    if (typeof res.usd_cny_rate === 'number' && res.usd_cny_rate > 0) {
      usdCnyRate.value = res.usd_cny_rate
    }
  } catch {
    samples.value = []
    samplesLive.value = false
  }
}

async function loadWeeklyReport() {
  try {
    weeklyReport.value = await adminAPI.generationContent.getWeeklyReport()
  } catch {
    weeklyReport.value = null
  }
}

async function loadStaffOptions() {
  try {
    const res = await adminAPI.users.list(1, 100)
    staffOptions.value = res.items || []
  } catch {
    staffOptions.value = []
  }
}

async function reload() {
  loading.value = true
  try {
    const [, , , , rateStats] = await Promise.all([
      loadLogs(),
      loadStats(),
      loadSamples(),
      loadWeeklyReport(),
      adminAPI.dashboard.getStats().catch(() => null),
    ])
    usdCnyRate.value = Number(rateStats?.usd_cny_rate || usdCnyRate.value || 7.2)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载调用记录失败'))
  } finally {
    loading.value = false
  }
}

function onAdoptionUpdated() {
  void loadSamples()
  void loadWeeklyReport()
}

function onFilterChanged() {
  page.value = 1
  void reload()
}

function clearFilters() {
  filterUserId.value = 0
  filterModel.value = ''
  filterStart.value = ''
  filterEnd.value = ''
  onFilterChanged()
}

function setPage(next: number) {
  page.value = Math.max(1, Math.min(next, totalPages.value))
  loading.value = true
  loadLogs()
    .catch((err) => appStore.showError(extractApiErrorMessage(err, '加载调用记录失败')))
    .finally(() => {
      loading.value = false
    })
}

onMounted(() => {
  applyRouteQuery()
  void loadStaffOptions()
  void reload()
})
</script>
