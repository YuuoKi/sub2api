<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 border-b border-gray-200 pb-4 dark:border-dark-700 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 class="ui-heading">总览</h1>
          <p class="ui-subheading mt-1">花了多少钱、做了多少调用、谁在用什么 AI，一眼看完。</p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <div class="inline-flex rounded-lg border border-gray-200 bg-gray-50 p-0.5 dark:border-dark-700 dark:bg-dark-800">
            <button
              v-for="option in rangeOptions"
              :key="option.key"
              type="button"
              class="rounded-md px-3 py-1.5 text-xs font-medium transition-colors"
              :class="rangeKey === option.key
                ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
                : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'"
              @click="setRange(option.key)"
            >
              {{ option.label }}
            </button>
          </div>
          <button class="btn btn-outline" type="button" :disabled="loading" @click="loadAll">
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
            刷新
          </button>
        </div>
      </div>

      <!-- 公司月度总预算进度 -->
      <section
        v-if="monthlyBudgetCNY > 0"
        class="ui-panel p-4"
      >
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div class="min-w-0">
            <p class="text-sm font-medium text-gray-900 dark:text-white">本月预算进度</p>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              已用 ¥{{ monthlySpendCNY.toFixed(2) }} / 预算 ¥{{ monthlyBudgetCNY.toFixed(2) }}
              （{{ monthlyBudgetUsagePercent.toFixed(1) }}%）
            </p>
          </div>
          <form class="flex flex-wrap items-center gap-2" @submit.prevent="saveMonthlyBudget">
            <input
              v-model.number="budgetDraft"
              type="number"
              min="0"
              step="1"
              class="input h-9 w-32"
              placeholder="月预算 ¥"
            />
            <button class="btn btn-sm btn-outline" type="submit" :disabled="budgetSaving">
              {{ budgetSaving ? '保存中…' : '更新预算' }}
            </button>
          </form>
        </div>
        <div class="mt-3 h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
          <div
            class="h-full rounded-full transition-all"
            :class="monthlyBudgetUsagePercent >= 100
              ? 'bg-red-500'
              : monthlyBudgetUsagePercent >= 80
                ? 'bg-yellow-500'
                : 'bg-teal-500'"
            :style="{ width: `${Math.min(100, monthlyBudgetUsagePercent)}%` }"
          />
        </div>
      </section>
      <section
        v-else
        class="flex flex-col gap-3 rounded-lg border border-dashed border-gray-300 bg-gray-50 p-4 dark:border-dark-600 dark:bg-dark-800/60 sm:flex-row sm:items-center sm:justify-between"
      >
        <p class="text-sm text-gray-600 dark:text-gray-300">尚未设置公司月度总预算。设一个数字后，总览会显示本月花费进度条。</p>
        <form class="flex flex-wrap items-center gap-2" @submit.prevent="saveMonthlyBudget">
          <input
            v-model.number="budgetDraft"
            type="number"
            min="0"
            step="1"
            class="input h-9 w-32"
            placeholder="例如 1000"
          />
          <button class="btn btn-sm btn-outline" type="submit" :disabled="budgetSaving">设置月预算</button>
        </form>
      </section>

      <!-- P2-4：备份超期黄条 -->
      <section
        v-if="backupStale"
        class="flex flex-col gap-3 rounded-lg border border-yellow-200 bg-yellow-50 p-4 dark:border-yellow-500/30 dark:bg-yellow-500/10 sm:flex-row sm:items-center sm:justify-between"
        role="status"
      >
        <div class="min-w-0">
          <p class="text-sm font-medium text-yellow-800 dark:text-yellow-200">
            {{ backupStaleMessage }}
          </p>
          <p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
            建议到数据管理做一次备份，避免故障时无法回滚。
          </p>
        </div>
        <RouterLink class="btn btn-sm btn-outline shrink-0" to="/admin/settings">
          去系统设置
        </RouterLink>
      </section>

      <!-- 核心指标卡 -->
      <section class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <div
          v-for="card in statCards"
          :key="card.label"
          class="relative overflow-hidden rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800"
        >
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0">
              <div class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ card.label }}</div>
              <div class="mt-2 truncate text-3xl font-semibold tabular-nums text-gray-900 dark:text-white">
                <AnimatedNumber :value="card.value" :format="card.format" />
              </div>
              <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ card.hint }}</div>
            </div>
            <svg class="h-14 w-14 shrink-0 -rotate-90" viewBox="0 0 48 48" aria-hidden="true">
              <circle cx="24" cy="24" r="20" fill="none" stroke-width="4" class="stroke-gray-100 dark:stroke-dark-700" />
              <circle
                cx="24" cy="24" r="20" fill="none" stroke-width="4" stroke-linecap="round"
                :class="card.ringClass"
                :stroke-dasharray="RING_CIRCUMFERENCE"
                :stroke-dashoffset="ringOffset(card.ratio)"
                style="transition: stroke-dashoffset 0.9s cubic-bezier(0.33, 1, 0.68, 1)"
              />
              <foreignObject x="12" y="12" width="24" height="24" class="rotate-90 origin-center">
                <div class="flex h-full w-full items-center justify-center">
                  <Icon :name="card.icon" size="sm" :class="card.iconClass" />
                </div>
              </foreignObject>
            </svg>
          </div>
        </div>
      </section>

      <!-- 花费趋势 + 模型分布 -->
      <div class="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]">
        <section class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
          <div class="flex items-center justify-between gap-3">
            <div>
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">花费趋势</h2>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ rangeLabel }}内每天实际花费（人民币）</p>
            </div>
            <div class="text-right">
              <div class="text-xs text-gray-500 dark:text-gray-400">本期合计</div>
              <div class="text-lg font-semibold text-teal-600 dark:text-teal-300">{{ formatMoney(totalSpend, usdCnyRate) }}</div>
            </div>
          </div>
          <div class="mt-4 h-64">
            <Line v-if="trendChartData" :data="trendChartData" :options="trendChartOptions" />
            <div v-else class="flex h-full items-center justify-center text-sm text-gray-500 dark:text-gray-400">
              {{ loading ? '加载中…' : '本期还没有调用记录。' }}
            </div>
          </div>
        </section>

        <section class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">用了什么 AI</h2>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">按实际花费统计的模型分布</p>
          <div v-if="modelChartData" class="mt-4 flex items-center gap-5">
            <div class="h-44 w-44 shrink-0">
              <Doughnut :data="modelChartData" :options="modelChartOptions" />
            </div>
            <div class="max-h-48 min-w-0 flex-1 space-y-2 overflow-y-auto pr-1">
              <button
                v-for="(model, index) in topModels"
                :key="model.model"
                type="button"
                class="flex w-full items-center gap-2 rounded-md px-1 py-0.5 text-left text-xs hover:bg-gray-50 dark:hover:bg-dark-700/40"
                @click="goAiRecordsByModel(model.model)"
              >
                <span class="h-2.5 w-2.5 shrink-0 rounded-full" :style="{ backgroundColor: chartColors[index % chartColors.length] }"></span>
                <span class="min-w-0 flex-1 truncate text-gray-700 dark:text-gray-200" :title="model.model">{{ model.model }}</span>
                <span class="shrink-0 tabular-nums text-gray-500 dark:text-gray-400">{{ formatCount(model.requests) }} 次</span>
                <span class="shrink-0 tabular-nums font-medium text-teal-600 dark:text-teal-300">{{ formatMoney(model.actual_cost, usdCnyRate) }}</span>
              </button>
            </div>
          </div>
          <div v-else class="mt-4 flex h-44 items-center justify-center text-sm text-gray-500 dark:text-gray-400">
            {{ loading ? '加载中…' : '本期还没有模型调用。' }}
          </div>
        </section>
      </div>

      <!-- 成员消费排行 -->
      <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="flex items-center justify-between gap-3 border-b border-gray-200 px-5 py-4 dark:border-dark-700">
          <div>
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">成员消费排行</h2>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ rangeLabel }}内谁在用、用了多少。员工和工具分开标记，互不污染。</p>
          </div>
          <RouterLink class="btn btn-sm btn-outline" to="/admin/console/staff">
            <Icon name="users" size="sm" />
            成员与开卡
          </RouterLink>
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs uppercase text-gray-500 dark:bg-dark-700/40 dark:text-gray-400">
              <tr>
                <th class="px-5 py-3 font-medium">排名</th>
                <th class="px-5 py-3 font-medium">成员</th>
                <th class="px-5 py-3 font-medium">调用次数</th>
                <th class="px-5 py-3 font-medium">Tokens</th>
                <th class="px-5 py-3 font-medium">花费</th>
                <th class="w-1/5 px-5 py-3 font-medium">占比</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr
                v-for="(item, index) in ranking"
                :key="item.user_id"
                class="cursor-pointer hover:bg-gray-50 dark:hover:bg-dark-700/40"
                @click="goAiRecordsByUser(item.user_id)"
              >
                <td class="px-5 py-3">
                  <span
                    class="inline-flex h-6 w-6 items-center justify-center rounded-full text-xs font-semibold"
                    :class="index < 3 ? 'bg-teal-100 text-teal-700 dark:bg-teal-500/20 dark:text-teal-200' : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'"
                  >
                    {{ index + 1 }}
                  </span>
                </td>
                <td class="px-5 py-3 font-medium text-gray-900 dark:text-white">
                  <span class="inline-flex items-center gap-1.5">
                    {{ item.email || `用户 #${item.user_id}` }}
                    <span v-if="item.member_type === 'tool'" class="rounded bg-sky-100 px-1.5 py-0.5 text-[10px] font-medium text-sky-700 dark:bg-sky-500/20 dark:text-sky-200">工具</span>
                  </span>
                </td>
                <td class="px-5 py-3 tabular-nums text-gray-700 dark:text-gray-200">{{ formatCount(item.requests) }}</td>
                <td class="px-5 py-3 tabular-nums text-gray-700 dark:text-gray-200">{{ formatTokens(item.tokens) }}</td>
                <td class="px-5 py-3 tabular-nums font-medium text-teal-600 dark:text-teal-300">{{ formatMoney(item.actual_cost, usdCnyRate) }}</td>
                <td class="px-5 py-3">
                  <div class="flex items-center gap-2">
                    <div class="h-2 flex-1 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                      <div
                        class="h-full rounded-full bg-gradient-to-r from-teal-500 to-cyan-400"
                        :style="{ width: `${spendRatio(item.actual_cost)}%`, transition: 'width 0.9s cubic-bezier(0.33, 1, 0.68, 1)' }"
                      ></div>
                    </div>
                    <span class="w-12 shrink-0 text-right text-xs tabular-nums text-gray-500 dark:text-gray-400">{{ spendRatio(item.actual_cost).toFixed(1) }}%</span>
                  </div>
                </td>
              </tr>
              <tr v-if="!loading && !ranking.length">
                <td colspan="6" class="px-5 py-10 text-center text-sm text-gray-500 dark:text-gray-400">
                  本期还没有成员调用记录。先去
                  <RouterLink class="font-medium text-teal-600 hover:underline dark:text-teal-300" to="/admin/console/staff">成员与开卡</RouterLink>
                  给成员开一张卡。
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import {
  ArcElement,
  CategoryScale,
  Chart as ChartJS,
  Filler,
  Legend,
  LinearScale,
  LineElement,
  PointElement,
  Tooltip,
} from 'chart.js'
import { Doughnut, Line } from 'vue-chartjs'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import AnimatedNumber from './AnimatedNumber.vue'
import { adminAPI } from '@/api/admin'
import type { ModelStat, TrendDataPoint, UserSpendingRankingItem } from '@/types'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { DEFAULT_USD_CNY_RATE } from '@/composables/useDisplayCurrency'
import {
  type ConsoleRangeKey,
  formatCount,
  formatMoney,
  formatTokens,
  getConsoleRange,
} from './consoleUtils'

