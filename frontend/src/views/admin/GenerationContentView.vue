<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- 标题 + is_live 二态徽标 -->
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h1 class="text-xl font-bold text-gray-900 dark:text-white">
            {{ t('admin.generationContent.title') }}
          </h1>
          <p class="mt-0.5 text-sm text-gray-500 dark:text-dark-400">
            {{ t('admin.generationContent.description') }}
          </p>
        </div>
        <span v-if="isLive" class="badge badge-success">{{
          t('admin.generationContent.live')
        }}</span>
        <span v-else class="badge badge-gray">{{ t('admin.generationContent.exampleOff') }}</span>
      </div>

      <!-- 快照卡:6 个英雄指标(空态降透明,绝不伪造数字) -->
      <div
        class="grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-6"
        :class="{ 'opacity-60': !isLive }"
      >
        <StatCard
          :title="t('admin.generationContent.capturedToday')"
          :value="stats?.captured_today ?? 0"
          :format-value="fmtNum"
          icon-variant="primary"
        />
        <StatCard
          :title="t('admin.generationContent.capturedWeek')"
          :value="stats?.captured_week ?? 0"
          :format-value="fmtNum"
          icon-variant="primary"
        />
        <StatCard
          :title="t('admin.generationContent.distinctEmployees')"
          :value="stats?.distinct_employees ?? 0"
          :format-value="fmtNum"
          icon-variant="success"
        />
        <StatCard
          :title="t('admin.generationContent.distinctTeams')"
          :value="stats?.distinct_teams ?? 0"
          :format-value="fmtNum"
          icon-variant="success"
        />
        <StatCard
          :title="t('admin.generationContent.distinctModels')"
          :value="stats?.distinct_models ?? 0"
          :format-value="fmtNum"
          icon-variant="warning"
        />
        <StatCard
          :title="t('admin.generationContent.totalBytes')"
          :value="stats?.total_bytes ?? 0"
          :format-value="fmtBytes"
          icon-variant="danger"
        />
      </div>

      <!-- 7 日趋势 sparkline + 日均速率 -->
      <div class="card flex items-center gap-4 p-4" :class="{ 'opacity-60': !isLive }">
        <CaptureSparkline :series="dailySeries" :width="160" :height="40" />
        <div class="flex flex-col">
          <span class="stat-trend stat-trend-up text-base"
            >+{{ dailyRateText }}/{{ t('admin.generationContent.perDay') }}</span
          >
          <span class="text-xs text-gray-400">{{ t('admin.generationContent.trendCaption') }}</span>
        </div>
      </div>

      <!-- 真实脱敏样本墙 -->
      <div v-if="weeklyReport" class="card space-y-4 p-4" :class="{ 'opacity-60': !isLive }">
        <div class="grid grid-cols-2 gap-3 md:grid-cols-5">
          <StatCard
            :title="t('admin.generationContent.weeklyEntries')"
            :value="weeklyReport.entries"
            :format-value="fmtNum"
            icon-variant="primary"
          />
          <StatCard
            :title="t('admin.generationContent.weeklyCost')"
            :value="weeklyReport.total_cost_estimate"
            :format-value="fmtCurrency"
            icon-variant="warning"
          />
          <StatCard
            :title="t('admin.generationContent.adopted')"
            :value="weeklyReport.adopted_count"
            :format-value="fmtNum"
            icon-variant="success"
          />
          <StatCard
            :title="t('admin.generationContent.pending')"
            :value="weeklyReport.pending_count + weeklyReport.unreviewed_count"
            :format-value="fmtNum"
            icon-variant="primary"
          />
          <StatCard
            :title="t('admin.generationContent.anomalies')"
            :value="weeklyAnomalyCount"
            :format-value="fmtNum"
            icon-variant="danger"
          />
        </div>
        <pre
          class="max-h-64 overflow-auto whitespace-pre-wrap rounded border border-gray-100 bg-gray-50 p-3 text-xs text-gray-600 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-300"
        >{{ weeklyReport.markdown }}</pre>
      </div>

      <ContentWall :samples="samples" :is-live="isLive" @updated="load" />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import StatCard from '@/components/common/StatCard.vue'
import CaptureSparkline from '@/components/admin/generation-content/CaptureSparkline.vue'
import ContentWall from '@/components/admin/generation-content/ContentWall.vue'
import { adminAPI } from '@/api/admin'
import type {
  GenerationContentStats,
  GenerationContentWeeklyReport,
  GenerationSample
} from '@/api/admin/generation_content'
import { formatBytes, formatCompactNumber, formatCurrency } from '@/utils/format'

const { t } = useI18n()

const stats = ref<GenerationContentStats | null>(null)
const samples = ref<GenerationSample[]>([])
const weeklyReport = ref<GenerationContentWeeklyReport | null>(null)
const loading = ref(false)
let abortController: AbortController | null = null

const isLive = computed(() => stats.value?.is_live ?? false)
const dailySeries = computed(() => (stats.value?.daily_series ?? []).map((p) => p.count))
const dailyRateText = computed(() => (stats.value?.daily_rate ?? 0).toFixed(1))
const weeklyAnomalyCount = computed(() => {
  const anomalies = weeklyReport.value?.anomalies
  if (!anomalies) return 0
  return anomalies.failed_tasks + anomalies.missing_task_joins + anomalies.truncated_rows
})

const fmtNum = (v: number | string) => formatCompactNumber(Number(v))
const fmtBytes = (v: number | string) => formatBytes(Number(v))
const fmtCurrency = (v: number | string) => formatCurrency(Number(v))

const load = async () => {
  abortController?.abort()
  const c = new AbortController()
  abortController = c
  loading.value = true
  try {
    const [s, sm, wr] = await Promise.all([
      adminAPI.generationContent.getStats({ signal: c.signal }),
      adminAPI.generationContent.getSamples({ signal: c.signal }),
      adminAPI.generationContent.getWeeklyReport({ signal: c.signal })
    ])
    if (c.signal.aborted) return
    stats.value = s
    samples.value = sm.samples ?? []
    weeklyReport.value = wr
  } catch (e: unknown) {
    // 取消请求(切走页面)不算错误
    if ((e as { name?: string })?.name !== 'AbortError' && (e as { code?: string })?.code !== 'ERR_CANCELED') {
      console.error('Failed to load generation content:', e)
    }
  } finally {
    if (abortController === c) loading.value = false
  }
}

onMounted(load)
onUnmounted(() => abortController?.abort())
</script>
