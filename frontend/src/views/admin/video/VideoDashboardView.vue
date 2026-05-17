<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 border-b border-gray-200 pb-4 dark:border-dark-700 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ pageTitle }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ pageDescription }}</p>
        </div>
        <div class="flex flex-wrap gap-2">
          <RouterLink class="btn btn-outline" to="/admin/video/providers">
            <Icon name="server" size="sm" />
            {{ isVideoGatewayDemoMode ? 'API 通道池' : '模型通道' }}
          </RouterLink>
          <RouterLink class="btn btn-primary" to="/admin/video/create">
            <Icon name="plus" size="sm" />
            {{ isVideoGatewayDemoMode ? '发起调用' : '创建任务' }}
          </RouterLink>
          <button class="btn btn-outline" type="button" :disabled="loading" @click="loadDashboard">
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
            刷新
          </button>
        </div>
      </div>

      <section
        v-if="isVideoGatewayDemoMode"
        class="rounded-lg border border-teal-200 bg-teal-50 p-5 dark:border-teal-500/20 dark:bg-teal-500/10"
      >
        <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <p class="text-sm font-medium text-teal-700 dark:text-teal-200">企业 API 网关核心价值</p>
            <h2 class="mt-2 text-xl font-semibold text-gray-950 dark:text-white">
              统一托管多平台 API Key，集中调度视频生成任务，追踪用量、结果与失败原因。
            </h2>
          </div>
          <div class="flex flex-wrap gap-2">
            <RouterLink class="btn btn-primary" to="/admin/video/create">
              <Icon name="play" size="sm" />
              通过网关提交
            </RouterLink>
            <RouterLink class="btn btn-outline" to="/admin/video/providers">
              <Icon name="key" size="sm" />
              管理 API Key
            </RouterLink>
          </div>
        </div>
      </section>

      <section v-else class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <p class="text-sm font-medium text-gray-500 dark:text-gray-400">今日运行结论</p>
            <h2 class="mt-2 text-xl font-semibold text-gray-900 dark:text-white">
              今日已处理 {{ dashboard?.today_tasks ?? 0 }} 个视频任务，成功率 {{ Math.round(dashboard?.success_rate ?? 0) }}%，失败 {{ dashboard?.failed_tasks ?? 0 }} 个。
            </h2>
          </div>
          <div class="flex flex-wrap gap-2">
            <RouterLink class="btn btn-primary" to="/admin/video/create">
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

      <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-6">
        <div v-for="item in statItems" :key="item.label" class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
          <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ item.label }}</div>
          <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ item.value }}</div>
          <div v-if="item.hint" class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ item.hint }}</div>
        </div>
      </div>

      <section v-if="isVideoGatewayDemoMode" class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
        <div class="flex flex-col gap-2 border-b border-gray-200 pb-4 dark:border-dark-700 md:flex-row md:items-end md:justify-between">
          <div>
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">API 调用链路</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">从企业主配置 API Key 到任务分发、状态回收和用量审计的统一网关流程。</p>
          </div>
          <span class="text-xs font-medium text-gray-500 dark:text-gray-400">真实通道未启用节点显示为待配置</span>
        </div>
        <div class="mt-5 grid gap-3 md:grid-cols-4 xl:grid-cols-7">
          <div
            v-for="node in gatewayFlowNodes"
            :key="node.title"
            class="relative rounded-lg border p-4"
            :class="node.done ? 'border-emerald-200 bg-emerald-50 dark:border-emerald-500/20 dark:bg-emerald-500/10' : 'border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-700/40'"
          >
            <div
              v-if="node.arrow"
              class="absolute -left-2 top-1/2 hidden h-px w-4 bg-gray-300 dark:bg-dark-600 md:block"
              aria-hidden="true"
            ></div>
            <div class="flex items-center justify-between gap-2">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ node.title }}</h3>
              <span
                class="shrink-0 rounded-md px-2 py-1 text-xs font-medium"
                :class="node.done ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-200' : 'bg-amber-100 text-amber-700 dark:bg-amber-500/20 dark:text-amber-200'"
              >
                {{ node.done ? '已完成' : node.status }}
              </span>
            </div>
            <p class="mt-2 text-xs leading-5 text-gray-500 dark:text-gray-400">{{ node.description }}</p>
          </div>
        </div>
      </section>

      <section v-else class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
        <div class="grid gap-4 md:grid-cols-3">
          <div v-for="step in workflowSteps" :key="step.title" class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
            <div class="flex items-center justify-between gap-3">
              <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ step.title }}</h2>
              <span class="rounded-md px-2 py-1 text-xs font-medium" :class="step.done ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300' : 'bg-slate-100 text-slate-700 dark:bg-slate-500/10 dark:text-slate-300'">
                {{ step.done ? '已完成' : step.status }}
              </span>
            </div>
            <p class="mt-2 min-h-10 text-sm text-gray-500 dark:text-gray-400">{{ step.description }}</p>
            <RouterLink class="btn btn-sm btn-outline mt-4" :to="step.to">{{ step.action }}</RouterLink>
          </div>
        </div>
      </section>

      <section v-if="isVideoGatewayDemoMode" class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
        <div class="border-b border-gray-200 pb-4 dark:border-dark-700">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">企业主现在能控制什么？</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">把 API 资产、员工调用入口和结果回收集中到一个可审计的管理面板。</p>
        </div>
        <div class="mt-5 grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          <div v-for="item in controlItems" :key="item.title" class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
            <div class="flex items-start gap-3">
              <Icon :name="item.icon" size="sm" class="mt-0.5 text-primary-600 dark:text-primary-300" />
              <div>
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ item.title }}</h3>
                <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">{{ item.description }}</p>
              </div>
            </div>
          </div>
        </div>
      </section>

      <div class="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
        <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <div class="border-b border-gray-200 px-5 py-4 dark:border-dark-700">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ isVideoGatewayDemoMode ? 'API 通道池快照' : '通道状态' }}</h2>
          </div>
          <div class="overflow-x-auto">
            <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
              <thead class="bg-gray-50 text-left text-xs uppercase text-gray-500 dark:bg-dark-700/40 dark:text-gray-400">
                <tr>
                  <th class="px-5 py-3 font-medium">{{ isVideoGatewayDemoMode ? 'API 通道' : '通道' }}</th>
                  <th class="px-5 py-3 font-medium">状态</th>
                  <th class="px-5 py-3 font-medium">Key 状态</th>
                  <th class="px-5 py-3 font-medium">{{ isVideoGatewayDemoMode ? '今日调用' : '今日任务' }}</th>
                  <th class="px-5 py-3 font-medium">{{ isVideoGatewayDemoMode ? '当前并发' : '处理中' }}</th>
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
                      {{ providerKeyLabel(provider.api_key_configured, provider.masked_key, provider.key_status) }}
                    </span>
                  </td>
                  <td class="px-5 py-3 text-gray-700 dark:text-gray-200">{{ provider.today_tasks }}</td>
                  <td class="px-5 py-3 text-gray-700 dark:text-gray-200">{{ provider.running_tasks }}</td>
                  <td class="px-5 py-3 text-gray-700 dark:text-gray-200">{{ provider.failed_tasks }}</td>
                </tr>
                <tr v-if="!loading && !(dashboard?.provider_status || []).length">
                  <td colspan="6" class="px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400">
                    {{ isVideoGatewayDemoMode ? '暂无 API 通道数据，请先进入 API 通道池确认演示通道状态。' : '暂无通道数据，请先进入模型通道确认演示通道状态。' }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <div class="border-b border-gray-200 px-5 py-4 dark:border-dark-700">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ isVideoGatewayDemoMode ? '用量审计概览' : '用量概览' }}</h2>
          </div>
          <div class="divide-y divide-gray-100 dark:divide-dark-700">
            <div v-for="item in dashboard?.usage_overview || []" :key="`${item.provider}-${item.model}-${item.status}`" class="flex items-center justify-between px-5 py-3 text-sm">
              <div>
                <span class="font-medium text-gray-900 dark:text-white">{{ providerLabel(item.provider) }}</span>
                <span class="ml-2 text-gray-500 dark:text-gray-400">{{ modelDisplayName(item.provider, item.model) }}</span>
              </div>
              <div class="flex items-center gap-2">
                <span class="rounded-md px-2 py-1 text-xs font-medium" :class="statusBadgeClass(item.status)">{{ statusLabel(item.status) }}</span>
                <span class="text-gray-700 dark:text-gray-200">{{ item.count }}</span>
              </div>
            </div>
            <div v-if="!loading && !(dashboard?.usage_overview || []).length" class="space-y-3 px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400">
              <div>{{ isVideoGatewayDemoMode ? '暂无用量审计记录。' : '暂无用量记录。' }}</div>
              <RouterLink class="btn btn-sm btn-outline" to="/admin/video/create">{{ isVideoGatewayDemoMode ? '发起一次演示调用' : '创建一个演示任务' }}</RouterLink>
            </div>
          </div>
        </section>
      </div>

      <div class="grid gap-6 lg:grid-cols-2">
        <RecentTaskPanel :title="isVideoGatewayDemoMode ? '最近成功调用' : '最近成功'" :tasks="dashboard?.recent_successes || []" />
        <RecentTaskPanel :title="isVideoGatewayDemoMode ? '最近失败调用' : '最近失败'" :tasks="dashboard?.recent_failures || []" />
      </div>

      <section v-if="isVideoGatewayDemoMode" class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="border-b border-gray-200 px-5 py-4 dark:border-dark-700">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">API 健康诊断</h2>
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs uppercase text-gray-500 dark:bg-dark-700/40 dark:text-gray-400">
              <tr>
                <th class="px-5 py-3 font-medium">API 通道</th>
                <th class="px-5 py-3 font-medium">路由账号</th>
                <th class="px-5 py-3 font-medium">异常类型</th>
                <th class="px-5 py-3 font-medium">最近错误</th>
                <th class="px-5 py-3 font-medium">建议动作</th>
                <th class="px-5 py-3 font-medium">状态</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="item in dashboard?.health_diagnostics || []" :key="`${item.provider}-${item.route_account}`">
                <td class="px-5 py-3 text-gray-700 dark:text-gray-200">{{ providerLabel(item.provider) }}</td>
                <td class="px-5 py-3 text-gray-700 dark:text-gray-200">{{ item.route_account }}</td>
                <td class="px-5 py-3 text-gray-700 dark:text-gray-200">{{ item.exception_type }}</td>
                <td class="px-5 py-3 text-gray-700 dark:text-gray-200">{{ item.recent_error || '-' }}</td>
                <td class="px-5 py-3 text-amber-700 dark:text-amber-300">{{ item.suggested_action || '-' }}</td>
                <td class="px-5 py-3">
                  <span class="inline-flex rounded-md px-2 py-1 text-xs font-medium" :class="item.status === '正常' ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300' : 'bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'">
                    {{ item.status }}
                  </span>
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
import { computed, defineComponent, h, onMounted, ref, type PropType } from 'vue'
import { RouterLink } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
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
const loading = ref(false)
const dashboard = ref<VideoDashboard | null>(null)