ChartJS.register(ArcElement, CategoryScale, LinearScale, PointElement, LineElement, Filler, Tooltip, Legend)

const appStore = useAppStore()
const router = useRouter()

const loading = ref(false)
const rangeKey = ref<ConsoleRangeKey>('30d')
const trend = ref<TrendDataPoint[]>([])
const models = ref<ModelStat[]>([])
const ranking = ref<UserSpendingRankingItem[]>([])
const rankingTotals = ref({ actual_cost: 0, requests: 0, tokens: 0 })
// 后端 /admin/dashboard/stats 未提供实时汇率字段；使用系统默认汇率展示，不臆造动态汇率。
const usdCnyRate = ref(DEFAULT_USD_CNY_RATE)
const monthlyBudgetCNY = ref(0)
const monthlySpendCNY = ref(0)
const monthlyBudgetUsagePercent = ref(0)
const budgetDraft = ref(0)
const budgetSaving = ref(false)
const backupStale = ref(false)
const backupStaleMessage = ref('')

const rangeOptions: Array<{ key: ConsoleRangeKey; label: string }> = [
  { key: '7d', label: '近 7 天' },
  { key: '30d', label: '近 30 天' },
  { key: 'month', label: '本月' },
]

const rangeLabel = computed(() => rangeOptions.find((option) => option.key === rangeKey.value)?.label ?? '本期')

