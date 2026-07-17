<template>
  <AppLayout>
    <div class="video-tasks-view ui-page min-w-0 space-y-5">
      <header class="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 class="ui-heading">任务记录</h2>
          <p class="ui-subheading mt-1">仅展示本人创建的内部模拟任务。</p>
        </div>
        <div class="flex gap-2">
          <RouterLink class="btn btn-primary" to="/video/create">创建任务</RouterLink>
          <button type="button" class="btn btn-secondary" :disabled="loading" @click="load">刷新</button>
        </div>
      </header>

      <p v-if="error" class="text-sm text-red-600" role="alert">{{ error }}</p>

      <div class="ui-panel overflow-x-auto" role="region" aria-label="我的模拟任务">
        <table class="w-full min-w-[720px] text-left text-sm">
          <thead>
            <tr>
              <th class="p-3">任务</th>
              <th class="p-3">状态</th>
              <th class="p-3">费用</th>
              <th class="p-3">结果</th>
              <th class="p-3">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="task in tasks" :key="task.id" class="align-top border-t border-gray-100 dark:border-dark-700">
              <td class="p-3">
                <RouterLink class="font-medium text-primary-600" :to="`/video/tasks/${task.id}`">
                  #{{ task.id }}
                </RouterLink>
                <div class="mt-1 max-w-xs truncate text-xs text-gray-500" :title="task.prompt">{{ task.prompt }}</div>
              </td>
              <td class="p-3">
                <TaskProgressRing :phase="phaseLabel(task.status)" :size="26" side-label />
              </td>
              <td class="whitespace-nowrap p-3">{{ task.cost }} {{ task.currency || 'USD' }}</td>
              <td class="p-3">
                <template v-if="task.status === 'succeeded' && previewUrls[task.id]">
                  <img
                    v-if="mediaKind === 'image'"
                    :src="previewUrls[task.id]"
                    alt="模拟结果预览"
                    class="h-16 w-auto max-w-[120px] rounded border border-gray-200 object-contain dark:border-dark-600"
                  />
                  <span v-else class="text-xs text-gray-500">当前结果不是图像，无法在线预览。</span>
                </template>
                <span v-else-if="task.error" class="text-red-600">{{ task.error }}</span>
                <span v-else class="text-gray-500">—</span>
              </td>
              <td class="p-3">
                <button
                  v-if="isCancellable(task.status)"
                  type="button"
                  class="btn btn-sm btn-outline"
                  :disabled="cancellingId === task.id"
                  @click="onCancel(task.id)"
                >
                  取消
                </button>
                <RouterLink v-else class="text-sm text-primary-600" :to="`/video/tasks/${task.id}`">详情</RouterLink>
              </td>
            </tr>
          </tbody>
        </table>
        <div v-if="!loading && tasks.length === 0" class="p-4">
          <AnimatedEmptyState
            variant="video-tasks"
            title="暂无模拟任务"
            description="创建任务后，这里会显示状态与模拟结果预览。"
          />
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, onUnmounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import AnimatedEmptyState from '@/components/common/AnimatedEmptyState.vue'
import TaskProgressRing from '@/components/common/TaskProgressRing.vue'
import {
  cancelSimulationTask,
  downloadSimulationResult,
  getSimulationContract,
  listSimulationTasks,
  type VideoSimulationTask,
  type VideoSimulationTaskStatus,
} from '@/api/user/video_simulation'
import { extractApiErrorMessage } from '@/utils/apiError'

const tasks = ref<VideoSimulationTask[]>([])
const loading = ref(false)
const error = ref('')
const cancellingId = ref<number | null>(null)
const previewUrls = ref<Record<number, string>>({})
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

function revokePreviews() {
  for (const url of Object.values(previewUrls.value)) {
    URL.revokeObjectURL(url)
  }
  previewUrls.value = {}
}

async function loadPreviews(items: VideoSimulationTask[], signal: AbortSignal) {
  const created: string[] = []
  const next: Record<number, string> = { ...previewUrls.value }
  try {
    for (const task of items) {
      if (signal.aborted) return
      if (task.status !== 'succeeded' || next[task.id]) continue
      const blob = await downloadSimulationResult(task.id, { signal })
      if (signal.aborted) {
        for (const url of created) URL.revokeObjectURL(url)
        return
      }
      const url = URL.createObjectURL(blob)
      created.push(url)
      next[task.id] = url
    }
    if (!signal.aborted) {
      previewUrls.value = next
    } else {
      for (const url of created) URL.revokeObjectURL(url)
    }
  } catch (err) {
    for (const url of created) URL.revokeObjectURL(url)
    if (signal.aborted || isAbortError(err)) return
    // Preview is best-effort; detail page can still download.
  }
}

async function load() {
  loadAbortController?.abort()
  const controller = new AbortController()
  loadAbortController = controller
  loading.value = true
  error.value = ''
  try {
    const result = await listSimulationTasks({ signal: controller.signal })
    if (controller.signal.aborted) return
    tasks.value = result.items ?? []
    await loadPreviews(tasks.value, controller.signal)
  } catch (err) {
    if (controller.signal.aborted || isAbortError(err)) return
    error.value = extractApiErrorMessage(err, '加载任务失败')
  } finally {
    if (loadAbortController === controller) loading.value = false
  }
}

async function onCancel(id: number) {
  cancellingId.value = id
  try {
    await cancelSimulationTask(id)
    await load()
  } catch (err) {
    error.value = extractApiErrorMessage(err, '取消任务失败')
  } finally {
    cancellingId.value = null
  }
}

function startPolling() {
  stopPolling()
  pollTimer = setInterval(() => {
    if (tasks.value.some((task) => isCancellable(task.status))) {
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

onMounted(() => {
  void loadContract()
  void load().then(startPolling)
})

onBeforeUnmount(() => {
  stopPolling()
  loadAbortController?.abort()
  revokePreviews()
})

onUnmounted(() => {
  stopPolling()
  loadAbortController?.abort()
})
</script>
