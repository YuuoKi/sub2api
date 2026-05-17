<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 border-b border-gray-200 pb-4 dark:border-dark-700 md:flex-row md:items-center md:justify-between">
        <div>
          <div class="flex items-center gap-3">
            <RouterLink class="text-sm font-medium text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white" to="/admin/video/tasks">
              {{ isVideoGatewayDemoMode ? '调用任务' : '任务列表' }}
            </RouterLink>
            <span class="text-gray-300 dark:text-dark-500">/</span>
            <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ isVideoGatewayDemoMode ? 'API 调用详情' : `#${task?.id || route.params.id}` }}</h1>
          </div>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ isVideoGatewayDemoMode ? '该任务由 API 网关提交至模型通道，系统负责状态轮询、结果回收和失败记录。' : '查看单个任务的参数、处理时间线、结果或失败原因。' }}
          </p>
        </div>
        <div class="flex flex-wrap gap-2">
          <button v-if="task && !isTerminalStatus(task.status)" class="btn btn-outline" type="button" :disabled="cancelling" @click="cancelTask">
            <Icon name="ban" size="sm" />
            {{ isVideoGatewayDemoMode ? '取消调用' : '取消任务' }}
          </button>
          <button v-if="task" class="btn btn-outline" type="button" @click="copyToCreate">
            {{ isVideoGatewayDemoMode ? '复制参数重新发起' : '复制参数' }}
          </button>
          <button class="btn btn-outline" type="button" :disabled="loading" @click="loadTask">
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
            刷新
          </button>
        </div>
      </div>

      <section v-if="task" class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ isVideoGatewayDemoMode ? '状态概览' : '任务概览' }}</h2>
        <div class="mt-4 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ isVideoGatewayDemoMode ? '调用状态' : '状态' }}</div>
            <div class="mt-2">
              <span class="inline-flex rounded-md px-2 py-1 text-xs font-medium" :class="statusBadgeClass(task.status)">
                {{ statusLabel(task.status) }}
              </span>
            </div>
          </div>
          <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ isVideoGatewayDemoMode ? 'API 通道' : '通道' }}</div>
            <div class="mt-2 text-lg font-semibold text-gray-900 dark:text-white">{{ providerLabel(task.provider) }}</div>
          </div>
          <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">模型</div>
            <div class="mt-2 truncate text-lg font-semibold text-gray-900 dark:text-white">{{ modelDisplayName(task.provider, task.model) }}</div>
          </div>
          <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
            <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">更新时间</div>
            <div class="mt-2 text-sm font-medium text-gray-900 dark:text-white">{{ formatDate(task.updated_at) }}</div>
          </div>
        </div>
      </section>

      <div v-if="task" class="grid gap-6 xl:grid-cols-[0.8fr_1.2fr]">
        <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <div class="border-b border-gray-200 px-5 py-4 dark:border-dark-700">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ isVideoGatewayDemoMode ? 'API 通道信息' : '任务参数' }}</h2>
          </div>
          <dl class="divide-y divide-gray-100 text-sm dark:divide-dark-700">
            <div v-for="row in channelRows" :key="row.label" class="grid gap-2 px-5 py-3 sm:grid-cols-[160px_1fr]">
              <dt class="text-gray-500 dark:text-gray-400">{{ row.label }}</dt>
              <dd class="min-w-0 break-words text-gray-900 dark:text-gray-100">{{ row.value || '-' }}</dd>
            </div>
          </dl>
        </section>

        <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <div class="border-b border-gray-200 px-5 py-4 dark:border-dark-700">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ isVideoGatewayDemoMode ? '网关处理时间线' : '处理时间线' }}</h2>
          </div>
          <div class="space-y-4 p-5">
            <div v-for="event in task.events || []" :key="event.id" class="relative pl-6">
              <span class="absolute left-0 top-1.5 h-2.5 w-2.5 rounded-full" :class="eventDotClass(event.event_type)"></span>
              <div class="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <div class="font-medium text-gray-900 dark:text-white">{{ eventMessageLabel(event.message, event.event_type) }}</div>
                  <div class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ eventTypeLabel(event.event_type) }}</div>
                </div>
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ formatDate(event.created_at) }}</div>
              </div>
              <details v-if="!isVideoGatewayDemoMode && event.payload_json && Object.keys(event.payload_json).length" class="mt-2">
                <summary class="cursor-pointer text-xs font-medium text-gray-500 dark:text-gray-400">技术 payload</summary>
                <pre class="mt-2 max-h-64 overflow-auto rounded-md bg-gray-50 p-3 text-xs text-gray-700 dark:bg-dark-900 dark:text-gray-200">{{ JSON.stringify(event.payload_json, null, 2) }}</pre>
              </details>
            </div>
            <div v-if="!(task.events || []).length" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">{{ isVideoGatewayDemoMode ? '暂无网关时间线事件' : '暂无时间线事件' }}</div>
          </div>
        </section>
      </div>

      <section v-if="task" class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="border-b border-gray-200 px-5 py-4 dark:border-dark-700">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ isVideoGatewayDemoMode ? '调用参数' : '任务参数' }}</h2>
        </div>
        <dl class="divide-y divide-gray-100 text-sm dark:divide-dark-700">
          <div v-for="row in parameterRows" :key="row.label" class="grid gap-2 px-5 py-3 sm:grid-cols-[160px_1fr]">
            <dt class="text-gray-500 dark:text-gray-400">{{ row.label }}</dt>
            <dd class="min-w-0 break-words text-gray-900 dark:text-gray-100">{{ row.value || '-' }}</dd>
          </div>
        </dl>
      </section>

      <section v-if="task" class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ isVideoGatewayDemoMode ? '结果 / 失败原因' : '结果区' }}</h2>
          <div class="flex flex-wrap gap-2">
            <a v-if="task.result_url" class="btn btn-sm btn-outline" :href="task.result_url" target="_blank" rel="noreferrer">
              {{ isVideoGatewayDemoMode ? '打开结果链接' : '打开结果' }}
            </a>
            <button v-if="task.result_url" class="btn btn-sm btn-outline" type="button" @click="copyResultUrl">
              复制结果链接
            </button>
            <button class="btn btn-sm btn-outline" type="button" @click="copyToCreate">
              {{ isVideoGatewayDemoMode ? '复制参数复用' : '复制参数' }}
            </button>
          </div>
        </div>
        <div class="mt-4 space-y-3 text-sm">
          <div v-if="task.result_url" class="rounded-lg border border-emerald-200 bg-emerald-50 p-4 text-emerald-800 dark:border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-200">
            <div class="font-medium">{{ isVideoGatewayDemoMode ? '调用成功，结果链接已由网关回收。' : '视频生成成功，结果链接已回收。' }}</div>
            <p class="mt-1 text-xs text-emerald-700 dark:text-emerald-300">
              {{ isVideoGatewayDemoMode ? '演示通道返回闭环结果；真实通道配置 API Key 后会回收上游真实结果。' : '演示通道返回本地闭环结果；正式通道配置 API Key 后会回收上游真实结果。' }}
            </p>
            <ul v-if="isVideoGatewayDemoMode" class="mt-3 list-disc space-y-1 pl-5 text-xs">
              <li>打开结果链接</li>
              <li>复制结果链接</li>
              <li>复制参数复用</li>
            </ul>
          </div>
          <div v-else-if="task.status !== 'failed'" class="rounded-lg border border-dashed border-gray-200 p-4 text-gray-500 dark:border-dark-700 dark:text-gray-400">
            {{ isVideoGatewayDemoMode ? '暂无结果链接。状态回收完成后这里会显示结果入口。' : '暂无结果链接。任务完成后这里会显示结果入口。' }}
          </div>
          <div v-if="task.error_message" class="rounded-lg border border-red-200 bg-red-50 p-4 text-red-700 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-300">
            <div class="font-medium">{{ isVideoGatewayDemoMode ? '调用失败原因' : '失败原因' }}：{{ errorMessageLabel(task.error_message) }}</div>
            <ul class="mt-3 list-disc space-y-1 pl-5 text-xs">
              <template v-if="isVideoGatewayDemoMode">
                <li>检查提示词是否触发上游审核</li>
                <li>切换 API 通道</li>
                <li>复制参数重新发起调用</li>
              </template>
              <template v-else>
                <li>检查提示词是否过短、冲突或触发演示失败场景。</li>
                <li>更换模型通道，或确认通道已启用。</li>
                <li>复制参数重新创建，便于演示失败恢复流程。</li>
              </template>
            </ul>
          </div>
        </div>
      </section>

      <details v-if="task" class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
        <summary class="cursor-pointer text-base font-semibold text-gray-900 dark:text-white">技术详情</summary>
        <div v-if="isVideoGatewayDemoMode" class="mt-4 rounded-md bg-gray-50 p-4 text-sm text-gray-600 dark:bg-dark-900 dark:text-gray-300">
          技术详情默认折叠。客户演示模式仅展示调用状态、通道、参数、时间线、结果和失败原因；底层字段保留给内部排障环境。
        </div>
        <pre v-else class="mt-4 max-h-96 overflow-auto rounded-md bg-gray-50 p-4 text-xs text-gray-700 dark:bg-dark-900 dark:text-gray-200">{{ JSON.stringify(task, null, 2) }}</pre>
      </details>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { videoTaskAPI, type VideoTask } from '@/api/admin/video'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { isVideoGatewayDemoMode } from '@/utils/productMode'
