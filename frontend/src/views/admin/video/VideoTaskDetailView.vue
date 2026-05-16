<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 border-b border-gray-200 pb-4 dark:border-dark-700 md:flex-row md:items-center md:justify-between">
        <div>
          <div class="flex items-center gap-3">
            <RouterLink class="text-sm font-medium text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white" to="/admin/video/tasks">
              Tasks
            </RouterLink>
            <span class="text-gray-300 dark:text-dark-500">/</span>
            <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">#{{ task?.id || route.params.id }}</h1>
          </div>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ task ? shortText(task.prompt, 140) : 'Loading' }}</p>
        </div>
        <div class="flex flex-wrap gap-2">
          <button v-if="task && !isTerminalStatus(task.status)" class="btn btn-outline" type="button" :disabled="cancelling" @click="cancelTask">
            <Icon name="ban" size="sm" />
            Cancel
          </button>
          <button class="btn btn-outline" type="button" :disabled="loading" @click="loadTask">
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
            Refresh
          </button>
        </div>
      </div>

      <section v-if="task" class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <div class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
          <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Status</div>
          <div class="mt-2">
            <span class="inline-flex rounded-md px-2 py-1 text-xs font-medium" :class="statusBadgeClass(task.status)">
              {{ statusLabel(task.status) }}
            </span>
          </div>
        </div>
        <div class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
          <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Provider</div>
          <div class="mt-2 text-lg font-semibold text-gray-900 dark:text-white">{{ providerLabel(task.provider) }}</div>
        </div>
        <div class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
          <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Model</div>
          <div class="mt-2 truncate text-lg font-semibold text-gray-900 dark:text-white">{{ task.model }}</div>
        </div>
        <div class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
          <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Updated</div>
          <div class="mt-2 text-sm font-medium text-gray-900 dark:text-white">{{ formatDate(task.updated_at) }}</div>
        </div>
      </section>

      <div v-if="task" class="grid gap-6 xl:grid-cols-[0.9fr_1.1fr]">
        <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <div class="border-b border-gray-200 px-5 py-4 dark:border-dark-700">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">Parameters</h2>
          </div>
          <dl class="divide-y divide-gray-100 text-sm dark:divide-dark-700">
            <div v-for="row in parameterRows" :key="row.label" class="grid gap-2 px-5 py-3 sm:grid-cols-[160px_1fr]">
              <dt class="text-gray-500 dark:text-gray-400">{{ row.label }}</dt>
              <dd class="min-w-0 break-words text-gray-900 dark:text-gray-100">{{ row.value || '-' }}</dd>
            </div>
          </dl>
        </section>

        <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <div class="border-b border-gray-200 px-5 py-4 dark:border-dark-700">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">Timeline</h2>
          </div>
          <div class="space-y-4 p-5">
            <div v-for="event in task.events || []" :key="event.id" class="relative pl-6">
              <span class="absolute left-0 top-1.5 h-2.5 w-2.5 rounded-full" :class="eventDotClass(event.event_type)"></span>
              <div class="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
                <div class="font-medium text-gray-900 dark:text-white">{{ event.message || event.event_type }}</div>
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ formatDate(event.created_at) }}</div>
              </div>
              <details v-if="event.payload_json && Object.keys(event.payload_json).length" class="mt-2">
                <summary class="cursor-pointer text-xs font-medium text-gray-500 dark:text-gray-400">Payload</summary>
                <pre class="mt-2 max-h-64 overflow-auto rounded-md bg-gray-50 p-3 text-xs text-gray-700 dark:bg-dark-900 dark:text-gray-200">{{ JSON.stringify(event.payload_json, null, 2) }}</pre>
              </details>
            </div>
            <div v-if="!(task.events || []).length" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">No events</div>
          </div>
        </section>
      </div>

      <section v-if="task" class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">Result</h2>
        <div class="mt-4 space-y-3 text-sm">
          <div v-if="task.result_url" class="flex flex-wrap items-center gap-3">
            <a class="inline-flex items-center gap-2 font-medium text-primary-600 hover:text-primary-700 dark:text-primary-300" :href="task.result_url" target="_blank" rel="noreferrer">
              <Icon name="externalLink" size="sm" />
              {{ task.result_url }}
            </a>
          </div>
          <div v-else class="text-gray-500 dark:text-gray-400">-</div>
          <div v-if="task.error_message" class="rounded-lg border border-red-200 bg-red-50 p-3 text-red-700 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-300">
            {{ task.error_message }}
          </div>
        </div>
      </section>

      <details v-if="task" class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
        <summary class="cursor-pointer text-base font-semibold text-gray-900 dark:text-white">Technical Details</summary>
        <pre class="mt-4 max-h-96 overflow-auto rounded-md bg-gray-50 p-4 text-xs text-gray-700 dark:bg-dark-900 dark:text-gray-200">{{ JSON.stringify(task, null, 2) }}</pre>
      </details>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { videoTaskAPI, type VideoTask } from '@/api/admin/video'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDate, isTerminalStatus, providerLabel, shortText, statusBadgeClass, statusLabel, taskTypeLabel } from './videoUtils'

const route = useRoute()
const appStore = useAppStore()
const task = ref<VideoTask | null>(null)
const loading = ref(false)
const cancelling = ref(false)
let refreshTimer: ReturnType<typeof setInterval> | null = null

const taskId = computed(() => Number(route.params.id))
const parameterRows = computed(() => {
  if (!task.value) return []
  return [
    { label: 'Task Type', value: taskTypeLabel(task.value.task_type) },
    { label: 'Prompt', value: task.value.prompt },
    { label: 'Negative Prompt', value: task.value.negative_prompt },
    { label: 'Reference Image', value: task.value.reference_image_url },
    { label: 'Reference Video', value: task.value.reference_video_url },
    { label: 'Aspect Ratio', value: task.value.aspect_ratio },
    { label: 'Duration', value: `${task.value.duration}s` },
    { label: 'Resolution', value: task.value.resolution },
    { label: 'Upstream Task ID', value: task.value.upstream_task_id },
  ]
})

function eventDotClass(eventType: string): string {
  if (eventType === 'succeeded') return 'bg-emerald-500'
  if (eventType === 'failed') return 'bg-red-500'
  if (eventType === 'running') return 'bg-amber-500'
  return 'bg-gray-400'
}

async function loadTask() {
  loading.value = true
  try {
    task.value = await videoTaskAPI.get(taskId.value)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, 'Failed to load video task'))
  } finally {
    loading.value = false
  }
}

async function cancelTask() {
  if (!task.value) return
  cancelling.value = true
  try {
    task.value = await videoTaskAPI.cancel(task.value.id)
    appStore.showSuccess('Video task cancelled')
    await loadTask()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, 'Failed to cancel video task'))
  } finally {
    cancelling.value = false
  }
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
