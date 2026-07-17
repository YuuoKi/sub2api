<template>
  <AppLayout>
    <div class="video-create-view ui-page min-w-0 space-y-5">
      <header>
        <h2 class="ui-heading">创建任务</h2>
        <p class="ui-subheading mt-1">
          使用内部模拟通道创建任务。零费用、无外网，结果为标注模拟的图像预览。
        </p>
      </header>

      <div v-if="contract" class="ui-panel space-y-2 p-4 text-sm">
        <p>
          <span class="font-medium">通道：</span>{{ contract.label }}（{{ contract.provider }} / {{ contract.model }}）
        </p>
        <p>
          <span class="font-medium">规格：</span>{{ contract.resolution }} · {{ contract.duration_seconds }}s ·
          {{ contract.media_kind }}
        </p>
        <p class="text-emerald-700 dark:text-emerald-400">
          费用：{{ contract.cost_amount }} {{ contract.currency }}（内部模拟 · 不计费 · 无外网）
        </p>
      </div>
      <p v-else-if="contractError" class="text-sm text-red-600">{{ contractError }}</p>
      <p v-else class="text-sm text-gray-500">正在加载模拟合同…</p>

      <form class="ui-panel max-w-xl space-y-4 p-4" @submit.prevent="onSubmit">
        <div>
          <label for="sim-api-key" class="mb-1 block text-sm font-medium">API 密钥</label>
          <select id="sim-api-key" v-model="apiKeyId" class="input w-full" required :disabled="loadingKeys || submitting">
            <option disabled value="">选择本人启用中的密钥</option>
            <option v-for="key in activeKeys" :key="key.id" :value="String(key.id)">
              {{ key.name || `密钥 #${key.id}` }}
            </option>
          </select>
          <p v-if="!loadingKeys && activeKeys.length === 0" class="mt-1 text-xs text-amber-600">
            没有可用的启用密钥。请先在「我的密钥」中创建并启用。
          </p>
        </div>

        <div>
          <label for="sim-prompt" class="mb-1 block text-sm font-medium">提示词</label>
          <textarea
            id="sim-prompt"
            v-model="prompt"
            class="input min-h-28 w-full"
            required
            maxlength="8000"
            placeholder="描述要模拟生成的内容"
            :disabled="submitting"
          />
        </div>

        <p v-if="submitError" class="text-sm text-red-600" role="alert">{{ submitError }}</p>
        <p v-if="createdTaskId" class="text-sm text-emerald-700 dark:text-emerald-400">
          已创建任务 #{{ createdTaskId }}。
          <RouterLink class="text-primary-600 underline" :to="`/video/tasks/${createdTaskId}`">查看详情</RouterLink>
        </p>

        <button type="submit" class="btn btn-primary" :disabled="!canSubmit">
          {{ submitting ? '创建中…' : '创建模拟任务' }}
        </button>
      </form>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import { keysAPI } from '@/api/keys'
import {
  createSimulationTask,
  getSimulationContract,
  type VideoSimulationContract,
} from '@/api/user/video_simulation'
import type { ApiKey } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'

const contract = ref<VideoSimulationContract | null>(null)
const contractError = ref('')
const activeKeys = ref<ApiKey[]>([])
const loadingKeys = ref(true)
const apiKeyId = ref('')
const prompt = ref('')
const submitting = ref(false)
const submitError = ref('')
const createdTaskId = ref<number | null>(null)

const canSubmit = computed(
  () => !!contract.value && !!apiKeyId.value && prompt.value.trim().length > 0 && !submitting.value,
)

async function loadContract() {
  try {
    contract.value = await getSimulationContract()
    contractError.value = ''
  } catch (error) {
    contract.value = null
    contractError.value = extractApiErrorMessage(error, '无法加载模拟合同')
  }
}

async function loadKeys() {
  loadingKeys.value = true
  try {
    const page = await keysAPI.list(1, 100, { status: 'active' })
    activeKeys.value = page.items ?? []
  } catch (error) {
    activeKeys.value = []
    submitError.value = extractApiErrorMessage(error, '加载密钥失败')
  } finally {
    loadingKeys.value = false
  }
}

async function onSubmit() {
  if (!canSubmit.value) return
  submitting.value = true
  submitError.value = ''
  createdTaskId.value = null
  // One creation_key per Create click (intentional submit). Retries after failure
  // mint a new key on the next click so each user action is a distinct attempt.
  const creationKey = crypto.randomUUID()
  try {
    const task = await createSimulationTask({
      api_key_id: Number(apiKeyId.value),
      prompt: prompt.value.trim(),
      creation_key: creationKey,
    })
    createdTaskId.value = task.id
  } catch (error) {
    submitError.value = extractApiErrorMessage(error, '创建模拟任务失败')
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  void loadContract()
  void loadKeys()
})
</script>
