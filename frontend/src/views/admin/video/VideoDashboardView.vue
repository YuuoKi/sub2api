<template>
  <AppLayout>
    <!-- 演示模式总览:系统状态 → 一个推荐下一步 → 辅助入口 -->
    <div v-if="isVideoGatewayDemoMode" class="space-y-6">
      <div class="flex flex-col gap-3 border-b border-ui-border pb-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 class="ui-heading">总览</h1>
        </div>
        <button class="btn btn-outline" type="button" :disabled="loading" @click="loadDashboard">
          <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
          刷新
        </button>
      </div>

      <section class="ui-panel p-5" aria-label="系统状态">
        <div class="mb-4">
          <h2 class="text-base font-semibold text-ui-text">系统状态</h2>
        </div>
        <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          <div
            v-for="item in bossStatusItems"
            :key="item.label"
            class="rounded-xl border border-ui-border bg-ui-canvas/60 p-4"
          >
            <div class="text-xs font-medium text-ui-text-muted">{{ item.label }}</div>
            <div class="mt-2 text-2xl font-semibold tabular-nums text-ui-text">{{ item.value }}</div>
            <div class="mt-1 text-xs text-ui-text-muted">{{ item.hint }}</div>
          </div>
        </div>
      </section>

      <section class="ui-panel border-l-4 border-l-teal-500 p-5" aria-label="推荐下一步">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <p class="text-sm font-medium text-teal-700 dark:text-teal-300">推荐下一步</p>
            <h2 class="mt-2 text-xl font-semibold text-ui-text">
              先试跑一条任务，确认系统能接收、处理并留下记录。
            </h2>
            <p class="ui-subheading mt-2 max-w-2xl leading-6">
              不会调用真实生成服务，结果可在任务记录查看。
            </p>
          </div>
          <RouterLink class="btn btn-primary shrink-0" to="/admin/video/create" data-testid="video-primary-action">
            <Icon name="play" size="sm" />
            试跑一条任务
          </RouterLink>
        </div>
      </section>

      <section aria-label="辅助入口">
        <div class="mb-3">
          <h2 class="text-base font-semibold text-ui-text">从哪开始</h2>
        </div>
        <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <RouterLink
            v-for="entry in bossEntryCards"
            :key="entry.title"
            :to="entry.to"
            class="ui-panel p-5 transition hover:border-teal-400 hover:shadow-md"
          >
            <Icon :name="entry.icon" size="md" class="text-teal-600 dark:text-teal-300" />
            <h3 class="mt-4 text-base font-semibold text-ui-text">{{ entry.title }}</h3>
            <p v-if="entry.description" class="ui-subheading mt-2 leading-6">{{ entry.description }}</p>
            <span class="mt-4 inline-flex text-sm font-semibold text-teal-700 dark:text-teal-300">{{ entry.action }}</span>
          </RouterLink>
        </div>
      </section>
    </div>

    <!-- 运营模式总览:运行结论 + 一个推荐下一步 → 辅助计数与链接 → 明细 -->
    <div v-else class="space-y-6">
      <div class="flex flex-col gap-3 border-b border-ui-border pb-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 class="ui-heading">{{ pageTitle }}</h1>
          <p class="ui-subheading mt-1">{{ pageDescription }}</p>
        </div>
        <div class="flex flex-wrap gap-2">
          <RouterLink class="btn btn-outline" to="/admin/video/providers">
            <Icon name="server" size="sm" />
            模型通道
          </RouterLink>
          <button class="btn btn-outline" type="button" :disabled="loading" @click="loadDashboard">
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
            刷新
          </button>
        </div>
      </div>

      <section class="ui-panel border-l-4 border-l-teal-500 p-5" aria-label="今日运行结论">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <p class="text-sm font-medium text-ui-text-muted">今日运行结论</p>
          </div>
          <div class="flex flex-wrap gap-2">
            <RouterLink class="btn btn-primary" to="/admin/video/create" data-testid="video-primary-action">
              <Icon name="plus" size="sm" />
              创建视频任务
            </RouterLink>
            <RouterLink class="btn btn-outline" to="/admin/video/providers">
              <Icon name="server" size="sm" />
              管理模型通道
            </RouterLink>
          </div>
        </div>
      </section>

      <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
        <div v-for="item in statItems" :key="item.label" class="ui-panel p-4">
          <div class="text-xs font-medium uppercase text-ui-text-muted">{{ item.label }}</div>
          <div class="mt-2 text-2xl font-semibold tabular-nums text-ui-text">{{ item.value }}</div>
          <div v-if="item.hint" class="mt-1 text-xs text-ui-text-muted">{{ item.hint }}</div>
        </div>
      </div>

      <section class="ui-panel p-5" aria-label="推荐路径">
        <div class="grid gap-4 md:grid-cols-3">
          <div v-for="step in workflowSteps" :key="step.title" class="rounded-xl border border-ui-border p-4">
            <div class="flex items-center justify-between gap-3">
              <h2 class="text-sm font-semibold text-ui-text">{{ step.title }}</h2>
              <span class="rounded-md px-2 py-1 text-xs font-medium" :class="step.done ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300' : 'bg-slate-100 text-slate-700 dark:bg-slate-500/10 dark:text-slate-300'">
                {{ step.done ? '已完成' : step.status }}
              </span>
            </div>
            <p class="ui-subheading mt-2 min-h-10">{{ step.description }}</p>
            <RouterLink class="btn btn-sm btn-outline mt-4" :to="step.to">{{ step.action }}</RouterLink>
          </div>
        </div>
      </section>

      <div class="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
        <section class="ui-panel">
          <div class="border-b border-ui-border px-5 py-4">
            <h2 class="text-base font-semibold text-ui-text">通道状态</h2>
          </div>
          <div class="overflow-x-auto">
            <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
              <thead class="bg-gray-50 text-left text-xs uppercase text-gray-500 dark:bg-dark-700/40 dark:text-gray-400">
                <tr>
                  <th class="px-5 py-3 font-medium">通道</th>
                  <th class="px-5 py-3 font-medium">状态</th>
                  <th class="px-5 py-3 font-medium">凭证状态</th>
                  <th class="px-5 py-3 font-medium">今日任务</th>
                  <th class="px-5 py-3 font-medium">处理中</th>
                  <th class="px-5 py-3 font-medium">失败</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                <tr v-for="provider in dashboard?.provider_status || []" :key="provider.provider">
                  <td class="px-5 py-3">
                    <span class="inline-flex rounded-md px-2 py-1 text-xs font-medium" :class="providerBadgeClass(provider.provider)">
                      {{ providerDisplayName(provider) }}
                    </span>
                    <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ modelDisplayName(provider.provider, provider.default_model) }}</div>
                  </td>
                  <td class="px-5 py-3">
                    <span
                      class="inline-flex rounded-md px-2 py-1 text-xs font-medium"
                      :class="providerRuntimeStatusClass(providerRuntimeStatus(provider))"
                    >
                      {{ providerRuntimeStatus(provider) }}
                    </span>
                  </td>
                  <td class="px-5 py-3">
                    <span class="inline-flex rounded-md px-2 py-1 text-xs font-medium" :class="keyStatusClass(provider.key_status)">
                      {{ providerKeyLabel(provider.api_key_configured, provider.masked_key, provider.key_status, provider.provider) }}
                    </span>
                  </td>
                  <td class="px-5 py-3 text-gray-700 dark:text-gray-200">{{ provider.today_tasks }}</td>
                  <td class="px-5 py-3 text-gray-700 dark:text-gray-200">{{ provider.running_tasks }}</td>
                  <td class="px-5 py-3 text-gray-700 dark:text-gray-200">{{ provider.failed_tasks }}</td>
                </tr>
                <tr v-if="!loading && !(dashboard?.provider_status || []).length">
                  <td colspan="6">
                    <AnimatedEmptyState
                      variant="video-dashboard"
                      title="暂无通道数据"
                      description="请先进入模型通道，确认演示通道状态。"
                    />
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <section class="ui-panel">
          <div class="border-b border-ui-border px-5 py-4">
            <h2 class="text-base font-semibold text-ui-text">用量概览</h2>
          </div>
          <div class="divide-y divide-gray-100 dark:divide-dark-700">
            <div v-for="item in dashboard?.usage_overview || []" :key="`${item.provider}-${item.model}-${item.status}-${item.currency}-${item.pricing_source}-${item.pricing_version}`" class="flex items-center justify-between px-5 py-3 text-sm">
              <div>
                <span class="font-medium text-gray-900 dark:text-white">{{ providerLabel(item.provider) }}</span>
                <span class="ml-2 text-gray-500 dark:text-gray-400">{{ modelDisplayName(item.provider, item.model) }}</span>
              </div>
              <div class="flex items-center gap-2">
                <span class="rounded-md px-2 py-1 text-xs font-medium" :class="statusBadgeClass(item.status)">{{ statusLabel(item.status) }}</span>
                <span class="text-gray-700 dark:text-gray-200">{{ item.count }} 次</span>
                <span class="font-medium tabular-nums text-gray-900 dark:text-white">{{ formatVideoUsageCost(item.cost_estimate, item.currency) }}</span>
              </div>
            </div>
            <AnimatedEmptyState
              v-if="!loading && !(dashboard?.usage_overview || []).length"
              variant="video-dashboard"
              title="暂无用量记录"
              action-label="创建一个演示任务"
              @action="goCreate"
            />
          </div>
        </section>
      </div>

      <div class="grid gap-6 lg:grid-cols-2">
        <RecentTaskPanel title="最近成功" :tasks="dashboard?.recent_successes || []" />
        <RecentTaskPanel title="最近失败" :tasks="dashboard?.recent_failures || []" />
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref, type PropType } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import AnimatedEmptyState from '@/components/common/AnimatedEmptyState.vue'
import { adminAPI } from '@/api/admin'
import type { VideoDashboard, VideoTaskSummary } from '@/api/admin/video'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { isVideoGatewayDemoMode } from '@/utils/productMode'
import {
  errorMessageLabel,
  formatDate,
  keyStatusClass,
  modelDisplayName,
  providerBadgeClass,
  providerDisplayName,
  providerKeyLabel,
  providerLabel,
  providerRuntimeStatus,
  providerRuntimeStatusClass,
  promptDisplayText,
  shortText,
  statusBadgeClass,
  statusLabel,
} from './videoUtils'

