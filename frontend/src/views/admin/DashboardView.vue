<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- Header: page identity + the single teal primary action -->
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 class="ui-heading">{{ t('admin.dashboard.title') }}</h1>
          <p class="ui-subheading">{{ t('admin.dashboard.description') }}</p>
        </div>
        <button
          type="button"
          class="btn bg-ui-accent text-white hover:opacity-90 dark:text-dark-950"
          :disabled="chartsLoading"
          @click="loadDashboardStats"
        >
          <Icon name="refresh" size="sm" :stroke-width="2" />
          {{ t('common.refresh') }}
        </button>
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="flex items-center justify-center py-12">
        <LoadingSpinner />
      </div>

      <template v-else-if="stats">
        <!-- 1. Summary: reconciliation conclusion with its honest pending state -->
        <section class="ui-panel p-5" data-testid="dashboard-summary">
          <div class="flex flex-wrap items-center gap-3">
            <p class="text-sm font-medium text-ui-text">
              {{ t('admin.providerBilling.bossStrip') }}
            </p>
            <template v-if="billingConclusions.length">
              <span
                v-for="(item, idx) in billingConclusions"
                :key="idx"
                class="rounded px-2 py-1 text-xs font-medium"
                :class="conclusionClass(item.conclusion)"
              >
                <span v-if="item.provider">{{ item.provider }} · </span>{{ conclusionLabel(item.conclusion) }}
              </span>
            </template>
            <span
              v-else
              class="rounded bg-gray-100 px-2 py-1 text-xs text-gray-600 dark:bg-dark-700 dark:text-gray-300"
            >
              {{ t('admin.providerBilling.conclusionNotUploaded') }}
            </span>
          </div>
        </section>

        <!-- 2. KPIs: four primary numbers, secondary metrics visually quieter -->
        <section data-testid="dashboard-kpis">
          <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <!-- Today Requests -->
            <div class="ui-panel p-4">
              <p class="text-xs font-medium text-ui-text-muted">
                {{ t('admin.dashboard.todayRequests') }}
              </p>
              <p class="mt-1 text-2xl font-semibold tabular-nums text-ui-text">
                {{ formatNumber(stats.today_requests) }}
              </p>
              <p class="mt-1 text-xs tabular-nums text-ui-text-muted">
                {{ t('common.total') }}: {{ formatNumber(stats.total_requests) }}
              </p>
            </div>

            <!-- Today Tokens -->
            <div class="ui-panel p-4">
              <p class="text-xs font-medium text-ui-text-muted">
                {{ t('admin.dashboard.todayTokens') }}
              </p>
              <p class="mt-1 text-2xl font-semibold tabular-nums text-ui-text">
                {{ formatTokens(stats.today_tokens) }}
              </p>
              <p class="mt-1 text-xs tabular-nums">
                <span
                  class="text-emerald-600 dark:text-emerald-400"
                  :title="t('admin.dashboard.actual')"
                  >${{ formatCost(stats.today_actual_cost) }}</span
                >
                <span class="text-ui-text-muted"> / </span>
                <span
                  class="text-orange-500 dark:text-orange-400"
                  :title="t('admin.dashboard.accountCost')"
                  >${{ formatCost(stats.today_account_cost) }}</span
                >
                <span class="text-ui-text-muted"> / </span>
                <span class="text-ui-text-muted" :title="t('admin.dashboard.standard')"
                  >${{ formatCost(stats.today_cost) }}</span
                >
              </p>
            </div>

            <!-- Accounts -->
            <div class="ui-panel p-4">
              <p class="text-xs font-medium text-ui-text-muted">
                {{ t('admin.dashboard.accounts') }}
              </p>
              <p class="mt-1 text-2xl font-semibold tabular-nums text-ui-text">
                {{ formatNumber(stats.total_accounts) }}
              </p>
              <p class="mt-1 text-xs tabular-nums">
                <span class="text-emerald-600 dark:text-emerald-400"
                  >{{ stats.normal_accounts }} {{ t('common.active') }}</span
                >
                <span v-if="stats.error_accounts > 0" class="ml-1 text-red-500"
                  >{{ stats.error_accounts }} {{ t('common.error') }}</span
                >
              </p>
            </div>

            <!-- Avg Response -->
            <div class="ui-panel p-4">
              <p class="text-xs font-medium text-ui-text-muted">
                {{ t('admin.dashboard.avgResponse') }}
              </p>
              <p class="mt-1 text-2xl font-semibold tabular-nums text-ui-text">
                {{ formatDuration(stats.average_duration_ms) }}
              </p>
              <p class="mt-1 text-xs tabular-nums text-ui-text-muted">
                {{ stats.active_users }} {{ t('admin.dashboard.activeUsers') }}
              </p>
            </div>
          </div>

          <!-- Secondary metrics: quieter presentation -->
          <div class="ui-panel mt-4 p-4">
            <p class="text-xs font-medium uppercase tracking-wide text-ui-text-muted">
              {{ t('admin.dashboard.moreMetrics') }}
            </p>
            <div class="mt-3 grid grid-cols-2 gap-x-4 gap-y-5 lg:grid-cols-4">
              <!-- API Keys -->
              <div>
                <p class="text-xs text-ui-text-muted">{{ t('admin.dashboard.apiKeys') }}</p>
                <p class="mt-0.5 text-lg font-medium tabular-nums text-ui-text">
                  {{ formatNumber(stats.total_api_keys) }}
                </p>
                <p class="text-xs tabular-nums text-ui-text-muted">
                  {{ stats.active_api_keys }} {{ t('common.active') }}
                </p>
              </div>

              <!-- New Users Today -->
              <div>
                <p class="text-xs text-ui-text-muted">{{ t('admin.dashboard.users') }}</p>
                <p class="mt-0.5 text-lg font-medium tabular-nums text-ui-text">
                  +{{ formatNumber(stats.today_new_users) }}
                </p>
                <p class="text-xs tabular-nums text-ui-text-muted">
                  {{ t('common.total') }}: {{ formatNumber(stats.total_users) }}
                </p>
              </div>

              <!-- Total Tokens -->
              <div>
                <p class="text-xs text-ui-text-muted">{{ t('admin.dashboard.totalTokens') }}</p>
                <p class="mt-0.5 text-lg font-medium tabular-nums text-ui-text">
                  {{ formatTokens(stats.total_tokens) }}
                </p>
                <p class="text-xs tabular-nums">
                  <span
                    class="text-emerald-600/80 dark:text-emerald-400/80"
                    :title="t('admin.dashboard.actual')"
                    >${{ formatCost(stats.total_actual_cost) }}</span
                  >
                  <span class="text-ui-text-muted"> / </span>
                  <span
                    class="text-orange-500/80 dark:text-orange-400/80"
                    :title="t('admin.dashboard.accountCost')"
                    >${{ formatCost(stats.total_account_cost) }}</span
                  >
                  <span class="text-ui-text-muted"> / </span>
                  <span class="text-ui-text-muted" :title="t('admin.dashboard.standard')"
                    >${{ formatCost(stats.total_cost) }}</span
                  >
                </p>
              </div>

              <!-- Performance (RPM/TPM) -->
              <div>
                <p class="text-xs text-ui-text-muted">{{ t('admin.dashboard.performance') }}</p>
                <p class="mt-0.5 text-lg font-medium tabular-nums text-ui-text">
                  {{ formatTokens(stats.rpm) }}
                  <span class="text-xs font-normal text-ui-text-muted">RPM</span>
                </p>
                <p class="text-xs tabular-nums text-ui-text-muted">
                  {{ formatTokens(stats.tpm) }} TPM
                </p>
              </div>
            </div>
          </div>
        </section>

        <!-- 3. Attention: exceptions first, then next actions -->
        <section class="ui-panel p-5" data-testid="dashboard-attention">
          <h2 class="text-sm font-semibold text-ui-text">
            {{ t('admin.dashboard.attention') }}
          </h2>
          <ul v-if="attentionItems.length" class="mt-3 space-y-2">
            <li
              v-for="(item, idx) in attentionItems"
              :key="idx"
              class="flex items-center gap-2 text-sm text-ui-text"
            >
              <Icon :name="item.icon" size="sm" :class="item.iconClass" />
              <span class="min-w-0 flex-1">{{ item.text }}</span>
              <button
                v-if="item.to"
                type="button"
                class="shrink-0 text-xs font-medium text-ui-accent hover:underline"
                @click="item.to && router.push(item.to)"
              >
                {{ item.actionLabel }}
              </button>
            </li>
          </ul>
          <p v-else class="mt-3 flex items-center gap-2 text-sm text-ui-text-muted">
            <Icon name="checkCircle" size="sm" class="text-emerald-500" />
            {{ t('admin.dashboard.attentionEmpty') }}
          </p>

          <div class="mt-4 border-t border-ui-border pt-4">
            <p class="text-xs font-medium uppercase tracking-wide text-ui-text-muted">
              {{ t('admin.dashboard.nextActions') }}
            </p>
            <div class="mt-3 grid grid-cols-1 gap-3 md:grid-cols-2">
              <button
                v-if="canUseBatchImage"
                type="button"
                class="group flex items-center gap-3 rounded-xl border border-ui-border bg-ui-surface-raised p-3 text-left transition-colors hover:border-ui-accent"
                @click="router.push('/batch-image')"
              >
                <Icon
                  name="sparkles"
                  size="md"
                  :stroke-width="2"
                  class="shrink-0 text-ui-text-muted transition-colors group-hover:text-ui-accent"
                />
                <span class="min-w-0 flex-1">
                  <span class="block text-sm font-medium text-ui-text">
                    {{ t('admin.dashboard.batchImage') }}
                  </span>
                  <span class="block text-xs text-ui-text-muted">
                    {{ t('admin.dashboard.batchImageDesc') }}
                  </span>
                </span>
                <Icon
                  name="chevronRight"
                  size="sm"
                  class="text-ui-text-muted transition-colors group-hover:text-ui-accent"
                />
              </button>
              <button
                type="button"
                class="group flex items-center gap-3 rounded-xl border border-ui-border bg-ui-surface-raised p-3 text-left transition-colors hover:border-ui-accent"
                @click="router.push('/admin/groups')"
              >
                <Icon
                  name="grid"
                  size="md"
                  :stroke-width="2"
                  class="shrink-0 text-ui-text-muted transition-colors group-hover:text-ui-accent"
                />
                <span class="min-w-0 flex-1">
                  <span class="block text-sm font-medium text-ui-text">
                    {{ t('admin.dashboard.groupPricing') }}
                  </span>
                  <span class="block text-xs text-ui-text-muted">
                    {{ t('admin.dashboard.groupPricingDesc') }}
                  </span>
                </span>
                <Icon
                  name="chevronRight"
                  size="sm"
                  class="text-ui-text-muted transition-colors group-hover:text-ui-accent"
                />
              </button>
            </div>
          </div>
        </section>

        <!-- 4. Trends & rankings: below the decision layer -->
        <section class="space-y-4" data-testid="dashboard-trends">
          <h2 class="text-sm font-semibold text-ui-text">
            {{ t('admin.dashboard.trendsAndRankings') }}
          </h2>

          <!-- Date Range Filter -->
          <div class="ui-toolbar flex-wrap gap-4">
            <div class="flex items-center gap-2">
              <span class="text-sm font-medium text-ui-text">{{ t('admin.dashboard.timeRange') }}:</span>
              <DateRangePicker
                v-model:start-date="startDate"
                v-model:end-date="endDate"
                @change="onDateRangeChange"
              />
            </div>
            <div class="ml-auto flex items-center gap-2">
              <span class="text-sm font-medium text-ui-text">{{ t('admin.dashboard.granularity') }}:</span>
              <div class="w-28">
                <Select
                  v-model="granularity"
                  :options="granularityOptions"
                  @change="loadChartData"
                />
              </div>
            </div>
          </div>

          <!-- Charts Grid -->
          <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
            <ModelDistributionChart
              :model-stats="modelStats"
              :enable-ranking-view="true"
              :ranking-items="rankingItems"
              :ranking-total-actual-cost="rankingTotalActualCost"
              :ranking-total-requests="rankingTotalRequests"
              :ranking-total-tokens="rankingTotalTokens"
              :loading="chartsLoading"
              :ranking-loading="rankingLoading"
              :ranking-error="rankingError"
              :start-date="startDate"
              :end-date="endDate"
              @ranking-click="goToUserUsage"
            />
            <TokenUsageTrend :trend-data="trendData" :loading="chartsLoading" />
          </div>

          <!-- User Usage Trend (Full Width) -->
          <div class="ui-panel p-4">
            <h3 class="mb-4 text-sm font-semibold text-ui-text">
              {{ t('admin.dashboard.recentUsage') }} (Top 12)
            </h3>
            <div class="h-64">
              <div v-if="userTrendLoading" class="flex h-full items-center justify-center">
                <LoadingSpinner size="md" />
              </div>
              <Line v-else-if="userTrendChartData" :data="userTrendChartData" :options="lineOptions" />
              <div
                v-else
                class="flex h-full items-center justify-center text-sm text-ui-text-muted"
              >
                {{ t('admin.dashboard.noDataAvailable') }}
              </div>
            </div>
          </div>
        </section>
      </template>

      <!-- Explicit empty state: stats failed to load -->
      <div v-else-if="statsLoadFailed" class="ui-panel">
        <AnimatedEmptyState
          variant="generic"
          :title="t('admin.dashboard.emptyTitle')"
          :description="t('admin.dashboard.emptyDescription')"
        />
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
import { adminAPI } from '@/api/admin'
import type {
  DashboardStats,
  TrendDataPoint,
  ModelStat,
  UserUsageTrendPoint,
  UserSpendingRankingItem
} from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import AnimatedEmptyState from '@/components/common/AnimatedEmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Select from '@/components/common/Select.vue'
import ModelDistributionChart from '@/components/charts/ModelDistributionChart.vue'
import TokenUsageTrend from '@/components/charts/TokenUsageTrend.vue'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'

