<template>
  <AppLayout>
    <div class="video-tasks-view min-w-0 space-y-5 overflow-x-clip">
      <header class="video-tasks-header">
        <h2 class="video-tasks-title ui-heading">视频任务证据</h2>
        <p class="video-tasks-description ui-subheading mt-1">按真实任务记录核对请求人、模型、终态、上下游 ID、资产、费用和调度结果。</p>
      </header>

      <div class="video-task-filters ui-toolbar flex flex-col gap-3 sm:flex-row sm:items-end">
        <div class="video-task-status-field w-full sm:max-w-52">
          <label for="video-task-status" class="mb-1 block text-sm font-medium">任务状态</label>
          <select id="video-task-status" v-model="status" class="input w-full" @change="load">
            <option value="">全部状态</option>
            <option v-for="item in states" :key="item" :value="item">{{ item }}</option>
          </select>
        </div>
        <button type="button" class="btn btn-secondary" @click="load">刷新</button>
      </div>

      <p id="video-task-scroll-hint" class="video-task-scroll-hint text-xs text-gray-500">
        窄屏时可在下方表格区域横向滚动；本地任务与状态列会保持可见。
      </p>
      <div
        class="video-task-table-shell ui-panel max-w-full overflow-x-auto focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40"
        tabindex="0"
        role="region"
        aria-label="视频任务证据表"
        aria-describedby="video-task-scroll-hint"
      >
        <table class="video-task-table w-full min-w-[1060px] text-left text-sm">
          <thead>
            <tr>
              <th class="video-task-sticky-id bg-white p-3 dark:bg-dark-900">本地任务</th>
              <th class="video-task-sticky-status bg-white p-3 dark:bg-dark-900">终态</th>
              <th class="p-3">请求人</th>
              <th class="p-3">模型</th>
              <th class="p-3">上游任务 ID</th>
              <th class="p-3">费用</th>
              <th class="p-3">真实调度</th>
              <th class="p-3">资产 / 错误</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="task in tasks" :key="task.id" class="video-task-row align-top">
              <td class="video-task-sticky-id bg-white p-3 dark:bg-dark-900">
                <RouterLink class="font-medium text-primary-600" :to="`/admin/video/tasks/${task.id}`" :aria-label="`查看本地任务 ${task.id} 的完整证据`">#{{ task.id }}</RouterLink>
                <div class="max-w-48 truncate text-xs text-gray-500" :title="task.prompt">{{ task.prompt }}</div>
              </td>
              <td class="video-task-sticky-status bg-white p-3 dark:bg-dark-900"><TaskProgressRing :phase="taskPhaseLabel(task.status)" :size="26" side-label /></td>
              <td class="p-3">员工 #{{ task.created_by }}</td>
              <td class="max-w-48 break-all p-3">{{ task.model || '后端未提供' }}</td>
              <td class="max-w-56 break-all p-3 font-mono text-xs">{{ task.upstream_task_id || '未返回' }}</td>
              <td class="whitespace-nowrap p-3">{{ task.cost_amount }} {{ task.currency || 'USD' }}</td>
              <td class="p-3">{{ task.real_dispatch_count }}</td>
              <td class="max-w-64 p-3">
                <a v-if="task.result_url" :href="task.result_url" target="_blank" rel="noreferrer" class="break-all text-primary-600" :aria-label="`打开任务 ${task.id} 的结果资产`">打开结果资产</a>
                <span v-else class="break-words text-red-600">{{ task.error_message || task.provider_error_message || '尚无结果资产' }}</span>
              </td>
            </tr>
          </tbody>
        </table>
        <div v-if="!tasks.length" class="p-2">
          <AnimatedEmptyState
            variant="video-tasks"
            title="暂无任务证据"
            description="有任务落库后，这里会展示终态、上下游 ID 与资产证据。"
          />
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import AnimatedEmptyState from '@/components/common/AnimatedEmptyState.vue'
import TaskProgressRing from '@/components/common/TaskProgressRing.vue'
import { adminAPI } from '@/api/admin'
import type { VideoTaskAdmin } from '@/api/admin/video'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'

const app = useAppStore()
const tasks = ref<VideoTaskAdmin[]>([])
const status = ref('')
function taskPhaseLabel(status: string): string {
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
    default:
      return status || '未知'
  }
}

const states = ['queued', 'submitted', 'running', 'succeeded', 'failed', 'cancelled']

async function load() {
  try {
    tasks.value = (await adminAPI.video.listTasks(1, 100, status.value)).items
  } catch (error) {
    app.showError(extractApiErrorMessage(error, '加载视频任务证据失败'))
  }
}

onMounted(load)
</script>

<style scoped>
/* Hallmark · pre-emit critique: P5 H5 E4 S5 R5 V4 · macrostructure: operator-evidence-table */
.video-tasks-view {
  overflow-x: clip;
}

.video-task-table-shell {
  overscroll-behavior-x: contain;
}

.video-task-sticky-id,
.video-task-sticky-status {
  position: sticky;
  z-index: 1;
  background: inherit;
}

.video-task-sticky-id {
  left: 0;
  min-width: 11rem;
}

.video-task-sticky-status {
  left: 11rem;
  min-width: 6.5rem;
}

thead .video-task-sticky-id,
thead .video-task-sticky-status {
  z-index: 2;
}

@media (max-width: 639px) {
  .video-task-sticky-id {
    min-width: 8.5rem;
  }

  .video-task-sticky-status {
    left: 8.5rem;
  }
}
</style>
