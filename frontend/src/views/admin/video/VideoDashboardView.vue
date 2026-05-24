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
            {{ isVideoGatewayDemoMode ? '通道池' : '模型通道' }}
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
        <div class="flex flex-col gap-5 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <p class="text-sm font-medium text-teal-700 dark:text-teal-200">三秒价值区</p>
            <h2 class="mt-2 text-xl font-semibold text-gray-950 dark:text-white">
              公司统一 API 接收 AI 中剧/短剧任务，中央主机自动路由 provider，并沉淀 Prompt / Shot / Skill 数据。
            </h2>
            <p class="mt-2 max-w-3xl text-sm text-teal-800 dark:text-teal-100">
              员工、脚本、n8n 和内部页面都只是 API client；员工不接触 Seedance/Kling 凭证，老板看通道健康、Skill 学习和导出准备。
            </p>
            <div class="mt-4 grid gap-2 sm:grid-cols-2">
              <div class="rounded-lg border border-teal-200 bg-white/70 p-3 dark:border-teal-500/20 dark:bg-dark-800/60">
                <div class="text-xs font-semibold text-teal-700 dark:text-teal-200">内部可验收版本</div>
                <div class="mt-1 text-sm text-gray-700 dark:text-gray-200">API-first 与 Skill 学习准备态可验收，但真实 provider 未验证。</div>
              </div>
              <div class="rounded-lg border border-amber-200 bg-amber-50 p-3 dark:border-amber-500/20 dark:bg-amber-500/10">
                <div class="text-xs font-semibold text-amber-700 dark:text-amber-200">真实生产状态</div>
                <div class="mt-1 text-sm text-gray-700 dark:text-gray-200">暂不可用于真实生产，待 Phase 4C 授权验证。</div>
              </div>
            </div>
            <div class="mt-4 grid gap-2 sm:grid-cols-3">
              <RouterLink v-for="role in roleEntrances" :key="role.title" class="rounded-lg border border-teal-200 bg-white/70 p-3 text-sm hover:border-teal-400 dark:border-teal-500/20 dark:bg-dark-800/60" :to="role.to">
                <div class="font-semibold text-gray-950 dark:text-white">{{ role.title }}</div>
                <div class="mt-1 text-xs leading-5 text-gray-600 dark:text-gray-300">{{ role.description }}</div>
              </RouterLink>
            </div>
          </div>
          <div class="flex flex-wrap gap-2">
            <RouterLink class="btn btn-primary" to="/admin/video/create">
              <Icon name="play" size="sm" />
              通过网关提交
            </RouterLink>
            <RouterLink class="btn btn-outline" to="/admin/video/providers">
              <Icon name="key" size="sm" />
              管理密钥
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
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">API-first 生产链路</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">员工/自动化工具调用公司统一 API，中央主机完成 provider 路由、队列、结果回收和 Skill 学习事件记录。</p>
          </div>
          <span class="text-xs font-medium text-gray-500 dark:text-gray-400">员工无需接触真实账号或 provider 凭证</span>
        </div>
        <div class="mt-5 grid gap-3 md:grid-cols-3 xl:grid-cols-6">
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
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">老板现在能控制什么？</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">把公司统一 API、provider 冗余、剧种模板、Skill 学习和 AI 分析导出集中到一个内部管理面板。</p>
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

      <section v-if="isVideoGatewayDemoMode" class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
        <div class="border-b border-gray-200 pb-4 dark:border-dark-700">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">Skill Learning Center</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">用于生产经验沉淀、公司资产复用和自动化生产准备，不用于监视员工。</p>
        </div>
        <div class="mt-5 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <div v-for="item in skillLearningItems" :key="item.title" class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ item.title }}</h3>
            <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">{{ item.description }}</p>
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
                  <th class="px-5 py-3 font-medium">密钥状态</th>
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
                      {{ providerKeyLabel(provider.api_key_configured, provider.masked_key, provider.key_status, provider.provider) }}
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
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">通道健康诊断</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">这里不是报错堆栈，而是告诉老板哪个供应商账号影响了任务，以及下一步该处理什么。</p>
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs uppercase text-gray-500 dark:bg-dark-700/40 dark:text-gray-400">
              <tr>
                <th class="px-5 py-3 font-medium">API 通道</th>
                <th class="px-5 py-3 font-medium">系统调度账号</th>
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
                <td class="px-5 py-3 text-gray-700 dark:text-gray-200">{{ humanIssueLabel(item.exception_type || item.key_status) }}</td>
                <td class="px-5 py-3 text-gray-700 dark:text-gray-200">{{ humanIssueLabel(item.recent_error) }}</td>
                <td class="px-5 py-3 text-amber-700 dark:text-amber-300">{{ diagnosticSuggestedAction(item) }}</td>
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
  diagnosticSuggestedAction,
  errorMessageLabel,
  formatDate,
  humanIssueLabel,
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