const chartColors = ['#14b8a6', '#06b6d4', '#8b5cf6', '#f59e0b', '#ef4444', '#ec4899', '#84cc16', '#6366f1', '#f97316', '#10b981']

const RING_CIRCUMFERENCE = 2 * Math.PI * 20

function ringOffset(ratio: number): number {
  const clamped = Math.max(0.03, Math.min(1, ratio))
  return RING_CIRCUMFERENCE * (1 - clamped)
}

const totalSpend = computed(() => rankingTotals.value.actual_cost)
const totalRequests = computed(() => rankingTotals.value.requests)
const totalTokens = computed(() => rankingTotals.value.tokens)
const activeStaff = computed(
  () => ranking.value.filter((item) => item.actual_cost > 0 || item.requests > 0).length,
)

type StatCard = {
  label: string
  value: number
  hint: string
  format?: (value: number) => string
  ratio: number
  icon: 'dollar' | 'bolt' | 'play' | 'users'
  ringClass: string
  iconClass: string
}

const statCards = computed<StatCard[]>(() => {
  return [
    {
      label: `${rangeLabel.value}总花费`,
      value: totalSpend.value,
      hint: '所有成员实际扣费合计',
      format: (v) => formatMoney(v, usdCnyRate.value),
      ratio: totalSpend.value > 0 ? 1 : 0,
      icon: 'dollar',
      ringClass: 'stroke-teal-500',
      iconClass: 'text-teal-600 dark:text-teal-300',
    },
    {
      label: 'AI 调用次数',
      value: totalRequests.value,
      hint: `${rangeLabel.value}全部模型调用`,
      ratio: totalRequests.value > 0 ? 1 : 0,
      icon: 'bolt',
      ringClass: 'stroke-cyan-500',
      iconClass: 'text-cyan-600 dark:text-cyan-300',
    },
    {
      label: 'Token 用量',
      value: totalTokens.value,
      hint: `${rangeLabel.value}累计 Token`,
      format: (v) => formatTokens(v),
      ratio: totalTokens.value > 0 ? 1 : 0,
      icon: 'play',
      ringClass: 'stroke-emerald-500',
      iconClass: 'text-emerald-600 dark:text-emerald-300',
    },
    {
      label: '活跃成员',
      value: activeStaff.value,
      hint: `${rangeLabel.value}有调用记录的成员`,
      ratio: activeStaff.value > 0 ? 1 : 0,
      icon: 'users',
      ringClass: 'stroke-violet-500',
      iconClass: 'text-violet-600 dark:text-violet-300',
    },
  ]
})

