<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 border-b border-gray-200 pb-4 dark:border-dark-700 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">创建视频任务</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">提交文生视频、图生视频或参考视频任务，创建成功后自动进入任务详情。</p>
        </div>
        <RouterLink class="btn btn-outline" to="/admin/video/tasks">
          <Icon name="document" size="sm" />
          任务列表
        </RouterLink>
      </div>

      <section class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
        <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div>
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">演示模板</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">选择一个业务场景，系统会自动填入提示词、任务类型、画幅、时长和分辨率。</p>
          </div>
          <div class="flex flex-wrap gap-2">
            <button v-for="template in demoTemplates" :key="template.name" class="btn btn-sm btn-outline" type="button" @click="applyTemplate(template)">
              {{ template.name }}
            </button>
          </div>
        </div>
      </section>

      <div class="grid gap-6 xl:grid-cols-[0.95fr_1.05fr]">
        <section class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
          <form class="space-y-4" @submit.prevent="submitTask">
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">供应商通道</label>
              <select v-model.number="form.provider_account_id" class="input" required @change="syncProviderModel">
                <option :value="0" disabled>请选择通道</option>
                <option v-for="provider in enabledProviders" :key="provider.id" :value="provider.id">
                  {{ providerDisplayName(provider) }} | {{ modelDisplayName(provider.provider, provider.default_model) }}
                </option>
              </select>
            </div>

            <div class="grid gap-4 sm:grid-cols-2">
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">任务类型</label>
                <select v-model="form.task_type" class="input">
                  <option v-for="item in taskTypeOptions" :key="item.value" :value="item.value">{{ item.label }}</option>
                </select>
              </div>
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">模型</label>
                <input v-model="form.model" class="input" maxlength="200" />
              </div>
            </div>

            <div>
              <div class="mb-1 flex items-center justify-between gap-3">
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">提示词</label>
                <div class="flex flex-wrap gap-3">
                  <button class="text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-300" type="button" @click="applyMockSuccess">
                    填入成功演示
                  </button>
                  <button class="text-xs font-medium text-red-600 hover:text-red-700 dark:text-red-300" type="button" @click="applyMockFailure">
                    填入失败演示
                  </button>
                </div>
              </div>
              <textarea v-model="form.prompt" class="input min-h-36 resize-y" maxlength="8000" required />
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">失败演示会提交一个预设失败场景，用于查看失败原因和重新创建流程。</p>
            </div>

            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">负向提示词</label>
              <textarea v-model="form.negative_prompt" class="input min-h-20 resize-y" maxlength="4000" />
            </div>

            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">参考图 URL</label>
              <input v-model="form.reference_image_url" class="input" maxlength="1000" placeholder="https://..." />
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">仅图生视频或参考视频任务需要填写；请使用 http 或 https 链接。</p>
            </div>

            <div class="grid gap-4 sm:grid-cols-3">
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">画幅比例</label>
                <select v-model="form.aspect_ratio" class="input">
                  <option value="16:9">16:9</option>
                  <option value="9:16">9:16</option>
                  <option value="1:1">1:1</option>
                  <option value="4:3">4:3</option>
                </select>
              </div>
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">时长（秒）</label>
                <input v-model.number="form.duration" class="input" type="number" min="1" max="60" />
              </div>
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">分辨率</label>
                <select v-model="form.resolution" class="input">
                  <option value="720p">720p</option>
                  <option value="1080p">1080p</option>
                  <option value="480p">480p</option>
                </select>
              </div>
            </div>

            <p v-if="validationMessage" class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-700 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-300">
              {{ validationMessage }}
            </p>

            <button class="btn btn-primary" type="submit" :disabled="submitting || !form.provider_account_id || Boolean(validationMessage)">
              <Icon name="play" size="sm" />
              创建任务
            </button>
          </form>
        </section>

        <section class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">通道状态</h2>
          <div class="mt-4 space-y-3">
            <div v-for="provider in providers" :key="provider.id" class="flex items-center justify-between gap-3 rounded-lg border border-gray-200 p-3 dark:border-dark-700">
              <div class="min-w-0">
                <span class="inline-flex rounded-md px-2 py-1 text-xs font-medium" :class="providerBadgeClass(provider.provider)">
                  {{ providerDisplayName(provider) }}
                </span>
                <div class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">{{ modelDisplayName(provider.provider, provider.default_model) }}</div>
              </div>
              <div class="text-right text-xs">
                <div :class="provider.enabled ? 'text-emerald-600 dark:text-emerald-300' : 'text-gray-500 dark:text-gray-400'">
                  {{ providerEnabledLabel(provider.enabled) }}
                </div>
                <div class="mt-1 text-gray-500 dark:text-gray-400">{{ providerKeyLabel(provider.api_key_configured, provider.masked_key) }}</div>
              </div>
            </div>
            <div v-if="!enabledProviders.length" class="rounded-lg border border-dashed border-gray-200 p-4 text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">
              暂无可用通道，请先到模型通道页面启用演示通道。
            </div>
          </div>
          <RouterLink class="btn btn-outline mt-5" to="/admin/video/providers">
            <Icon name="key" size="sm" />
            配置模型通道
          </RouterLink>
        </section>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import { videoTaskAPI, type VideoProviderAccount, type VideoTaskCreatePayload } from '@/api/admin/video'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import {
  loadTaskDraft,
  modelDisplayName,
  providerBadgeClass,
  providerDisplayName,
  providerEnabledLabel,
  providerKeyLabel,
  taskTypeOptions,
} from './videoUtils'

const router = useRouter()
const appStore = useAppStore()
const providers = ref<VideoProviderAccount[]>([])
const submitting = ref(false)