import {
  errorMessageLabel,
  eventMessageLabel,
  eventTypeLabel,
  formatDate,
  isTerminalStatus,
  modelDisplayName,
  createdByLabel,
  providerLabel,
  routeAccountLabel,
  routingStrategyLabel,
  saveTaskDraft,
  statusBadgeClass,
  statusLabel,
  taskTypeLabel,
} from './videoUtils'

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const task = ref<VideoTask | null>(null)
const loading = ref(false)
const cancelling = ref(false)
let refreshTimer: ReturnType<typeof setInterval> | null = null

const taskId = computed(() => Number(route.params.id))
const channelRows = computed(() => {
  if (!task.value) return []
  if (isVideoGatewayDemoMode) {
    return [
      { label: '调用编号', value: `#${task.value.id}` },
      { label: '发起人', value: createdByLabel(task.value) },
      { label: 'API 通道', value: providerLabel(task.value.provider) },
      { label: '实际路由账号', value: routeAccountLabel(task.value) },
      { label: '路由策略', value: routingStrategyLabel(task.value.routing_strategy) },
      { label: '路由原因', value: task.value.routing_reason || '-' },
      { label: '模型', value: modelDisplayName(task.value.provider, task.value.model) },
      { label: '网关状态', value: statusLabel(task.value.status) },
      { label: '创建时间', value: formatDate(task.value.created_at) },
      { label: '完成时间', value: task.value.completed_at ? formatDate(task.value.completed_at) : '-' },
    ]
  }
  return [
    { label: '任务编号', value: `#${task.value.id}` },
    { label: '发起人', value: createdByLabel(task.value) },
    { label: '通道', value: providerLabel(task.value.provider) },
    { label: '实际路由账号', value: routeAccountLabel(task.value) },
    { label: '模型', value: modelDisplayName(task.value.provider, task.value.model) },
    { label: '状态', value: statusLabel(task.value.status) },
  ]
})

