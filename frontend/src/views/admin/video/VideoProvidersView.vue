<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 border-b border-gray-200 pb-4 dark:border-dark-700 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ isVideoGatewayDemoMode ? 'API 通道池' : '模型通道' }}</h1>
          <p class="mt-1 max-w-4xl text-sm text-gray-500 dark:text-gray-400">
            {{ isVideoGatewayDemoMode ? '集中管理企业可用的视频模型 API 通道。Key 加密保存，前端只展示脱敏状态；调用任务由网关按启用状态、限流和队列统一分发。' : '管理演示通道与未来真实模型通道的启用状态和调用凭证。' }}
          </p>
        </div>
        <button class="btn btn-outline" type="button" :disabled="loading" @click="loadProviders">
          <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
          刷新
        </button>
      </div>

      <div class="grid gap-6 xl:grid-cols-[1.35fr_0.65fr]">
        <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <div class="overflow-x-auto">
            <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
              <thead class="bg-gray-50 text-left text-xs uppercase text-gray-500 dark:bg-dark-700/40 dark:text-gray-400">
                <tr>
                  <th class="px-5 py-3 font-medium">通道名称</th>
                  <th class="px-5 py-3 font-medium">Key 状态</th>
                  <th class="px-5 py-3 font-medium">启用状态</th>
                  <th class="px-5 py-3 font-medium">今日调用</th>
                  <th class="px-5 py-3 font-medium">当前处理中</th>
                  <th class="px-5 py-3 font-medium">最近异常</th>
                  <th class="px-5 py-3 font-medium">操作</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                <tr v-for="provider in providers" :key="provider.id" :class="selected?.id === provider.id ? 'bg-gray-50 dark:bg-dark-700/40' : ''">
                  <td class="px-5 py-3">
                    <span class="inline-flex rounded-md px-2 py-1 text-xs font-medium" :class="providerBadgeClass(provider.provider)">
                      {{ providerDisplayName(provider) }}
                    </span>
                    <div class="mt-1 max-w-xs text-xs text-gray-500 dark:text-gray-400">{{ providerDescription(provider.provider) }}</div>
                  </td>
                  <td class="px-5 py-3">
                    <span class="inline-flex rounded-md px-2 py-1 text-xs font-medium" :class="keyStatusClass(provider.key_status)">
                      {{ providerKeyLabel(provider.api_key_configured, provider.masked_key, provider.key_status) }}
                    </span>
                  </td>
                  <td class="px-5 py-3">
                    <button
                      type="button"
                      class="inline-flex rounded-md px-2 py-1 text-xs font-medium"
                      :class="providerRuntimeStatusClass(providerRuntimeStatus(provider))"
                      :disabled="savingId === provider.id"
                      @click="toggleEnabled(provider)"
                    >
                      {{ providerRuntimeStatus(provider) }}
                    </button>
                  </td>
                  <td class="px-5 py-3 text-gray-700 dark:text-gray-200">{{ provider.today_tasks }}</td>
                  <td class="px-5 py-3 text-gray-700 dark:text-gray-200">{{ provider.current_inflight }}</td>
                  <td class="px-5 py-3">
                    <div class="max-w-xs text-xs text-gray-600 dark:text-gray-300">
                      {{ provider.last_error || provider.diagnostic_type || provider.route_skip_reason || '-' }}
                    </div>
                    <div v-if="provider.suggested_action" class="mt-1 max-w-xs text-xs text-amber-700 dark:text-amber-300">
                      {{ provider.suggested_action }}
                    </div>
                  </td>
                  <td class="px-5 py-3">
                    <div class="flex flex-wrap gap-2">
                      <button class="btn btn-sm btn-outline" type="button" @click="selectProvider(provider)">
                        <Icon name="edit" size="xs" />
                        编辑
                      </button>
                      <button class="btn btn-sm btn-outline" type="button" :disabled="testingId === provider.id" @click="testProvider(provider.id)">
                        <Icon name="beaker" size="xs" />
                        测试
                      </button>
                      <button class="btn btn-sm btn-outline" type="button" :disabled="savingId === provider.id" @click="toggleEnabled(provider)">
                        {{ provider.enabled ? '停用' : '启用' }}
                      </button>
                    </div>
                  </td>
                </tr>
                <tr v-if="!loading && !providers.length">
                  <td colspan="7" class="px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400">
                    {{ isVideoGatewayDemoMode ? '暂无 API 通道，请确认初始化数据已写入。' : '暂无通道，请确认数据库迁移已执行。' }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <section class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ isVideoGatewayDemoMode ? 'API 通道配置' : '通道配置' }}</h2>
          <p v-if="isVideoGatewayDemoMode" class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            API Key 将加密保存，前端只显示脱敏状态。留空表示保留当前 Key。
          </p>
          <form v-if="selected" class="mt-4 space-y-4" @submit.prevent="saveProvider">
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">展示名称</label>
              <input v-model="form.display_name" class="input" maxlength="120" />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ isVideoGatewayDemoMode ? 'API 入口地址' : '上游地址' }}</label>
              <input v-model="form.base_url" class="input" maxlength="500" />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ isVideoGatewayDemoMode ? '默认 API 模型' : '默认模型' }}</label>
              <input v-model="form.default_model" class="input" maxlength="200" />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">API Key</label>
              <input v-model="form.api_key" class="input" type="password" autocomplete="off" placeholder="留空表示保留当前 Key" maxlength="4000" />
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">当前：{{ providerKeyLabel(selected.api_key_configured, selected.masked_key) }}</p>
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">每分钟限额</label>
              <input v-model.number="form.rate_limit_per_minute" class="input" type="number" min="0" max="100000" />
            </div>
            <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
              <input v-model="form.enabled" type="checkbox" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
              {{ isVideoGatewayDemoMode ? '允许员工通过该 API 通道调用' : '启用该通道' }}
            </label>
            <div class="flex flex-wrap gap-2">
              <button class="btn btn-primary" type="submit" :disabled="saving">
                <Icon name="check" size="sm" />
                保存
              </button>
              <button class="btn btn-outline" type="button" :disabled="testingId === selected.id" @click="testProvider(selected.id)">
                <Icon name="beaker" size="sm" />
                测试
              </button>
            </div>
          </form>
          <div v-else class="mt-6 text-sm text-gray-500 dark:text-gray-400">{{ isVideoGatewayDemoMode ? '请选择一个 API 通道进行编辑或测试。' : '请选择一个通道进行编辑或测试。' }}</div>

          <div v-if="testResult" class="mt-5 rounded-lg border border-gray-200 bg-gray-50 p-4 text-sm dark:border-dark-700 dark:bg-dark-700/40">
            <div class="flex items-center justify-between gap-3">
              <span class="font-medium text-gray-900 dark:text-white">最近一次测试</span>
              <span class="rounded-md px-2 py-1 text-xs font-medium" :class="testResult.reachable ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300' : 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-200'">
                {{ testResult.reachable ? '可用' : '已跳过' }}
              </span>
            </div>
            <p class="mt-2 text-gray-600 dark:text-gray-300">{{ providerTestMessage(testResult.message) }}</p>
            <details v-if="!isVideoGatewayDemoMode" class="mt-3">
              <summary class="cursor-pointer text-xs font-medium text-gray-500 dark:text-gray-400">技术 payload 预览</summary>
              <pre class="mt-2 max-h-64 overflow-auto rounded-md bg-white p-3 text-xs text-gray-700 dark:bg-dark-900 dark:text-gray-200">{{ JSON.stringify(testResult.payload_preview || {}, null, 2) }}</pre>
            </details>
          </div>
        </section>
      </div>

      <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="border-b border-gray-200 px-5 py-4 dark:border-dark-700">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">API 健康诊断</h2>
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs uppercase text-gray-500 dark:bg-dark-700/40 dark:text-gray-400">
              <tr>
                <th class="px-5 py-3 font-medium">API 通道</th>
                <th class="px-5 py-3 font-medium">路由账号</th>
                <th class="px-5 py-3 font-medium">Key 状态</th>
                <th class="px-5 py-3 font-medium">最近测试</th>
                <th class="px-5 py-3 font-medium">异常类型</th>
                <th class="px-5 py-3 font-medium">影响任务数</th>
                <th class="px-5 py-3 font-medium">最近错误</th>
                <th class="px-5 py-3 font-medium">建议动作</th>
                <th class="px-5 py-3 font-medium">状态</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="item in dashboard?.health_diagnostics || []" :key="`${item.provider}-${item.route_account}`">
                <td class="px-5 py-3 text-gray-700 dark:text-gray-200">{{ providerLabel(item.provider) }}</td>
                <td class="px-5 py-3 text-gray-700 dark:text-gray-200">{{ item.route_account }}</td>
                <td class="px-5 py-3">
                  <span class="inline-flex rounded-md px-2 py-1 text-xs font-medium" :class="keyStatusClass(item.key_status)">
                    {{ providerKeyLabel(item.key_status === 'normal', '', item.key_status) }}
                  </span>
                </td>
                <td class="px-5 py-3 text-gray-500 dark:text-gray-400">{{ formatDate(item.last_test_at) }}</td>
                <td class="px-5 py-3 text-gray-700 dark:text-gray-200">{{ item.exception_type }}</td>
                <td class="px-5 py-3 text-gray-700 dark:text-gray-200">{{ item.impact_tasks }}</td>
                <td class="px-5 py-3 text-gray-700 dark:text-gray-200">{{ item.recent_error || '-' }}</td>
                <td class="px-5 py-3 text-amber-700 dark:text-amber-300">{{ item.suggested_action || '-' }}</td>
                <td class="px-5 py-3">
                  <span class="inline-flex rounded-md px-2 py-1 text-xs font-medium" :class="item.status === '正常' ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300' : 'bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'">
                    {{ item.status }}
                  </span>
                </td>
              </tr>
              <tr v-if="!(dashboard?.health_diagnostics || []).length">
                <td colspan="9" class="px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400">暂无诊断数据。</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type { VideoDashboard, VideoProviderAccount, VideoProviderTestResult } from '@/api/admin/video'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { isVideoGatewayDemoMode } from '@/utils/productMode'