import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'
import { Line } from 'vue-chartjs'

// Register Chart.js components
ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
)

const appStore = useAppStore()
const router = useRouter()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()
const stats = ref<DashboardStats | null>(null)
const loading = ref(false)
const statsLoadFailed = ref(false)
const chartsLoading = ref(false)
const userTrendLoading = ref(false)
const rankingLoading = ref(false)
const rankingError = ref(false)
const billingConclusions = ref<Array<{ provider?: string; conclusion: string }>>([])

// Chart data
const trendData = ref<TrendDataPoint[]>([])
const modelStats = ref<ModelStat[]>([])
const userTrend = ref<UserUsageTrendPoint[]>([])
const rankingItems = ref<UserSpendingRankingItem[]>([])
const rankingTotalActualCost = ref(0)
const rankingTotalRequests = ref(0)
const rankingTotalTokens = ref(0)
let chartLoadSeq = 0
let usersTrendLoadSeq = 0
let rankingLoadSeq = 0
const rankingLimit = 12

// Helper function to format date in local timezone
const formatLocalDate = (date: Date): string => {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

const getLast24HoursRangeDates = (): { start: string; end: string } => {
  const end = new Date()
  const start = new Date(end.getTime() - 24 * 60 * 60 * 1000)
  return {
    start: formatLocalDate(start),
    end: formatLocalDate(end)
  }
}

// Date range
const granularity = ref<'day' | 'hour'>('hour')
const defaultRange = getLast24HoursRangeDates()
const startDate = ref(defaultRange.start)
const endDate = ref(defaultRange.end)

// Granularity options for Select component
const granularityOptions = computed(() => [
  { value: 'day', label: t('admin.dashboard.day') },
  { value: 'hour', label: t('admin.dashboard.hour') }
])

// Dark mode detection
const isDarkMode = computed(() => {
  return document.documentElement.classList.contains('dark')
})

// Chart colors
const chartColors = computed(() => ({
  text: isDarkMode.value ? '#e5e7eb' : '#374151',
  grid: isDarkMode.value ? '#374151' : '#e5e7eb'
}))

// Line chart options (for user trend chart)
const lineOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: {
    intersect: false,
    mode: 'index' as const
  },
  plugins: {
    legend: {
      position: 'top' as const,
      labels: {
        color: chartColors.value.text,
        usePointStyle: true,
        pointStyle: 'circle',
        padding: 15,
        font: {
          size: 11
        }
      }
    },
    tooltip: {
      itemSort: (a: any, b: any) => {
        const aValue = typeof a?.raw === 'number' ? a.raw : Number(a?.parsed?.y ?? 0)
        const bValue = typeof b?.raw === 'number' ? b.raw : Number(b?.parsed?.y ?? 0)
        return bValue - aValue
      },
      callbacks: {
        label: (context: any) => {
          return `${context.dataset.label}: ${formatTokens(context.raw)}`
        }
      }
    }
  },
  scales: {
    x: {
      grid: {
        color: chartColors.value.grid
      },
      ticks: {
        color: chartColors.value.text,
        font: {
          size: 10
        }
      }
    },
    y: {
      grid: {
        color: chartColors.value.grid
      },
      ticks: {
        color: chartColors.value.text,
        font: {
          size: 10
        },
        callback: (value: string | number) => formatTokens(Number(value))
      }
    }
  }
}))

