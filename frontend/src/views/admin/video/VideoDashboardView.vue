<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 border-b border-gray-200 pb-4 dark:border-dark-700 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">AI 视频网关</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">查看视频任务吞吐、通道状态和 Mock 演示结果</p>
        </div>
        <div class="flex flex-wrap gap-2">
          <RouterLink class="btn btn-outline" to="/admin/video/providers">
            <Icon name="server" size="sm" />
            模型通道
          </RouterLink>
          <RouterLink class="btn btn-primary" to="/admin/video/create">
            <Icon name="plus" size="sm" />
            创建任务
          </RouterLink>
          <button class="btn btn-outline" type="button" :disabled="loading" @click="loadDashboard">
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
            刷新
          </button>
        </div>
      </div>

      <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
        <div v-for="item in statItems" :key="item.label" class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
          <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ item.label }}</div>
          <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ item.value }}</div>
        </div>
      </div>

      <div class="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
        <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <div class="border-b border-gray-200 px-5 py-4 dark:border-dark-700">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">通道状态</h2>
          </div>
          <div class="overflow-x-auto">
            <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
              <thead class="bg-gray-50 text-left text-xs uppercase text-gray-500 dark:bg-dark-700/40 dark:text-gray-400">
                <tr>
                  <th class="px-5 py-3 font-medium">供应商</th>
                  <th class="px-5 py-3 font-medium">启用状态</th>
                  <th class="px-5 py-3 font-medium">Key 状态</th>
                  <th class="px-5 py-3 font-medium">今日任务</th>
                  <th class="px-5 py-3 font-medium">处理中</th>
                  <th class="px-5 py-3 font-medium">失败</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                <tr v-for="provider in dashboard?.provider_status || []" :key="provider.provider">
                  <td class="px-5 py-3">
                    <span class="inline-flex rounded-md px-2 py-1 text-xs font-medium" :class="providerBadgeClass(provider.provider)">
                      {{ provider.display_name }}
                    </span>
                    <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ provider.default_model || '-' }}</div>
                  </td>
                  <td class="px-5 py-3">
                    <span
                      class="inline-flex rounded-md px-2 py-1 text-xs font-medium"
                      :class="provider.enabled ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300' : 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-200'"
                    >
                      {{ providerEnabledLabel(provider.enabled) }}
                    </span>
                  </td>
                  <td class="px-5 py-3 text-gray-700 dark:text-gray-200">{{ providerKeyLabel(provider.api_key_configured, provider.masked_key) }}</td>
                  <td class="px-5 py-3 text-gray-700 dark:text-gray-200">{{ provider.today_tasks }}</td>
                  <td class="px-5 py-3 text-gray-700 dark:text-gray-200">{{ provider.running_tasks }}</td>
                  <td class="px-5 py-3 text-gray-700 dark:text-gray-200">{{ provider.failed_tasks }}</td>
                </tr>
                <tr v-if="!loading && !(dashboard?.provider_status || []).length">
                  <td colspan="6" class="px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400">暂无通道</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <div class="border-b border-gray-200 px-5 py-4 dark:border-dark-700">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">用量概览</h2>
          </div>
          <div class="divide-y divide-gray-100 dark:divide-dark-700">
            <div v-for="item in dashboard?.usage_overview || []" :key="`${item.provider}-${item.model}-${item.status}`" class="flex items-center justify-between px-5 py-3 text-sm">
              <div>
                <span class="font-medium text-gray-900 dark:text-white">{{ providerLabel(item.provider) }}</span>
                <span class="ml-2 text-gray-500 dark:text-gray-400">{{ item.model }}</span>
              </div>
              <div class="flex items-center gap-2">
                <span class="rounded-md px-2 py-1 text-xs font-medium" :class="statusBadgeClass(item.status)">{{ statusLabel(item.status) }}</span>
                <span class="text-gray-700 dark:text-gray-200">{{ item.count }}</span>
              </div>
            </div>
            <div v-if="!loading && !(dashboard?.usage_overview || []).length" class="px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400">暂无用量记录</div>
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
import { RouterLink } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type { VideoDashboard, VideoTaskSummary } from '@/api/admin/video'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import {
  errorMessageLabel,
  formatDate,
  providerBadgeClass,
  providerEnabledLabel,
  providerKeyLabel,
  providerLabel,
  shortText,
  statusBadgeClass,
  statusLabel,
} from './videoUtils'

const appStore = useAppStore()
const loading = ref(false)
const dashboard = ref<VideoDashboard | null>(null)

const statItems = computed(() => {
  const d = dashboard.value
  return [
    { label: '今日任务', value: d?.today_tasks ?? 0 },
    { label: '成功率', value: `${Math.round(d?.success_rate ?? 0)}%` },
    { label: '失败', value: d?.failed_tasks ?? 0 },
    { label: '处理中', value: d?.running_tasks ?? 0 },
    { label: '排队中', value: d?.queued_tasks ?? 0 },
  ]
})

async function loadDashboard() {
  loading.value = true
  try {
    dashboard.value = await adminAPI.video.dashboard()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载视频总览失败'))
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
                      h('div', { class: 'truncate text-sm font-medium text-gray-900 dark:text-white' }, shortText(task.prompt, 120)),
                      h('div', { class: 'mt-1 text-xs text-gray-500 dark:text-gray-400' }, `${providerLabel(task.provider)} · ${formatDate(task.updated_at)}`),
                    ]),
                    h('span', { class: ['shrink-0 rounded-md px-2 py-1 text-xs font-medium', statusBadgeClass(task.status)] }, statusLabel(task.status)),
                  ]),
                  task.error_message ? h('div', { class: 'mt-2 text-xs text-red-600 dark:text-red-300' }, shortText(errorMessageLabel(task.error_message), 140)) : null,
                ]),
              )
            : [h('div', { class: 'px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400' }, '暂无任务')],
        ),
      ])
  },
})

onMounted(loadDashboard)
</script>
