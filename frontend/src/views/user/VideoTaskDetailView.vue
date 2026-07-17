<template>
  <AppLayout>
    <div class="video-task-detail-view ui-page min-w-0 space-y-5">
      <header>
        <RouterLink to="/video/tasks" class="inline-flex text-sm text-primary-600">返回任务记录</RouterLink>
        <h2 class="ui-heading mt-2">任务详情 #{{ taskId }}</h2>
        <p class="ui-subheading mt-1">内部模拟任务详情与结果预览。</p>
      </header>

      <p v-if="error" class="text-sm text-red-600" role="alert">{{ error }}</p>
      <p v-else-if="loading" class="text-sm text-gray-500">加载中…</p>

      <div v-if="task" class="ui-panel space-y-3 p-4 text-sm">
        <p><span class="font-medium">状态：</span>{{ phaseLabel(task.status) }}（{{ task.status }}）</p>
        <p><span class="font-medium">模型：</span>{{ task.provider }} / {{ task.model }}</p>
        <p><span class="font-medium">规格：</span>{{ task.resolution }} · {{ task.duration }}s</p>
        <p>
          <span class="font-medium">费用：</span>{{ task.cost }} {{ task.currency }}
          <span class="text-gray-500">（{{ task.pricing_source }} / {{ task.pricing_version }}）</span>
        </p>
        <p><span class="font-medium">提示词：</span></p>
        <pre class="whitespace-pre-wrap break-words rounded bg-gray-50 p-3 dark:bg-dark-800">{{ task.prompt }}</pre>
        <p v-if="task.error" class="text-red-600"><span class="font-medium">错误：</span>{{ task.error }}</p>

        <div class="flex flex-wrap gap-2 pt-2">
          <button
            v-if="isCancellable(task.status)"
            type="button"
            class="btn btn-outline"
            :disabled="cancelling"
            @click="onCancel"
          >
            取消任务
          </button>
          <button
            v-if="task.status === 'succeeded'"
            type="button"
            class="btn btn-secondary"
            :disabled="loadingResult"
            @click="onDownload"
          >
            下载结果
          </button>
        </div>
      </div>

      <div v-if="previewUrl || (task?.status === 'succeeded' && mediaKind !== 'image')" class="ui-panel p-4">
        <h3 class="mb-2 text-sm font-medium">结果预览（模拟图像）</h3>
        <img
          v-if="previewUrl && mediaKind === 'image'"
          :src="previewUrl"
          alt="模拟视频结果预览"
          class="max-h-96 w-auto max-w-full rounded border border-gray-200 object-contain dark:border-dark-600"
        />
        <p v-else-if="task?.status === 'succeeded' && mediaKind !== 'image'" class="text-sm text-gray-500">
          当前结果不是图像，无法在线预览。
        </p>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, onUnmounted, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import {
  cancelSimulationTask,
  downloadSimulationResult,
  getSimulationContract,
  getSimulationTask,
  type VideoSimulationTask,
  type VideoSimulationTaskStatus,
} from '@/api/user/video_simulation'
import { extractApiErrorMessage } from '@/utils/apiError'

const route = useRoute()
const taskId = computed(() => Number(route.params.id))
const task = ref<VideoSimulationTask | null>(null)
const loading = ref(false)
const error = ref('')
const cancelling = ref(false)
const loadingResult = ref(false)
const previewUrl = ref('')
/** From simulation contract; gates <img> so non-image media_kind shows a fallback. */
const mediaKind = ref<'image' | 'video'>('image')
let pollTimer: ReturnType<typeof setInterval> | null = null
let loadAbortController: AbortController | null = null

function isAbortError(err: unknown): boolean {
  return (
    (err as { name?: string })?.name === 'AbortError' ||
    (err as { code?: string })?.code === 'ERR_CANCELED'
  )
}

function phaseLabelKnown(status: VideoSimulationTaskStatus): string {
  switch (status) {
    case 'queued':
      return '排队中'
    case 'submitted':
      return '已提交'
    case 'running':
      return '生成中'
    case 'succeeded':
      return '已完成'
    case 'failed':
      return '失败'
    case 'cancelled':
      return '已取消'
    default: {
      const _exhaustive: never = status
      return _exhaustive
    }
  }
}

function phaseLabel(status: string): string {
  switch (status) {
    case 'queued':
    case 'submitted':
    case 'running':
    case 'succeeded':
    case 'failed':
    case 'cancelled':
      return phaseLabelKnown(status)
    default:
      return '未知'
  }
}

function isCancellable(status: string): boolean {
  return status === 'queued' || status === 'submitted' || status === 'running'
}

function clearPreview() {
  if (previewUrl.value) {
    URL.revokeObjectURL(previewUrl.value)
    previewUrl.value = ''
  }
}

async function loadResultPreview(signal: AbortSignal) {
  if (!task.value || task.value.status !== 'succeeded') return
  if (mediaKind.value !== 'image') return
  loadingResult.value = true
  try {
    const blob = await downloadSimulationResult(task.value.id, { signal })
    if (signal.aborted) return
    clearPreview()
    previewUrl.value = URL.createObjectURL(blob)
  } catch (err) {
    if (signal.aborted || isAbortError(err)) return
    error.value = extractApiErrorMessage(err, '加载结果预览失败')
  } finally {
    if (!signal.aborted) loadingResult.value = false
  }
}

async function load() {
  if (!Number.isFinite(taskId.value) || taskId.value <= 0) {
    error.value = '无效的任务 ID'
    return
  }
  loadAbortController?.abort()
  const controller = new AbortController()
  loadAbortController = controller
  loading.value = true
  error.value = ''
  try {
    task.value = await getSimulationTask(taskId.value, { signal: controller.signal })
    if (controller.signal.aborted) return
    if (task.value.status === 'succeeded' && !previewUrl.value) {
      await loadResultPreview(controller.signal)
    }
  } catch (err) {
    if (controller.signal.aborted || isAbortError(err)) return
    task.value = null
    error.value = extractApiErrorMessage(err, '加载任务详情失败')
  } finally {
    if (loadAbortController === controller) loading.value = false
  }
}

async function onCancel() {
  if (!task.value) return
  cancelling.value = true
  try {
    task.value = await cancelSimulationTask(task.value.id)
  } catch (err) {
    error.value = extractApiErrorMessage(err, '取消任务失败')
  } finally {
    cancelling.value = false
  }
}

async function onDownload() {
  if (!task.value) return
  loadingResult.value = true
  try {
    const blob = await downloadSimulationResult(task.value.id)
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = `simulation-task-${task.value.id}.svg`
    anchor.click()
    URL.revokeObjectURL(url)
  } catch (err) {
    error.value = extractApiErrorMessage(err, '下载结果失败')
  } finally {
    loadingResult.value = false
  }
}

function startPolling() {
  stopPolling()
  pollTimer = setInterval(() => {
    if (task.value && isCancellable(task.value.status)) {
      void load()
    }
  }, 3000)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

async function loadContract() {
  try {
    const contract = await getSimulationContract()
    mediaKind.value = contract.media_kind
  } catch {
    mediaKind.value = 'image'
  }
}

watch(taskId, () => {
  clearPreview()
  void load()
})

onMounted(() => {
  void loadContract()
  void load().then(startPolling)
})

onBeforeUnmount(() => {
  stopPolling()
  loadAbortController?.abort()
  clearPreview()
})

onUnmounted(() => {
  stopPolling()
  loadAbortController?.abort()
})
</script>