const pageTitle = computed(() => isVideoGatewayDemoMode ? 'AI 短剧 API 网关驾驶舱' : '视频总览')
const pageDescription = computed(() =>
  isVideoGatewayDemoMode
    ? '老板看公司统一 API、中央主机调度、引擎能力、Skill 学习和安全演示状态。'
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

const roleEntrances = [
  { title: '老板看总览', description: '看公司主机、统一 API、provider 健康和 Skill 沉淀。', to: '/admin/video' },
  { title: '员工发起调用', description: '只提交剧种、镜头目标和 prompt，系统自动推荐引擎。', to: '/admin/video/create' },
  { title: '运维处理异常', description: '看影响 provider、fallback 状态和建议动作。', to: '/admin/video/providers' },
]

const statItems = computed<StatItem[]>(() => {
  const d = dashboard.value
  if (isVideoGatewayDemoMode) {
    return [
      { label: '已接入供应商通道', value: providers.value.length, hint: '安全演示通道 + 真实通道预留' },
      { label: '已配置密钥', value: configuredKeyCount.value, hint: '前端仅展示脱敏状态' },
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
  const hasDiagnostics = Boolean((d?.health_diagnostics || []).length)
  return [
    { title: 'API 密钥托管', description: '企业统一配置，员工不直接接触凭证。', done: configuredKeyCount.value > 0, status: '待配置', arrow: false },
    { title: '通道池', description: '安全演示通道、Seedance、Kling 统一看状态。', done: providers.value.length > 0, status: '待接入', arrow: true },
    { title: '自动调度', description: '系统挑选当前可用且处理中较少的账号。', done: enabledProviderCount.value > 0, status: '待配置', arrow: true },
    { title: '任务执行', description: '员工提交后进入网关队列并分发。', done: hasTasks, status: '待调用', arrow: true },
    { title: '结果回收', description: '成功结果和失败原因都会回到任务详情。', done: hasResults, status: '待回收', arrow: true },
    { title: '异常诊断', description: '告诉老板哪个账号影响任务，以及下一步动作。', done: hasDiagnostics, status: '待诊断', arrow: true },
  ]
})

const controlItems = computed<Array<{ title: string; description: string; icon: 'key' | 'server' | 'clock' | 'xCircle' | 'externalLink' | 'swap' }>>(() => [
  { title: '统一 API 入口', description: '内部页面、脚本、n8n 和自动化工具都通过公司统一 API。', icon: 'server' },
  { title: 'provider 凭证隔离', description: `当前已配置 ${configuredKeyCount.value} 个凭证状态，前端只显示脱敏结果。`, icon: 'key' },
  { title: '并发和预算', description: `并发处理中 ${dashboard.value?.running_tasks ?? 0} 个，队列等待 ${dashboard.value?.queued_tasks ?? 0} 个。`, icon: 'clock' },
  { title: '失败模式沉淀', description: `今日失败 ${dashboard.value?.failed_tasks ?? 0} 个，可进入调用详情查看原因并进入 Skill 学习。`, icon: 'xCircle' },
  { title: 'AI Analysis Export', description: '只生成脱敏 JSON 与 Gemini/GPT/Kimi Prompt，不自动调用外部 AI。', icon: 'externalLink' },
  { title: '真实 provider 前置条件', description: realChannelConfigured.value ? '已有真实通道配置痕迹，仍需授权验证。' : 'Seedance 2.0 / Kling 均为待授权验证。', icon: 'swap' },
])

const skillLearningItems = [
  { title: '员工 Skill Card 草案', description: '按 employee_alias 汇总剧种、场景、镜头选择、prompt 结构和结果反馈。' },
  { title: '团队 Skill Summary', description: '沉淀团队常用剧种模板、模型选择建议和失败模式。' },
  { title: 'Prompt Structure Template', description: '把角色身份、场景上下文、戏剧目标、镜头语言和负向词结构化。' },
  { title: 'AI Analysis Export', description: '生成脱敏 JSON 与 Gemini/GPT/Kimi 分析 Prompt，不发起外部 AI 调用。' },
]

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