const pageTitle = computed(() => isVideoGatewayDemoMode ? 'API 网关驾驶舱' : '视频总览')
const pageDescription = computed(() =>
  isVideoGatewayDemoMode
    ? '集中查看企业视频模型 API 通道、Key 配置、并发队列、调用任务和结果回收状态。'
    : '查看视频任务吞吐、成功率、失败和通道状态。',
)

const providers = computed(() => dashboard.value?.provider_status || [])
const configuredKeyCount = computed(() => providers.value.filter((provider) => provider.api_key_configured).length)
const enabledProviderCount = computed(() => providers.value.filter((provider) => provider.enabled).length)
const realChannelConfigured = computed(() => providers.value.some((provider) => provider.provider !== 'mock' && provider.api_key_configured && provider.enabled))

type StatItem = {
  label: string
  value: string | number
  hint?: string
}

const statItems = computed<StatItem[]>(() => {
  const d = dashboard.value
  if (isVideoGatewayDemoMode) {
    return [
      { label: '已接入 API 通道', value: providers.value.length, hint: '演示通道 + 真实通道预留' },
      { label: '已配置 Key', value: configuredKeyCount.value, hint: '前端仅展示脱敏状态' },
      { label: '启用通道', value: enabledProviderCount.value, hint: '允许员工通过网关调用' },
      { label: '今日 API 调用任务', value: d?.today_tasks ?? 0, hint: '统一记录调用入口' },
      { label: '并发处理中', value: d?.running_tasks ?? 0, hint: `队列等待 ${d?.queued_tasks ?? 0}` },
      { label: '网关成功率', value: `${Math.round(d?.success_rate ?? 0)}%`, hint: '基于已回收任务统计' },
    ]
  }

  return [
    { label: '今日任务', value: d?.today_tasks ?? 0 },
    { label: '成功率', value: `${Math.round(d?.success_rate ?? 0)}%` },
    { label: '失败', value: d?.failed_tasks ?? 0 },
    { label: '处理中', value: d?.running_tasks ?? 0 },
    { label: '排队中', value: d?.queued_tasks ?? 0 },
  ]
})