// User trend chart data
const userTrendChartData = computed(() => {
  if (!userTrend.value?.length) return null

  const getDisplayName = (point: UserUsageTrendPoint): string => {
    const username = point.username?.trim()
    if (username) {
      return username
    }

    const email = point.email?.trim()
    if (email) {
      return email
    }

    return t('admin.redeem.userPrefix', { id: point.user_id })
  }

  // Group by user_id to avoid merging different users with the same display name
  const userGroups = new Map<number, { name: string; data: Map<string, number> }>()
  const allDates = new Set<string>()

  userTrend.value.forEach((point) => {
    allDates.add(point.date)
    const key = point.user_id
    if (!userGroups.has(key)) {
      userGroups.set(key, { name: getDisplayName(point), data: new Map() })
    }
    userGroups.get(key)!.data.set(point.date, point.tokens)
  })

  const sortedDates = Array.from(allDates).sort()
  const colors = [
    '#3b82f6',
    '#10b981',
    '#f59e0b',
    '#ef4444',
    '#8b5cf6',
    '#ec4899',
    '#14b8a6',
    '#f97316',
    '#6366f1',
    '#84cc16',
    '#06b6d4',
    '#a855f7'
  ]

  const datasets = Array.from(userGroups.values()).map((group, idx) => ({
    label: group.name,
    data: sortedDates.map((date) => group.data.get(date) || 0),
    borderColor: colors[idx % colors.length],
    backgroundColor: `${colors[idx % colors.length]}20`,
    fill: false,
    tension: 0.3
  }))

  return {
    labels: sortedDates,
    datasets
  }
})

