<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 border-b border-gray-200 pb-4 dark:border-dark-700 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ isVideoGatewayDemoMode ? 'API 调用任务' : '任务列表' }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ isVideoGatewayDemoMode ? '查看所有通过网关提交的视频生成调用，按 API 通道、状态和失败原因追踪结果。' : '按状态和通道查看最近任务，快速定位结果、失败原因并复制参数重新创建。' }}
          </p>
        </div>
        <div class="flex flex-wrap gap-2">
          <RouterLink class="btn btn-primary" to="/admin/video/create">
            <Icon name="plus" size="sm" />
            {{ isVideoGatewayDemoMode ? '发起调用' : '创建任务' }}
          </RouterLink>
          <button class="btn btn-outline" type="button" :disabled="loading" @click="loadTasks">
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
            刷新
          </button>
        </div>
      </div>

      <section class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
        <div class="grid gap-3 md:grid-cols-[220px_220px_1fr]">
          <select v-model="filters.status" class="input" @change="resetAndLoad">
            <option value="">全部状态</option>
            <option v-for="status in statusOptions" :key="status.value" :value="status.value">{{ status.label }}</option>
          </select>
          <select v-model="filters.provider" class="input" @change="resetAndLoad">
            <option value="">{{ isVideoGatewayDemoMode ? '全部 API 通道' : '全部通道' }}</option>
            <option v-for="provider in providerOptions" :key="provider.value" :value="provider.value">{{ provider.label }}</option>
          </select>
          <div class="flex justify-end">
            <button class="btn btn-outline" type="button" @click="clearFilters">
              <Icon name="x" size="sm" />
              清空筛选
            </button>
          </div>
        </div>
      </section>

      <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs uppercase text-gray-500 dark:bg-dark-700/40 dark:text-gray-400">
              <tr>
                <th class="px-5 py-3 font-medium">{{ isVideoGatewayDemoMode ? '调用任务' : '任务' }}</th>
                <th v-if="authStore.isAdmin" class="px-5 py-3 font-medium">发起人</th>
                <th class="px-5 py-3 font-medium">{{ isVideoGatewayDemoMode ? 'API 通道' : '通道' }}</th>
                <th class="px-5 py-3 font-medium">{{ isVideoGatewayDemoMode ? '调用状态' : '状态' }}</th>
                <th class="px-5 py-3 font-medium">创建时间</th>
                <th class="px-5 py-3 font-medium">结果入口</th>
                <th class="px-5 py-3 font-medium">操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="task in tasks" :key="task.id" class="hover:bg-gray-50 dark:hover:bg-dark-700/40">
                <td class="px-5 py-3">
                  <RouterLink class="font-medium text-gray-900 hover:text-primary-600 dark:text-white dark:hover:text-primary-300" :to="`/admin/video/tasks/${task.id}`">
                    #{{ task.id }} | {{ taskTypeLabel(task.task_type) }}
                  </RouterLink>
                  <div class="prompt-summary mt-1 max-w-xl text-xs text-gray-500 dark:text-gray-400">{{ promptDisplayText(task.prompt) }}</div>
                  <div v-if="task.error_message" class="mt-1 max-w-xl truncate text-xs text-red-600 dark:text-red-300">
                    {{ isVideoGatewayDemoMode ? '调用失败原因' : '失败原因' }}：{{ errorMessageLabel(task.error_message) }}
                  </div>
                </td>
                <td v-if="authStore.isAdmin" class="px-5 py-3 text-gray-700 dark:text-gray-200">
                  {{ createdByLabel(task) }}
                </td>
                <td class="px-5 py-3">
                  <span class="inline-flex rounded-md px-2 py-1 text-xs font-medium" :class="providerBadgeClass(task.provider)">
                    {{ providerLabel(task.provider) }}
                  </span>
                  <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">路由账号：{{ routeAccountLabel(task) }}</div>
                  <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ modelDisplayName(task.provider, task.model) }}</div>
                </td>
                <td class="px-5 py-3">
                  <span class="inline-flex rounded-md px-2 py-1 text-xs font-medium" :class="statusBadgeClass(task.status)">
                    {{ statusLabel(task.status) }}
                  </span>
                </td>
                <td class="px-5 py-3 text-gray-500 dark:text-gray-400">{{ formatDate(task.created_at) }}</td>
                <td class="px-5 py-3">
                  <a v-if="task.result_url" class="inline-flex items-center gap-1 text-primary-600 hover:text-primary-700 dark:text-primary-300" :href="task.result_url" target="_blank" rel="noreferrer">
                    <Icon name="externalLink" size="xs" />
                    打开结果
                  </a>
                  <span v-else class="text-gray-400">-</span>
                </td>
                <td class="px-5 py-3">
                  <div class="flex flex-wrap gap-2">
                    <a v-if="task.result_url" class="btn btn-sm btn-outline" :href="task.result_url" target="_blank" rel="noreferrer">打开结果</a>
                    <button class="btn btn-sm btn-outline" type="button" @click="copyToCreate(task)">
                      {{ isVideoGatewayDemoMode ? '复制参数重新发起' : '复制参数' }}
                    </button>
                    <RouterLink class="btn btn-sm btn-outline" :to="`/admin/video/tasks/${task.id}`">
                      {{ isVideoGatewayDemoMode ? '查看调用详情' : '查看详情' }}
                    </RouterLink>
                  </div>
                </td>
              </tr>
              <tr v-if="!loading && !tasks.length">
                <td :colspan="authStore.isAdmin ? 7 : 6" class="px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400">
                  <div class="space-y-3">
                    <div>{{ isVideoGatewayDemoMode ? '当前还没有调用任务。你可以先通过演示通道发起一次调用，验证网关流转。' : '没有任务。可以先创建一个演示任务验证流转。' }}</div>
                    <RouterLink class="btn btn-sm btn-outline" to="/admin/video/create">{{ isVideoGatewayDemoMode ? '发起一次演示调用' : '创建一个演示任务' }}</RouterLink>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="flex items-center justify-between border-t border-gray-200 px-5 py-4 text-sm dark:border-dark-700">
          <div class="text-gray-500 dark:text-gray-400">共 {{ pagination.total }} 条</div>
          <div class="flex gap-2">
            <button class="btn btn-sm btn-outline" type="button" :disabled="pagination.page <= 1" @click="changePage(pagination.page - 1)">上一页</button>
            <span class="flex items-center px-2 text-gray-700 dark:text-gray-200">{{ pagination.page }} / {{ pagination.pages }}</span>
            <button class="btn btn-sm btn-outline" type="button" :disabled="pagination.page >= pagination.pages" @click="changePage(pagination.page + 1)">下一页</button>
          </div>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { videoTaskAPI, type VideoProvider, type VideoTask, type VideoTaskListParams, type VideoTaskStatus } from '@/api/admin/video'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { extractApiErrorMessage } from '@/utils/apiError'