const gatewayFlowNodes = computed(() => {
  const d = dashboard.value
  const hasTasks = (d?.today_tasks || 0) > 0
  const hasResults = Boolean((d?.recent_successes || []).length || (d?.recent_failures || []).length)
  const hasUsage = Boolean((d?.usage_overview || []).length)
  return [
    { title: '企业 API Key', description: '企业主统一配置，员工不直接接触凭证。', done: configuredKeyCount.value > 0, status: '待配置', arrow: false },
    { title: '加密托管', description: 'Key 由后端保存，前端只看脱敏状态。', done: configuredKeyCount.value > 0, status: '待配置', arrow: true },
    { title: 'API 通道池', description: '演示通道、Seedance 2.0、Kling 集中管理。', done: providers.value.length > 0, status: '待接入', arrow: true },
    { title: '限流 / 并发 / 队列', description: '按启用状态和每分钟限额统一排队。', done: enabledProviderCount.value > 0, status: '待配置', arrow: true },
    { title: '任务分发', description: '调用任务由网关提交到可用 API 通道。', done: hasTasks, status: '待调用', arrow: true },
    { title: '状态回收', description: '轮询并记录成功结果或失败原因。', done: hasResults, status: '待回收', arrow: true },
    { title: '用量审计', description: realChannelConfigured.value ? '按通道沉淀调用量和成功率。' : '真实通道启用后扩展成本统计。', done: hasUsage, status: '待接入统计', arrow: true },
  ]
})