type DemoTemplate = {
  name: string
  task_type: VideoTaskCreatePayload['task_type']
  prompt: string
  negative_prompt: string
  aspect_ratio: string
  duration: number
  resolution: string
}

const demoTemplates: DemoTemplate[] = [
  {
    name: '安全产品短片',
    task_type: 'text_to_video',
    prompt: '生成一段企业安全产品短片：深色控制台界面、模型通道状态、任务流转时间线和结果链接依次出现，画面克制、专业、适合客户演示。',
    negative_prompt: '夸张霓虹、卡通人物、杂乱背景、低清晰度',
    aspect_ratio: '16:9',
    duration: 6,
    resolution: '720p',
  },
  {
    name: '游戏广告素材',
    task_type: 'text_to_video',
    prompt: '生成一段游戏广告素材：未来城市赛道、角色快速穿越镜头、最后出现可投放的短视频构图，节奏紧凑但不杂乱。',
    negative_prompt: '血腥、恐怖、模糊画面、过度闪烁',
    aspect_ratio: '9:16',
    duration: 5,
    resolution: '720p',
  },
  {
    name: '短剧分镜片段',
    task_type: 'text_to_video',
    prompt: '生成一段短剧分镜片段：办公室门口的紧张对话、人物表情清晰、镜头从中景切到特写，适合作为剧情预告素材。',
    negative_prompt: '字幕遮挡、画面抖动、过度夸张表演',
    aspect_ratio: '16:9',
    duration: 5,
    resolution: '720p',
  },
]

const form = reactive<VideoTaskCreatePayload>({
  provider_account_id: 0,
  task_type: 'text_to_video',
  model: '',
  prompt: '生成一段企业 API 管理后台的安全产品演示短片，画面清晰、节奏简洁。',
  negative_prompt: '',
  reference_image_url: '',
  aspect_ratio: '16:9',
  duration: 5,
  resolution: '720p',
})

const enabledProviders = computed(() => providers.value.filter((provider) => provider.enabled))
const validationMessage = computed(() => {
  if (!enabledProviders.value.length) return '没有可用通道：请先前往模型通道页启用演示通道。'
  if (!form.provider_account_id) return '请选择模型通道。'
  if (!form.prompt.trim()) return '提示词不能为空：请描述要生成的视频内容。'
  if ((form.duration || 0) < 1 || (form.duration || 0) > 60) return '时长必须在 1 到 60 秒之间。'
  if (form.reference_image_url && !isValidOptionalUrl(form.reference_image_url)) return '参考图 URL 格式不正确：请使用 http 或 https 链接，或留空。'
  return ''
})

async function loadProviders() {
  try {
    const res = await adminAPI.video.listProviders()
    providers.value = res.items || []
    const mock = enabledProviders.value.find((provider) => provider.provider === 'mock')
    const first = mock || enabledProviders.value[0]
    if (first && !form.provider_account_id) {
      form.provider_account_id = first.id
      form.model = modelDisplayName(first.provider, first.default_model)
    }
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载模型通道失败'))
  }
}

function applyTemplate(template: DemoTemplate) {
  selectPreferredProvider()
  form.task_type = template.task_type
  form.prompt = template.prompt
  form.negative_prompt = template.negative_prompt
  form.reference_image_url = ''
  form.aspect_ratio = template.aspect_ratio
  form.duration = template.duration
  form.resolution = template.resolution
}

function syncProviderModel() {
  const provider = providers.value.find((item) => item.id === form.provider_account_id)
  if (provider?.default_model) {
    form.model = modelDisplayName(provider.provider, provider.default_model)
  }
}

function selectPreferredProvider() {
  const demoProvider = enabledProviders.value.find((provider) => provider.provider === 'mock')
  const first = demoProvider || enabledProviders.value[0]
  if (!first) return
  form.provider_account_id = first.id
  form.model = modelDisplayName(first.provider, first.default_model)
}

function selectMockProvider() {
  selectPreferredProvider()
}

function applyMockSuccess() {
  selectMockProvider()
  form.task_type = 'text_to_video'
  form.prompt = '生成一段企业 API 管理后台的安全产品演示短片，画面清晰、节奏简洁。'
  form.negative_prompt = ''
}

function applyMockFailure() {
  selectMockProvider()
  form.task_type = 'text_to_video'
  form.prompt = '生成一段用于演示失败处理的视频任务：提示词会触发预设失败场景，方便查看失败原因和重新创建流程。[fail]'
}

async function submitTask() {
  if (validationMessage.value) {
    appStore.showWarning(validationMessage.value)
    return
  }
  submitting.value = true
  try {
    const task = await videoTaskAPI.create({
      ...form,
      prompt: form.prompt.trim(),
      negative_prompt: form.negative_prompt?.trim(),
      reference_image_url: form.reference_image_url?.trim(),
      model: form.model?.trim(),
    })
    appStore.showSuccess('任务已进入队列，可在详情查看处理进度。')
    router.push(`/admin/video/tasks/${task.id}`)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '创建视频任务失败'))
  } finally {
    submitting.value = false
  }
}

function isValidOptionalUrl(value: string): boolean {
  const trimmed = value.trim()
  if (!trimmed) return true
  try {
    const url = new URL(trimmed)
    return url.protocol === 'http:' || url.protocol === 'https:'
  } catch {
    return false
  }
}

function applyDraftIfPresent() {
  const draft = loadTaskDraft()
  if (!draft) return
  Object.assign(form, draft)
  if (!enabledProviders.value.some((provider) => provider.id === form.provider_account_id)) {
    selectPreferredProvider()
  } else {
    syncProviderModel()
  }
  appStore.showInfo('已复制参数，可调整后重新创建任务。')
}

onMounted(async () => {
  await loadProviders()
  applyDraftIfPresent()
})
</script>