import { isVideoGatewayDemoMode } from '@/utils/productMode'
import {
  errorMessageLabel,
  formatDate,
  modelDisplayName,
  createdByLabel,
  providerBadgeClass,
  providerLabel,
  providerOptions,
  promptDisplayText,
  routeAccountLabel,
  saveTaskDraft,
  statusBadgeClass,
  statusLabel,
  statusOptions,
  taskTypeLabel,
} from './videoUtils'

const appStore = useAppStore()
const authStore = useAuthStore()
const router = useRouter()
const loading = ref(false)
const tasks = ref<VideoTask[]>([])
const filters = reactive({ status: '', provider: '' })
const pagination = reactive({ page: 1, page_size: 20, total: 0, pages: 1 })

async function loadTasks() {
  loading.value = true
  try {
    const params: VideoTaskListParams = {
      page: pagination.page,
      page_size: pagination.page_size,
    }
    if (filters.status) params.status = filters.status as VideoTaskStatus
    if (filters.provider) params.provider = filters.provider as VideoProvider
    const res = await videoTaskAPI.list(params)
    tasks.value = res.items || []
    pagination.total = res.total
    pagination.page = res.page
    pagination.page_size = res.page_size
    pagination.pages = res.pages
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, isVideoGatewayDemoMode ? '加载 API 调用任务失败' : '加载任务列表失败'))
  } finally {
    loading.value = false
  }
}

function resetAndLoad() {
  pagination.page = 1
  loadTasks()
}

function clearFilters() {
  filters.status = ''
  filters.provider = ''
  resetAndLoad()
}

function changePage(page: number) {
  pagination.page = page
  loadTasks()
}

function copyToCreate(task: VideoTask) {
  saveTaskDraft(task)
  appStore.showInfo(isVideoGatewayDemoMode ? '已复制调用参数，可在发起调用页调整后重新提交。' : '已复制任务参数，可在创建页调整后重新提交。')
  router.push('/admin/video/create')
}

onMounted(loadTasks)
</script>

<style scoped>
.prompt-summary {
  display: -webkit-box;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}
</style>