import {
  formatDate,
  keyStatusClass,
  providerBadgeClass,
  providerDescription,
  providerDisplayName,
  providerKeyLabel,
  providerLabel,
  providerRuntimeStatus,
  providerRuntimeStatusClass,
  providerTestMessage,
} from './videoUtils'

const appStore = useAppStore()
const loading = ref(false)
const saving = ref(false)
const savingId = ref<number | null>(null)
const testingId = ref<number | null>(null)
const providers = ref<VideoProviderAccount[]>([])
const selected = ref<VideoProviderAccount | null>(null)
const testResult = ref<VideoProviderTestResult | null>(null)
const dashboard = ref<VideoDashboard | null>(null)
const testResultsByProvider = ref<Record<number, VideoProviderTestResult>>({})

const form = reactive({
  display_name: '',
  enabled: false,
  api_key: '',
  base_url: '',
  default_model: '',
  rate_limit_per_minute: 60,
})

function selectProvider(provider: VideoProviderAccount) {
  selected.value = provider
  form.display_name = providerDisplayName(provider)
  form.enabled = provider.enabled
  form.api_key = ''
  form.base_url = provider.base_url
  form.default_model = provider.default_model
  form.rate_limit_per_minute = provider.rate_limit_per_minute
  testResult.value = testResultsByProvider.value[provider.id] || null
}