// Format helpers
const formatTokens = (value: number | undefined): string => {
  if (value === undefined || value === null) return '0'
  if (value >= 1_000_000_000) {
    return `${(value / 1_000_000_000).toFixed(2)}B`
  } else if (value >= 1_000_000) {
    return `${(value / 1_000_000).toFixed(2)}M`
  } else if (value >= 1_000) {
    return `${(value / 1_000).toFixed(2)}K`
  }
  return value.toLocaleString()
}

const toFiniteNumber = (value: unknown): number => {
  const numberValue = Number(value)
  return Number.isFinite(numberValue) ? numberValue : 0
}

const formatNumber = (value: number | null | undefined): string => {
  return toFiniteNumber(value).toLocaleString()
}

const formatCost = (value: number | null | undefined): string => {
  const safeValue = toFiniteNumber(value)
  if (safeValue >= 1000) {
    return (safeValue / 1000).toFixed(2) + 'K'
  } else if (safeValue >= 1) {
    return safeValue.toFixed(2)
  } else if (safeValue >= 0.01) {
    return safeValue.toFixed(3)
  }
  return safeValue.toFixed(4)
}

const formatDuration = (ms: number): string => {
  if (ms >= 1000) {
    return `${(ms / 1000).toFixed(2)}s`
  }
  return `${Math.round(ms)}ms`
}