const appStore = useAppStore()
const router = useRouter()
const loading = ref(false)
const dashboard = ref<VideoDashboard | null>(null)

function goCreate() {
  router.push('/admin/video/create')
}

const pageTitle = computed(() => isVideoGatewayDemoMode ? '总览' : '视频总览')
const pageDescription = computed(() =>
  isVideoGatewayDemoMode
    ? '老板看统一入口、任务处理、生成能力和演示状态。'
    : '查看视频任务吞吐、成功率、失败和通道状态。',
)

function formatVideoUsageCost(amount: number, currency: string): string {
  if (!Number.isFinite(amount) || amount <= 0) return '—'
  const normalized = String(currency || '').trim().toUpperCase()
  if (amount < 0.0001) return `<0.0001 ${normalized || '币种未知'}`
  if (normalized === 'CNY') return `¥${amount.toFixed(4)}`
  if (normalized === 'USD') return `$${amount.toFixed(4)}`
  return `${amount.toFixed(4)} ${normalized || '币种未知'}`
}

const providers = computed(() => dashboard.value?.provider_status || [])

type StatItem = {
  label: string
  value: string | number
  hint?: string
}

const bossEntryCards: Array<{ title: string; description?: string; action: string; to: string; icon: 'play' | 'document' | 'key' | 'shield' }> = [
  {
    title: '试跑任务',
    action: '试跑一条任务',
    to: '/admin/video/create',
    icon: 'play',
  },
  {
    title: '任务记录',
    description: '查看任务状态、结果和失败原因，确认流程留下可追踪记录。',
    action: '查看记录',
    to: '/admin/video/tasks',
    icon: 'document',
  },
  {
    title: '外部工具接入',
    description: '给自动化工具或脚本使用接入密钥，统一从公司入口提交任务。',
    action: '查看接入密钥',
    to: '/keys',
    icon: 'key',
  },
  {
    title: '系统检查',
    description: '确认本机服务、试跑任务、备份和内网访问是否可用。',
    action: '查看检查项',
    to: '/admin/video/system-check',
    icon: 'shield',
  },
]