const parameterRows = computed(() => {
  if (!task.value) return []
  return [
    { label: isVideoGatewayDemoMode ? '调用类型' : '任务类型', value: taskTypeLabel(task.value.task_type) },
    { label: '完整提示词', value: task.value.prompt },
    { label: '负向提示词', value: task.value.negative_prompt },
    { label: '参考图', value: task.value.reference_image_url },
    { label: '参考视频', value: task.value.reference_video_url },
    { label: '画幅比例', value: task.value.aspect_ratio },
    { label: '时长', value: `${task.value.duration}s` },
    { label: '分辨率', value: task.value.resolution },
  ]
})

function eventDotClass(eventType: string): string {
  if (eventType === 'succeeded') return 'bg-emerald-500'
  if (eventType === 'failed') return 'bg-red-500'
  if (eventType === 'routed') return 'bg-sky-500'
  if (eventType === 'running') return 'bg-amber-500'
  return 'bg-gray-400'
}

async function loadTask() {
  loading.value = true
  try {
    task.value = await videoTaskAPI.get(taskId.value)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, isVideoGatewayDemoMode ? '加载 API 调用详情失败' : '加载任务详情失败'))
  } finally {
    loading.value = false
  }
}

async function cancelTask() {
  if (!task.value) return
  cancelling.value = true
  try {
    task.value = await videoTaskAPI.cancel(task.value.id)
    appStore.showSuccess(isVideoGatewayDemoMode ? '调用已取消' : '视频任务已取消')
    await loadTask()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, isVideoGatewayDemoMode ? '取消调用失败' : '取消视频任务失败'))
  } finally {
    cancelling.value = false
  }
}

async function copyText(value: string, successMessage: string) {
  try {
    await navigator.clipboard.writeText(value)
    appStore.showSuccess(successMessage)
  } catch {
    appStore.showError('复制失败，请手动选择内容复制。')
  }
}

function copyResultUrl() {
  if (!task.value?.result_url) return
  copyText(task.value.result_url, '结果链接已复制。')
}

function copyToCreate() {
  if (!task.value) return
  saveTaskDraft(task.value)
  appStore.showInfo(isVideoGatewayDemoMode ? '已复制调用参数，可在发起调用页调整后重新提交。' : '已复制任务参数，可在创建页调整后重新提交。')
  router.push('/admin/video/create')
}

onMounted(async () => {
  await loadTask()
  refreshTimer = setInterval(() => {
    if (task.value && !isTerminalStatus(task.value.status)) {
      loadTask()
    }
  }, 2000)
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})
</script>