const trendChartData = computed(() => {
  if (!trend.value.length) return null
  return {
    labels: trend.value.map((point) => {
      const d = new Date(point.date)
      return Number.isNaN(d.getTime()) ? point.date : `${d.getMonth() + 1}/${d.getDate()}`
    }),
    datasets: [
      {
        label: '实际花费',
        data: trend.value.map((point) => Number(point.actual_cost.toFixed(4))),
        borderColor: '#14b8a6',
        backgroundColor: 'rgba(20, 184, 166, 0.15)',
        pointBackgroundColor: '#14b8a6',
        pointRadius: 2,
        pointHoverRadius: 4,
        borderWidth: 2,
        fill: true,
        tension: 0.35,
      },
    ],
  }
})

const trendChartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  animation: { duration: 800, easing: 'easeOutQuart' as const },
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        label: (context: { raw: unknown }) => ` 花费 ${formatMoney(Number(context.raw), usdCnyRate.value)}`,
      },
    },
  },
  scales: {
    x: { grid: { display: false }, ticks: { maxTicksLimit: 10, color: '#94a3b8' } },
    y: {
      grid: { color: 'rgba(148, 163, 184, 0.15)' },
      ticks: { color: '#94a3b8', callback: (value: string | number) => formatMoney(Number(value), usdCnyRate.value) },
      beginAtZero: true,
    },
  },
}

const topModels = computed(() => {
  return [...models.value].sort((a, b) => b.actual_cost - a.actual_cost).slice(0, 8)
})

const modelChartData = computed(() => {
  if (!topModels.value.length) return null
  return {
    labels: topModels.value.map((model) => model.model),
    datasets: [
      {
        data: topModels.value.map((model) => model.actual_cost),
        backgroundColor: chartColors.slice(0, topModels.value.length),
        borderWidth: 0,
      },
    ],
  }
})

const modelChartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  cutout: '62%',
  animation: { animateRotate: true, duration: 900 },
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        label: (context: { label?: string; raw: unknown; dataset: { data: unknown[] } }) => {
          const value = Number(context.raw)
          const total = (context.dataset.data as number[]).reduce((sum, v) => sum + v, 0)
          const pct = total > 0 ? ((value / total) * 100).toFixed(1) : '0.0'
          return ` ${context.label}: ${formatMoney(value, usdCnyRate.value)} (${pct}%)`
        },
      },
    },
  },
}

function spendRatio(cost: number): number {
  const total = totalSpend.value
  if (total <= 0) return 0
  return Math.min(100, (cost / total) * 100)
}

function setRange(key: ConsoleRangeKey) {
  if (rangeKey.value === key) return
  rangeKey.value = key
  void loadAll()
}

function goAiRecordsByUser(userId: number) {
  void router.push({ path: '/admin/console/ai-records', query: { user_id: String(userId) } })
}

function goAiRecordsByModel(model: string) {
  const trimmed = (model || '').trim()
  if (!trimmed) return
  void router.push({ path: '/admin/console/ai-records', query: { model: trimmed } })
}

async function saveMonthlyBudget() {
  const value = Number(budgetDraft.value)
  if (!Number.isFinite(value) || value < 0) {
    appStore.showError('月预算须为大于等于 0 的数字')
    return
  }
  budgetSaving.value = true
  try {
    const res = await adminAPI.dashboard.updateMonthlyBudget(value)
    monthlyBudgetCNY.value = Number(res.monthly_budget_cny || 0)
    monthlySpendCNY.value = Number(res.monthly_spend_cny || 0)
    monthlyBudgetUsagePercent.value = Number(res.monthly_budget_usage_percent || 0)
    budgetDraft.value = monthlyBudgetCNY.value
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '保存月预算失败'))
  } finally {
    budgetSaving.value = false
  }
}

async function loadBackupStale() {
  backupStale.value = false
  backupStaleMessage.value = ''
  try {
    const res = await adminAPI.dataManagement.listBackupJobs({ page_size: 20 })
    const items = res.items || []
    const succeeded = items.filter((job) => job.status === 'succeeded')
    if (!succeeded.length) {
      backupStale.value = true
      backupStaleMessage.value = '超过 7 天未检测到成功备份（或尚无备份记录）'
      return
    }
    const latest = succeeded
      .map((job) => new Date(job.finished_at || job.started_at || 0).getTime())
      .filter((ts) => Number.isFinite(ts) && ts > 0)
      .sort((a, b) => b - a)[0]
    if (!latest) {
      backupStale.value = true
      backupStaleMessage.value = '超过 7 天未检测到成功备份（或尚无备份记录）'
      return
    }
    const ageMs = Date.now() - latest
    if (ageMs > 7 * 24 * 60 * 60 * 1000) {
      const days = Math.floor(ageMs / (24 * 60 * 60 * 1000))
      backupStale.value = true
      backupStaleMessage.value = `最近一次成功备份已过去 ${days} 天`
    }
  } catch {
    // 数据管理未配置/无权限时静默，不挡总览
  }
}

async function loadAll() {
  loading.value = true
  const { start, end } = getConsoleRange(rangeKey.value)
  try {
    const [trendRes, modelsRes, rankingRes, budgetRes] = await Promise.all([
      adminAPI.dashboard.getUsageTrend({ start_date: start, end_date: end, granularity: 'day' }),
      adminAPI.dashboard.getModelStats({ start_date: start, end_date: end }),
      adminAPI.dashboard.getUserSpendingRanking({ start_date: start, end_date: end, limit: 20 }),
      adminAPI.dashboard.getStats(),
    ])
    monthlyBudgetCNY.value = Number(budgetRes.monthly_budget_cny || 0)
    monthlySpendCNY.value = Number(budgetRes.monthly_spend_cny || 0)
    monthlyBudgetUsagePercent.value = Number(budgetRes.monthly_budget_usage_percent || 0)
    budgetDraft.value = monthlyBudgetCNY.value
    trend.value = trendRes.trend || []
    models.value = modelsRes.models || []
    ranking.value = rankingRes.ranking || []
    rankingTotals.value = {
      actual_cost: rankingRes.total_actual_cost || 0,
      requests: rankingRes.total_requests || 0,
      tokens: rankingRes.total_tokens || 0,
    }
    void loadBackupStale()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载总览数据失败'))
  } finally {
    loading.value = false
  }
}

onMounted(loadAll)
</script>