const bossStatusItems = computed<StatItem[]>(() => {
  const d = dashboard.value
  return [
    { label: '系统状态', value: '正常', hint: '网页已打开，可继续试跑任务' },
    { label: '今日任务', value: d?.today_tasks ?? 0, hint: '试跑任务会统一进入记录' },
    { label: '处理中', value: d?.running_tasks ?? 0, hint: `队列等待 ${d?.queued_tasks ?? 0}` },
    { label: '成功率', value: `${Math.round(d?.success_rate ?? 0)}%`, hint: '基于已完成记录统计' },
  ]
})

const statItems = computed<StatItem[]>(() => {
  const d = dashboard.value
  return [
    { label: '今日任务', value: d?.today_tasks ?? 0 },
    { label: '成功率', value: `${Math.round(d?.success_rate ?? 0)}%` },
    { label: '失败', value: d?.failed_tasks ?? 0 },
    { label: '处理中', value: d?.running_tasks ?? 0 },
    { label: '排队中', value: d?.queued_tasks ?? 0 },
  ]
})

const workflowSteps = computed(() => {
  const hasDemoProvider = providers.value.some((provider) => provider.provider === 'mock' && provider.enabled)
  const hasTasks = (dashboard.value?.today_tasks || 0) > 0
  const hasResult = Boolean((dashboard.value?.recent_successes || []).length || (dashboard.value?.recent_failures || []).length)

  return [
    {
      title: '1. 配置模型通道',
      description: '确认演示通道可用，并了解 Seedance 2.0 与 Kling 的待授权状态。',
      action: '去配置通道',
      to: '/admin/video/providers',
      done: hasDemoProvider,
      status: '待完成',
    },
    {
      title: '2. 创建视频任务',
      description: '选择模板后提交一个视频任务，系统会进入队列并记录状态。',
      action: '试跑一条任务',
      to: '/admin/video/create',
      done: hasTasks,
      status: '待完成',
    },
    {
      title: '3. 查看结果与用量',
      description: '在任务详情查看处理时间线、结果链接或失败原因，再看用量概览。',
      action: '查看任务列表',
      to: '/admin/video/tasks',
      done: hasResult,
      status: '可选',
    },
  ]
})