async function loadProviders() {
  loading.value = true
  try {
    const [providerRes, dashboardRes] = await Promise.all([
      adminAPI.video.listProviders(),
      adminAPI.video.dashboard(),
    ])
    providers.value = providerRes.items || []
    dashboard.value = dashboardRes
    const current = selected.value ? providers.value.find((item) => item.id === selected.value?.id) : providers.value[0]
    if (current) selectProvider(current)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, isVideoGatewayDemoMode ? '加载 API 通道池失败' : '加载模型通道失败'))
  } finally {
    loading.value = false
  }
}

async function saveProvider() {
  if (!selected.value) return
  saving.value = true
  try {
    await adminAPI.video.updateProvider(selected.value.id, {
      display_name: form.display_name,
      enabled: form.enabled,
      base_url: form.base_url,
      default_model: form.default_model,
      rate_limit_per_minute: form.rate_limit_per_minute,
      ...(form.api_key.trim() ? { api_key: form.api_key.trim() } : {}),
    })
    form.api_key = ''
    appStore.showSuccess(isVideoGatewayDemoMode ? 'API 通道配置已保存' : '通道配置已保存')
    await loadProviders()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, isVideoGatewayDemoMode ? '保存 API 通道配置失败' : '保存通道配置失败'))
  } finally {
    saving.value = false
  }
}

async function toggleEnabled(provider: VideoProviderAccount) {
  savingId.value = provider.id
  try {
    await adminAPI.video.updateProvider(provider.id, { enabled: !provider.enabled })
    await loadProviders()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, isVideoGatewayDemoMode ? '更新 API 通道状态失败' : '更新通道状态失败'))
  } finally {
    savingId.value = null
  }
}

async function testProvider(id: number) {
  testingId.value = id
  try {
    const result = await adminAPI.video.testProvider(id)
    testResultsByProvider.value = { ...testResultsByProvider.value, [id]: result }
    testResult.value = result
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, isVideoGatewayDemoMode ? 'API 通道测试失败' : '通道测试失败'))
  } finally {
    testingId.value = null
  }
}

onMounted(loadProviders)
</script>