const controlItems = computed<Array<{ title: string; description: string; icon: 'key' | 'server' | 'clock' | 'xCircle' | 'externalLink' | 'swap' }>>(() => [
  { title: '哪些 API Key 已接入', description: `当前已配置 ${configuredKeyCount.value} 个 Key，前端只显示脱敏状态。`, icon: 'key' },
  { title: '哪些通道允许员工调用', description: `当前启用 ${enabledProviderCount.value} 个 API 通道，可随时启停。`, icon: 'server' },
  { title: '每个通道每分钟限额', description: '通道池内维护 rate limit，用于后续统一限流。', icon: 'clock' },
  { title: '当前多少任务正在处理', description: `并发处理中 ${dashboard.value?.running_tasks ?? 0} 个，队列等待 ${dashboard.value?.queued_tasks ?? 0} 个。`, icon: 'server' },
  { title: '哪些任务失败', description: `今日失败 ${dashboard.value?.failed_tasks ?? 0} 个，可进入调用详情查看原因。`, icon: 'xCircle' },
  { title: '哪些调用产生结果', description: `${dashboard.value?.recent_successes?.length ?? 0} 条最近成功调用已回收结果入口。`, icon: 'externalLink' },
  { title: '后续可接哪个真实通道', description: realChannelConfigured.value ? '已有真实通道可进入联调。' : 'Seedance 2.0 / Kling 均为待配置真实通道。', icon: 'swap' },
])

const workflowSteps = computed(() => {
  const hasDemoProvider = providers.value.some((provider) => provider.provider === 'mock' && provider.enabled)
  const hasTasks = (dashboard.value?.today_tasks || 0) > 0
  const hasResult = Boolean((dashboard.value?.recent_successes || []).length || (dashboard.value?.recent_failures || []).length)

  return [
    {
      title: '1. 配置模型通道',
      description: '确认演示通道可用，并了解 Seedance 2.0 与 Kling 的待配置状态。',
      action: '去配置通道',
      to: '/admin/video/providers',
      done: hasDemoProvider,
      status: '待完成',
    },
    {
      title: '2. 创建视频任务',
      description: '选择模板后提交一个视频任务，系统会进入队列并记录状态。',
      action: '创建演示任务',
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
    appStore.showError(extractApiErrorMessage(err, isVideoGatewayDemoMode ? '加载 API 网关驾驶舱失败' : '加载视频总览失败'))
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
    return () =>
      h('section', { class: 'rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800' }, [
        h('div', { class: 'border-b border-gray-200 px-5 py-4 dark:border-dark-700' }, [
          h('h2', { class: 'text-base font-semibold text-gray-900 dark:text-white' }, props.title),
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
                h('div', { class: 'space-y-3 px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400' }, [
                  h('div', props.title.includes('失败') ? '没有失败记录。' : '没有成功结果。'),
                  h(RouterLink, { to: '/admin/video/create', class: 'btn btn-sm btn-outline' }, () =>
                    isVideoGatewayDemoMode ? (props.title.includes('失败') ? '填入失败调用示例' : '发起成功调用示例') : (props.title.includes('失败') ? '运行失败演示' : '运行成功演示'),
                  ),
                ]),
              ],
        ),
      ])
  },
})

onMounted(loadDashboard)
</script>