async function loadDashboard() {
  loading.value = true
  try {
    dashboard.value = await adminAPI.video.dashboard()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, isVideoGatewayDemoMode ? '加载总览失败' : '加载视频总览失败'))
  } finally {
    loading.value = false
  }
}

const RecentTaskPanel = defineComponent({
  name: 'RecentTaskPanel',
  props: {
    title: { type: String, required: true },
    tasks: { type: Array as PropType<VideoTaskSummary[]>, required: true },
  },
  setup(props) {
    const router = useRouter()
    return () =>
      h('section', { class: 'ui-panel' }, [
        h('div', { class: 'border-b border-ui-border px-5 py-4' }, [
          h('h2', { class: 'text-base font-semibold text-ui-text' }, props.title),
        ]),
        h(
          'div',
          { class: 'divide-y divide-gray-100 dark:divide-dark-700' },
          props.tasks.length
            ? props.tasks.map((task) =>
                h(RouterLink, { to: `/admin/video/tasks/${task.id}`, class: 'block px-5 py-3 hover:bg-gray-50 dark:hover:bg-dark-700/50' }, () => [
                  h('div', { class: 'flex items-start justify-between gap-3' }, [
                    h('div', { class: 'min-w-0' }, [
                      h('div', { class: 'truncate text-sm font-medium text-gray-900 dark:text-white' }, shortText(promptDisplayText(task.prompt), 120)),
                      h('div', { class: 'mt-1 text-xs text-gray-500 dark:text-gray-400' }, `${providerLabel(task.provider)} | ${formatDate(task.updated_at)}`),
                    ]),
                    h('span', { class: ['shrink-0 rounded-md px-2 py-1 text-xs font-medium', statusBadgeClass(task.status)] }, statusLabel(task.status)),
                  ]),
                  task.error_message ? h('div', { class: 'mt-2 text-xs text-red-600 dark:text-red-300' }, shortText(errorMessageLabel(task.error_message), 140)) : null,
                ]),
              )
            : [
                h(AnimatedEmptyState, {
                  variant: 'video-dashboard',
                  title: props.title.includes('失败') ? '没有失败记录。' : '没有成功结果。',
                  actionLabel: isVideoGatewayDemoMode ? '试跑一条任务' : '创建一个演示任务',
                  onAction: () => router.push('/admin/video/create'),
                }),
              ],
        ),
      ])
  },
})

onMounted(loadDashboard)
</script>
