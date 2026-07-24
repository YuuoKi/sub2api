<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 border-b border-gray-200 pb-4 dark:border-dark-700 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 class="ui-heading">总览</h1>
          <p class="ui-subheading mt-1">消费与调用概览</p>
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

      <!-- 核心指标：紧凑数字行，无装饰圆环 -->
      <section class="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
        <div
          v-for="card in statCards"
          :key="card.label"
          class="rounded-lg border border-gray-200 bg-white px-4 py-3 dark:border-dark-700 dark:bg-dark-800"
        >
          <div class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ card.label }}</div>
          <div
            class="mt-1.5 truncate text-2xl font-semibold tabular-nums"
            :class="card.tone === 'red' ? 'text-red-600 dark:text-red-400' : card.tone === 'amber' ? 'text-amber-600 dark:text-amber-400' : 'text-gray-900 dark:text-white'"
          >
            <AnimatedNumber :value="card.value" :format="card.format" />
          </div>
          <div class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ card.hint }}</div>
        </div>
      </section>

      <!-- 花费趋势 + 模型分布；无数据时整块换成紧凑三步上手引导卡 -->
      <section v-if="showGuideCard" class="rounded-lg border border-gray-200 bg-white px-5 py-4 dark:border-dark-700 dark:bg-dark-800">
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">三步上手</h2>
        <div class="mt-3 flex flex-col gap-3 sm:flex-row sm:items-center sm:gap-6">
          <RouterLink class="flex items-center gap-2 text-sm text-gray-700 hover:text-gray-900 dark:text-gray-200 dark:hover:text-white" to="/admin/console/key-vault">
            <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-gray-100 text-xs font-semibold text-gray-600 dark:bg-dark-700 dark:text-gray-300">1</span>
            录入 AI 账号
          </RouterLink>
          <RouterLink class="flex items-center gap-2 text-sm text-gray-700 hover:text-gray-900 dark:text-gray-200 dark:hover:text-white" to="/admin/console/staff">
            <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-gray-100 text-xs font-semibold text-gray-600 dark:bg-dark-700 dark:text-gray-300">2</span>
            给员工开卡
          </RouterLink>
          <span class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
            <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-gray-100 text-xs font-semibold text-gray-600 dark:bg-dark-700 dark:text-gray-300">3</span>
            回到这里看消费
          </span>
        </div>
      </section>
      <section v-else-if="showLoadError" class="rounded-lg border border-gray-200 bg-white px-5 py-4 text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-400" data-test="overview-load-error">
        总览数据加载失败，点右上角「刷新」重试。
      </section>
      <div v-else class="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]">
        <section class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
          <div class="flex items-center justify-between gap-3">
            <div>
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">花费趋势</h2>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ rangeLabel }}内每天实际花费（人民币）</p>
            </div>
            <div class="text-right">
              <div class="text-xs text-gray-500 dark:text-gray-400">本期合计</div>
              <div class="text-lg font-semibold text-gray-900 dark:text-white">{{ formatMoney(totalSpend, usdCnyRate) }}</div>
            </div>
          </div>
          <div class="mt-4 h-52">
            <Line v-if="trendChartData" :data="trendChartData" :options="trendChartOptions" />
            <div v-else class="flex h-full items-center justify-center text-sm text-gray-500 dark:text-gray-400">
              {{ loading ? '加载中…' : '本期还没有调用记录。' }}
            </div>
          </div>
        </section>

        <section class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">用了什么 AI</h2>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">按实际花费统计的模型分布</p>
          <!-- 单色占比条 + 列表，不再使用多色饼图；Top 6 之外合并为「其他」 -->
          <div v-if="topModels.length" class="mt-4 space-y-2">
            <button
              v-for="model in topModels"
              :key="model.model"
              type="button"
              class="block w-full text-left"
              @click="goAiRecordsByModel(model.model)"
            >
              <div class="flex items-baseline justify-between gap-2 text-xs">
                <span class="min-w-0 flex-1 truncate text-gray-700 dark:text-gray-200" :title="model.model">{{ model.model }}</span>
                <span class="shrink-0 tabular-nums text-gray-500 dark:text-gray-400">{{ formatCount(model.requests) }} 次</span>
                <span class="shrink-0 tabular-nums font-medium text-gray-900 dark:text-white">{{ formatMoney(model.actual_cost, usdCnyRate) }}</span>
                <span class="w-12 shrink-0 text-right tabular-nums text-gray-500 dark:text-gray-400">{{ modelShare(model.actual_cost).toFixed(1) }}%</span>
              </div>
              <div class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                <div class="h-full rounded-full bg-ui-accent" :style="{ width: `${modelShare(model.actual_cost)}%` }" />
              </div>
            </button>
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
                <th class="px-5 py-2 font-medium">排名</th>
                <th class="px-5 py-2 font-medium">成员</th>
                <th class="px-5 py-2 font-medium">调用次数</th>
                <th class="px-5 py-2 font-medium">Tokens</th>
                <th class="px-5 py-2 font-medium">花费</th>
                <th class="w-1/5 px-5 py-2 font-medium">占比</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr
                v-for="(item, index) in ranking"
                :key="item.user_id"
                class="cursor-pointer hover:bg-gray-50 dark:hover:bg-dark-700/40"
                @click="goAiRecordsByUser(item.user_id)"
              >
                <td class="px-5 py-2">
                  <span class="inline-flex h-6 w-6 items-center justify-center rounded-full bg-gray-100 text-xs font-semibold text-gray-600 dark:bg-dark-700 dark:text-gray-300">
                    {{ index + 1 }}
                  </span>
                </td>
                <td class="px-5 py-2 font-medium text-gray-900 dark:text-white">
                  <span class="inline-flex items-center gap-1.5">
                    {{ item.email || `用户 #${item.user_id}` }}
                    <span v-if="item.member_type === 'tool'" class="text-xs text-gray-400 dark:text-gray-500">工具</span>
                  </span>
                </td>
                <td class="px-5 py-2 tabular-nums text-gray-700 dark:text-gray-200">{{ formatCount(item.requests) }}</td>
                <td class="px-5 py-2 tabular-nums text-gray-700 dark:text-gray-200">{{ formatTokens(item.tokens) }}</td>
                <td class="px-5 py-2 tabular-nums text-gray-900 dark:text-white">{{ formatMoney(item.actual_cost, usdCnyRate) }}</td>
                <td class="px-5 py-2">
                  <div class="flex items-center gap-2">
                    <div class="h-2 flex-1 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                      <div
                        class="h-full rounded-full bg-ui-accent"
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
                  <RouterLink class="font-medium text-gray-900 underline hover:no-underline dark:text-white" to="/admin/console/staff">成员与开卡</RouterLink>
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
  CategoryScale,
  Chart as ChartJS,
  Filler,
  Legend,
  LinearScale,
  LineElement,
  PointElement,
  Tooltip,
} from 'chart.js'
import { Line } from 'vue-chartjs'
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

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Filler, Tooltip, Legend)