const goToUserUsage = (item: UserSpendingRankingItem) => {
  void router.push({
    path: '/admin/usage',
    query: {
      user_id: String(item.user_id),
      start_date: startDate.value,
      end_date: endDate.value
    }
  })
}

// Date range change handler
const onDateRangeChange = (range: {
  startDate: string
  endDate: string
  preset: string | null
}) => {
  // Auto-select granularity based on date range
  const start = new Date(range.startDate)
  const end = new Date(range.endDate)
  const daysDiff = Math.ceil((end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24))

  // If range is 1 day, use hourly granularity
  if (daysDiff <= 1) {
    granularity.value = 'hour'
  } else {
    granularity.value = 'day'
  }

  loadChartData()
}

// Load data
const loadDashboardSnapshot = async (includeStats: boolean) => {
  const currentSeq = ++chartLoadSeq
  if (includeStats && !stats.value) {
    loading.value = true
  }
  chartsLoading.value = true
  try {
    const response = await adminAPI.dashboard.getSnapshotV2({
      start_date: startDate.value,
      end_date: endDate.value,
      granularity: granularity.value,
      include_stats: includeStats,
      include_trend: true,
      include_model_stats: true,
      include_group_stats: false,
      include_users_trend: false
    })
    if (currentSeq !== chartLoadSeq) return
    if (includeStats && response.stats) {
      stats.value = response.stats
      statsLoadFailed.value = false
    }
    trendData.value = response.trend || []
    modelStats.value = response.models || []
  } catch (error) {
    if (currentSeq !== chartLoadSeq) return
    if (includeStats) {
      statsLoadFailed.value = true
    }
    appStore.showError(t('admin.dashboard.failedToLoad'))
    console.error('Error loading dashboard snapshot:', error)
  } finally {
    if (currentSeq === chartLoadSeq) {
      loading.value = false
      chartsLoading.value = false
    }
  }
}