const appStore = useAppStore()
const router = useRouter()

const loading = ref(false)
// 加载失败不能伪装成「未上手」空态：失败时显示中性错误提示而非三步上手卡
const loadFailed = ref(false)
const rangeKey = ref<ConsoleRangeKey>('30d')
const trend = ref<TrendDataPoint[]>([])
const models = ref<ModelStat[]>([])
const ranking = ref<UserSpendingRankingItem[]>([])
const rankingTotals = ref({ actual_cost: 0, requests: 0, tokens: 0 })
// 异常上游数：status=error 的上游账号（老板第一眼要看到的风险指标）
const errorAccounts = ref(0)
// 后端 /admin/dashboard/stats 未提供实时汇率字段；使用系统默认汇率展示，不臆造动态汇率。
const usdCnyRate = ref(DEFAULT_USD_CNY_RATE)

const rangeOptions: Array<{ key: ConsoleRangeKey; label: string }> = [
  { key: '7d', label: '近 7 天' },
  { key: '30d', label: '近 30 天' },
  { key: 'month', label: '本月' },
]

const rangeLabel = computed(() => rangeOptions.find((option) => option.key === rangeKey.value)?.label ?? '本期')

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
  tone?: 'default' | 'amber' | 'red'
}

const statCards = computed<StatCard[]>(() => {
  return [
    {
      label: `${rangeLabel.value}总花费`,
      value: totalSpend.value,
      hint: '所有成员实际扣费合计',
      format: (v) => formatMoney(v, usdCnyRate.value),
    },
    {
      label: 'AI 调用次数',
      value: totalRequests.value,
      hint: `${rangeLabel.value}全部模型调用`,
    },
    {
      label: 'Token 用量',
      value: totalTokens.value,
      hint: `${rangeLabel.value}累计 Token`,
      format: (v) => formatTokens(v),
    },
    {
      label: '活跃成员',
      value: activeStaff.value,
      hint: `${rangeLabel.value}有调用记录的成员`,
    },
    {
      label: '异常上游',
      value: errorAccounts.value,
      hint: '状态异常的上游账号数',
      tone: errorAccounts.value > 0 ? 'red' : 'default',
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

const OTHER_MODEL_LABEL = '其他'

const topModels = computed(() => {
  const sorted = [...models.value].sort((a, b) => b.actual_cost - a.actual_cost)
  if (sorted.length <= 6) return sorted
  const top = sorted.slice(0, 6)
  const rest = sorted.slice(6)
  top.push({
    model: OTHER_MODEL_LABEL,
    requests: rest.reduce((sum, model) => sum + model.requests, 0),
    input_tokens: rest.reduce((sum, model) => sum + model.input_tokens, 0),
    output_tokens: rest.reduce((sum, model) => sum + model.output_tokens, 0),
    cache_creation_tokens: rest.reduce((sum, model) => sum + model.cache_creation_tokens, 0),
    cache_read_tokens: rest.reduce((sum, model) => sum + model.cache_read_tokens, 0),
    total_tokens: rest.reduce((sum, model) => sum + model.total_tokens, 0),
    cost: rest.reduce((sum, model) => sum + model.cost, 0),
    actual_cost: rest.reduce((sum, model) => sum + model.actual_cost, 0),
  })
  return top
})

const showGuideCard = computed(() => !loading.value && !loadFailed.value && !trendChartData.value && !topModels.value.length)
const showLoadError = computed(() => !loading.value && loadFailed.value && !trendChartData.value && !topModels.value.length)

// 单色占比：占本期模型总花费的比例
function modelShare(cost: number): number {
  const total = topModels.value.reduce((sum, model) => sum + model.actual_cost, 0)
  if (total <= 0) return 0
  return Math.min(100, (cost / total) * 100)
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
  // 「其他」是聚合行，没有对应模型名，去调用记录总表
  if (trimmed === OTHER_MODEL_LABEL) {
    void router.push({ path: '/admin/console/ai-records' })
    return
  }
  void router.push({ path: '/admin/console/ai-records', query: { model: trimmed } })
}

async function loadAll() {
  loading.value = true
  const { start, end } = getConsoleRange(rangeKey.value)
  try {
    const [trendRes, modelsRes, rankingRes, accountsRes] = await Promise.all([
      adminAPI.dashboard.getUsageTrend({ start_date: start, end_date: end, granularity: 'day' }),
      adminAPI.dashboard.getModelStats({ start_date: start, end_date: end }),
      adminAPI.dashboard.getUserSpendingRanking({ start_date: start, end_date: end, limit: 20 }),
      adminAPI.accounts.list(1, 100),
    ])
    trend.value = trendRes.trend || []
    models.value = modelsRes.models || []
    ranking.value = rankingRes.ranking || []
    rankingTotals.value = {
      actual_cost: rankingRes.total_actual_cost || 0,
      requests: rankingRes.total_requests || 0,
      tokens: rankingRes.total_tokens || 0,
    }
    errorAccounts.value = (accountsRes.items || []).filter((account) => account.status === 'error').length
    loadFailed.value = false
  } catch (err) {
    loadFailed.value = true
    appStore.showError(extractApiErrorMessage(err, '加载总览数据失败'))
  } finally {
    loading.value = false
  }
}

onMounted(loadAll)
</script>