const loadUsersTrend = async () => {
  const currentSeq = ++usersTrendLoadSeq
  userTrendLoading.value = true
  try {
    const response = await adminAPI.dashboard.getUserUsageTrend({
      start_date: startDate.value,
      end_date: endDate.value,
      granularity: granularity.value,
      limit: 12
    })
    if (currentSeq !== usersTrendLoadSeq) return
    userTrend.value = response.trend || []
  } catch (error) {
    if (currentSeq !== usersTrendLoadSeq) return
    console.error('Error loading users trend:', error)
    userTrend.value = []
  } finally {
    if (currentSeq === usersTrendLoadSeq) {
      userTrendLoading.value = false
    }
  }
}

const loadUserSpendingRanking = async () => {
  const currentSeq = ++rankingLoadSeq
  rankingLoading.value = true
  rankingError.value = false
  try {
    const response = await adminAPI.dashboard.getUserSpendingRanking({
      start_date: startDate.value,
      end_date: endDate.value,
      limit: rankingLimit
    })
    if (currentSeq !== rankingLoadSeq) return
    rankingItems.value = response.ranking || []
    rankingTotalActualCost.value = response.total_actual_cost || 0
    rankingTotalRequests.value = response.total_requests || 0
    rankingTotalTokens.value = response.total_tokens || 0
  } catch (error) {
    if (currentSeq !== rankingLoadSeq) return
    console.error('Error loading user spending ranking:', error)
    rankingItems.value = []
    rankingTotalActualCost.value = 0
    rankingTotalRequests.value = 0
    rankingTotalTokens.value = 0
    rankingError.value = true
  } finally {
    if (currentSeq === rankingLoadSeq) {
      rankingLoading.value = false
    }
  }
}

const loadDashboardStats = async () => {
  await Promise.all([
    loadDashboardSnapshot(true),
    loadUsersTrend(),
    loadUserSpendingRanking(),
    loadBillingConclusions()
  ])
}

const loadChartData = async () => {
  await Promise.all([
    loadDashboardSnapshot(false),
    loadUsersTrend(),
    loadUserSpendingRanking()
  ])
}

const loadBillingConclusions = async () => {
  try {
    const response = await adminAPI.providerBilling.getBossConclusions()
    billingConclusions.value = (response.items || []).map((item) => ({
      provider: item.provider,
      conclusion: item.conclusion
    }))
  } catch {
    billingConclusions.value = [{ conclusion: 'not_uploaded' }]
  }
}

const conclusionLabel = (conclusion: string) => {
  switch (conclusion) {
    case 'reconciled':
      return t('admin.providerBilling.conclusionReconciled')
    case 'has_diff':
      return t('admin.providerBilling.conclusionHasDiff')
    case 'not_uploaded':
      return t('admin.providerBilling.conclusionNotUploaded')
    default:
      return conclusion
  }
}

const conclusionClass = (conclusion: string) => {
  switch (conclusion) {
    case 'reconciled':
      return 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-200'
    case 'has_diff':
      return 'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-200'
    case 'not_uploaded':
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
    default:
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
  }
}

// Honest pending state: reconciliation is pending when no provider statement
// has been uploaded yet (or the conclusions request failed, which falls back
// to the same not_uploaded conclusion).
const billingPending = computed(() => {
  if (!billingConclusions.value.length) return true
  return billingConclusions.value.some((item) => item.conclusion === 'not_uploaded')
})

interface AttentionItem {
  icon: 'exclamationTriangle' | 'exclamationCircle' | 'infoCircle'
  iconClass: string
  text: string
  to?: string
  actionLabel?: string
}

// Exceptions that deserve the boss's attention, plus the next action for each.
const attentionItems = computed<AttentionItem[]>(() => {
  const items: AttentionItem[] = []
  const errorAccounts = stats.value?.error_accounts ?? 0
  if (errorAccounts > 0) {
    items.push({
      icon: 'exclamationTriangle',
      iconClass: 'text-red-500',
      text: t('admin.dashboard.errorAccountsAttention', { count: errorAccounts }),
      to: '/admin/accounts',
      actionLabel: t('admin.dashboard.manageAccounts')
    })
  }
  if (billingPending.value) {
    items.push({
      icon: 'infoCircle',
      iconClass: 'text-ui-text-muted',
      text: t('admin.dashboard.billingPendingHint'),
      to: '/admin/provider-billing',
      actionLabel: t('admin.dashboard.openReconciliation')
    })
  }
  if (rankingError.value) {
    items.push({
      icon: 'exclamationCircle',
      iconClass: 'text-amber-500',
      text: t('admin.dashboard.rankingUnavailable')
    })
  }
  return items
})

onMounted(() => {
  void refreshBatchImageAccess()
  loadDashboardStats()
})
</script>

<style scoped>
</style>
