<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 border-b border-ui-border pb-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 class="ui-heading">{{ isVideoGatewayDemoMode ? '试跑任务' : '创建视频任务' }}</h1>
          <p class="ui-subheading mt-1">
            {{ isVideoGatewayDemoMode ? '先用一条演示任务检查系统是否能正常接收、处理和记录任务，不会调用真实生成服务。' : '提交文生视频、图生视频或参考视频任务，创建成功后自动进入任务详情。' }}
          </p>
        </div>
        <RouterLink class="btn btn-outline" to="/admin/video/tasks">
          <Icon name="document" size="sm" />
          {{ isVideoGatewayDemoMode ? '任务记录' : '任务列表' }}
        </RouterLink>
      </div>

      <section class="ui-panel p-5" aria-label="提示词模板">
        <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div>
            <h2 class="text-base font-semibold text-ui-text">{{ isVideoGatewayDemoMode ? '业务提示词模板' : 'Prompt 模板候选' }}</h2>
            <p class="ui-subheading mt-1">
              {{ isVideoGatewayDemoMode ? '选择 AI 中剧 / 短剧生产模板，快速填入试跑任务参数。' : '选择业务候选，系统会自动填入提示词、任务类型、画幅、时长和分辨率。' }}
            </p>
          </div>
        </div>
        <div class="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-5">
          <button
            v-for="candidate in promptAssetCandidates"
            :key="candidate.id"
            class="rounded-xl border border-ui-border bg-ui-canvas/60 p-3 text-left transition hover:border-primary-300 hover:bg-primary-50 dark:hover:border-primary-500/40 dark:hover:bg-primary-500/10"
            type="button"
            @click="applyTemplate(candidate)"
          >
            <span class="inline-flex rounded-md bg-ui-surface px-2 py-1 text-xs font-medium text-ui-text-muted">{{ candidate.category }}</span>
            <span class="mt-3 block text-sm font-semibold text-ui-text">{{ candidate.name }}</span>
            <span class="candidate-prompt ui-subheading mt-2 block text-xs leading-5">{{ promptDisplayText(candidate.prompt) }}</span>
            <span class="mt-3 block text-xs font-medium text-ui-text-muted">{{ candidate.aspect_ratio }} · {{ candidate.duration }}s · {{ candidate.resolution }}</span>
          </button>
        </div>
      </section>

      <div class="grid gap-6" :class="{ 'xl:grid-cols-[1fr_0.78fr]': !isVideoGatewayDemoMode }">
        <!-- 主流程:执行模式与提示词直接可见,高级参数收进折叠区 -->
        <section class="ui-panel p-5" aria-label="任务参数">
          <form class="space-y-4" @submit.prevent="submitTask">
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300" for="video-execution-mode">执行模式</label>
              <select id="video-execution-mode" v-model="executionMode" class="input" data-testid="video-execution-mode">
                <option value="mock">免费试跑</option>
                <option value="review_real" :disabled="!reviewRealAvailable">一次真实复核</option>
                <option value="internal_real" :disabled="!internalRealAvailable">内部真实调用</option>
              </select>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">默认免费试跑；真实模式不会因为“偏好 mock”而误路由到真实账号。</p>
            </div>

            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">任务类型</label>
              <select v-model="form.task_type" class="input">
                <option v-for="item in taskTypeOptions" :key="item.value" :value="item.value">{{ item.label }}</option>
              </select>
            </div>

            <div>
              <div class="mb-1 flex items-center justify-between gap-3">
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">提示词</label>
                <div class="flex flex-wrap gap-3">
                  <button class="text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-300" type="button" @click="applyMockSuccess">
                    {{ isVideoGatewayDemoMode ? '填入成功任务示例' : '填入成功演示' }}
                  </button>
                  <button class="text-xs font-medium text-red-600 hover:text-red-700 dark:text-red-300" type="button" @click="applyMockFailure">
                    {{ isVideoGatewayDemoMode ? '填入失败任务示例' : '填入失败演示' }}
                  </button>
                </div>
              </div>
              <textarea v-model="form.prompt" class="input min-h-36 resize-y" maxlength="8000" required />
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ isVideoGatewayDemoMode ? '失败任务示例会提交预设失败场景，用于查看失败原因和重新发起流程。' : '失败演示会提交一个预设失败场景，用于查看失败原因和重新创建流程。' }}
              </p>
            </div>

            <details class="rounded-xl border border-ui-border bg-ui-canvas/60 p-4">
              <summary class="cursor-pointer text-sm font-semibold text-ui-text">高级参数</summary>
              <div class="mt-4 space-y-4">
                <div>
                  <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">模型偏好（可选）</label>
                  <input v-model="form.model" class="input" maxlength="200" />
                  <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">默认由网关按可用账号选择；普通员工可以保持不改。</p>
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
              </div>
            </details>

            <p v-if="validationMessage" class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-700 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-300">
              {{ validationMessage }}
            </p>

            <button class="btn btn-primary" type="submit" data-testid="video-primary-action" :disabled="submitting || Boolean(validationMessage)">
              <Icon name="play" size="sm" />
              {{ submitting ? '提交中…' : (executionMode === 'mock' ? (isVideoGatewayDemoMode ? '试跑一条任务' : '免费试跑') : '创建真实任务') }}
            </button>
            <p v-if="executionMode === 'mock'" class="text-xs text-gray-500 dark:text-gray-400">这次只检查系统接收、处理和记录能力，不会调用真实生成服务。</p>
          </form>

          <!-- 真实调用确认:醒目阻断式确认,不可错过 -->
          <div
            v-if="confirmRealOpen"
            class="fixed inset-0 z-40 flex items-center justify-center bg-black/50 p-4"
            role="alertdialog"
            aria-modal="true"
            aria-labelledby="video-real-confirm-title"
          >
            <div class="w-full max-w-md rounded-2xl border-2 border-amber-400 bg-white p-5 shadow-2xl dark:border-amber-400/70 dark:bg-dark-800">
              <div class="flex items-start gap-3">
                <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-amber-100 text-amber-600 dark:bg-amber-500/20 dark:text-amber-300">
                  <Icon name="exclamationTriangle" size="md" />
                </span>
                <div>
                  <h3 id="video-real-confirm-title" class="text-lg font-semibold text-gray-900 dark:text-white">确认真实调用</h3>
                  <p class="mt-1 text-sm font-medium text-amber-700 dark:text-amber-300">这不是免费试跑：提交后会真实计费并占用真实额度。</p>
                </div>
              </div>
              <ul class="mt-4 space-y-1 rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-gray-700 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-gray-200">
                <li>模型：{{ form.model || '默认' }}</li>
                <li>时长：{{ form.duration }} 秒</li>
                <li>分辨率：{{ form.resolution }}</li>
                <li>画幅：{{ form.aspect_ratio }}</li>
                <li>预估上限：{{ estimatedMaxCostLabel() }}</li>
                <li>次数影响：计入真实复核/内部策略额度</li>
              </ul>
              <div class="mt-5 flex justify-end gap-3">
                <button class="btn btn-outline" type="button" @click="confirmRealOpen = false">取消</button>
                <button class="btn btn-primary" type="button" :disabled="submitting" @click="confirmRealOnce">再次确认并提交</button>
              </div>
            </div>
          </div>
        </section>

        <div v-if="!isVideoGatewayDemoMode" class="space-y-6">
          <section class="ui-panel p-5" aria-label="通道状态">
            <h2 class="text-base font-semibold text-ui-text">通道状态</h2>
            <div class="mt-4 space-y-3">
              <div v-for="provider in providers" :key="provider.id" class="flex items-center justify-between gap-3 rounded-xl border border-ui-border p-3">
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
                  <div class="mt-1 text-gray-500 dark:text-gray-400">{{ providerKeyLabel(provider.api_key_configured, provider.masked_key, provider.key_status, provider.provider) }}</div>
                  <div v-if="provider.route_skip_reason" class="mt-1 text-red-600 dark:text-red-300">{{ humanIssueLabel(provider.route_skip_reason) }}</div>
                </div>
              </div>
              <div v-if="!enabledProviders.length" class="rounded-lg border border-dashed border-ui-border p-4 text-sm text-ui-text-muted">
                暂无可用通道，请先到模型通道页面启用演示通道。
              </div>
            </div>
            <RouterLink v-if="authStore.isAdmin" class="btn btn-outline mt-5" to="/admin/video/providers">
              <Icon name="key" size="sm" />
              配置模型通道
            </RouterLink>
          </section>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import {
  videoTaskAPI,
  type VideoExecutionCapabilities,
  type VideoExecutionMode,
  type VideoProviderAccount,
  type VideoTaskCreatePayload,
} from '@/api/admin/video'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { extractApiErrorMessage } from '@/utils/apiError'
import { isVideoGatewayDemoMode } from '@/utils/productMode'
import {
  candidateToTaskPayload,
  humanIssueLabel,
  loadTaskDraft,
  modelDisplayName,
  promptAssetCandidates,
  promptDisplayText,
  providerBadgeClass,
  providerDisplayName,
  providerEnabledLabel,
  providerKeyLabel,
  taskTypeOptions,
  type PromptAssetCandidate,
} from './videoUtils'

const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()
const providers = ref<VideoProviderAccount[]>([])
const capabilities = ref<VideoExecutionCapabilities>({ mock: true })
const executionMode = ref<VideoExecutionMode>('mock')
const submitting = ref(false)
const pendingIdempotencyKey = ref('')
const confirmRealOpen = ref(false)
const confirmRealSecond = ref(false)

const form = reactive<VideoTaskCreatePayload>({
  execution_mode: 'mock',
  task_type: 'text_to_video',
  model: '',
  prompt: isVideoGatewayDemoMode
    ? '真人短剧第 1 集情绪爆发镜头：女主在雨夜街边听到男主离开的消息，缓慢抬头，眼神从震惊转为克制的愤怒，镜头缓慢推进。'
    : '生成一段企业 API 管理后台的安全产品演示短片，画面清晰、节奏简洁。',
  negative_prompt: '',
  reference_image_url: '',
  aspect_ratio: '16:9',
  duration: 5,
  resolution: '720p',
})

const enabledProviders = computed(() => providers.value.filter((provider) => provider.route_available))
const reviewRealAvailable = computed(() => Boolean(capabilities.value.review_real))
const internalRealAvailable = computed(() => Boolean(capabilities.value.internal_real) && authStore.isAdmin)
const validationMessage = computed(() => {
  const mockOk = capabilities.value.mock !== false || enabledProviders.value.some((p) => p.provider === 'mock')
  if (executionMode.value === 'mock' && !mockOk && !enabledProviders.value.length) {
    return isVideoGatewayDemoMode ? '没有可用生成通道：请先前往生成通道页启用演示通道。' : '没有可用通道：请先前往模型通道页启用演示通道。'
  }
  if (executionMode.value === 'review_real' && !reviewRealAvailable.value) {
    return '一次真实复核暂不可用：请联系管理员开启复核会话并完成凭证引导。'
  }
  if (executionMode.value === 'internal_real' && !internalRealAvailable.value) {
    return '内部真实通道未开通：请先完成验收并由管理员配置正式内部通道策略。'
  }
  if (!form.prompt.trim()) return '提示词不能为空：请描述要生成的视频内容。'
  if ((form.duration || 0) < 1 || (form.duration || 0) > 60) return '时长必须在 1 到 60 秒之间。'
  if (form.reference_image_url && !isValidOptionalUrl(form.reference_image_url)) return '参考图 URL 格式不正确：请使用 http 或 https 链接，或留空。'
  return ''
})

async function loadProviders() {
  try {
    const res = await videoTaskAPI.listProviders()
    providers.value = res.items || []
    capabilities.value = res.execution_capabilities || {
      mock: true,
      review_real: false,
      internal_real: false,
    }
    if (!capabilities.value.mock && enabledProviders.value.some((p) => p.provider === 'mock')) {
      capabilities.value.mock = true
    }
    executionMode.value = 'mock'
    form.execution_mode = 'mock'
    const first = preferredProvider()
    if (first && !form.model) {
      form.model = defaultModelValue(first)
    }
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, isVideoGatewayDemoMode ? '加载生成通道失败' : '加载模型通道失败'))
  }
}

function applyTemplate(template: PromptAssetCandidate) {
  selectPreferredProvider()
  Object.assign(form, candidateToTaskPayload(template))
  syncProviderModel()
}

function syncProviderModel() {
  const provider = preferredProvider()
  if (provider?.default_model) {
    form.model = defaultModelValue(provider)
  }
}

function preferredProvider() {
  return enabledProviders.value.find((provider) => provider.provider === 'mock') || enabledProviders.value[0]
}

function selectPreferredProvider() {
  const first = preferredProvider()
  executionMode.value = 'mock'
  form.execution_mode = 'mock'
  delete form.provider_account_id
  if (!first) return
  form.model = defaultModelValue(first)
}

function estimatedMaxCostLabel() {
  const seconds = form.duration || 5
  return `约 ¥${(seconds * 1).toFixed(2)}（上限示意，以结算为准）`
}

function selectMockProvider() {
  selectPreferredProvider()
}

function applyMockSuccess() {
  selectMockProvider()
  form.task_type = 'text_to_video'
  form.prompt = isVideoGatewayDemoMode
    ? 'AI 短剧结尾钩子：主角推开旧房门，发现桌上有第二封信和一张陌生合影，镜头从信封推进到主角震惊表情。'
    : '生成一段企业 API 管理后台的安全产品演示短片，画面清晰、节奏简洁。'
  form.negative_prompt = ''
}

function applyMockFailure() {
  selectMockProvider()
  form.task_type = 'text_to_video'
  form.prompt = isVideoGatewayDemoMode
    ? '生成一段用于演示 AI 短剧任务失败审计的镜头：提示词会触发预设失败场景，方便查看失败原因、调度记录和经验事件。[fail]'
    : '生成一段用于演示失败处理的视频任务：提示词会触发预设失败场景，方便查看失败原因和重新创建流程。[fail]'
}

async function submitTask() {
  if (submitting.value) return
  if (validationMessage.value) {
    appStore.showWarning(validationMessage.value)
    return
  }
  form.execution_mode = executionMode.value
  if (executionMode.value !== 'mock' && !confirmRealSecond.value) {
    confirmRealOpen.value = true
    return
  }
  if (!pendingIdempotencyKey.value) {
    pendingIdempotencyKey.value = crypto.randomUUID()
  }
  submitting.value = true
  try {
    const payload: VideoTaskCreatePayload = {
      ...form,
      execution_mode: executionMode.value,
      prompt: form.prompt.trim(),
      negative_prompt: form.negative_prompt?.trim(),
      reference_image_url: form.reference_image_url?.trim(),
      model: form.model?.trim(),
    }
    delete payload.provider_account_id
    if (!payload.model) {
      delete payload.model
    }
    const task = await videoTaskAPI.create(payload, pendingIdempotencyKey.value)
    pendingIdempotencyKey.value = ''
    confirmRealOpen.value = false
    confirmRealSecond.value = false
    appStore.showSuccess(executionMode.value === 'mock'
      ? (isVideoGatewayDemoMode ? '试跑任务已提交，系统会检查接收、处理和记录流程。' : '免费试跑已进入队列。')
      : '真实任务已进入队列，可在详情查看处理进度。')
    router.push(`/admin/video/tasks/${task.id}`)
  } catch (err) {
    const message = extractApiErrorMessage(err, isVideoGatewayDemoMode ? '试跑任务提交失败' : '创建视频任务失败')
    appStore.showError(humanizeCreateError(message))
  } finally {
    submitting.value = false
  }
}

function humanizeCreateError(message: string): string {
  const lower = message.toLowerCase()
  if (lower.includes('balance') || message.includes('余额')) {
    return `${message}。下一步：检查账户余额或改用免费试跑。`
  }
  if (lower.includes('rate') || message.includes('限流')) {
    return `${message}。下一步：稍后重试或降低并发。`
  }
  if (lower.includes('budget') || message.includes('预算')) {
    return `${message}。下一步：联系管理员调整额度或改用免费试跑。`
  }
  if (lower.includes('session') || message.includes('复核')) {
    return `${message}。下一步：联系管理员开启复核会话，或改用免费试跑。`
  }
  if (lower.includes('timeout') || message.includes('超时')) {
    return `${message}。下一步：稍后重试。`
  }
  return message
}

function confirmRealOnce() {
  confirmRealSecond.value = true
  confirmRealOpen.value = false
  void submitTask()
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

function defaultModelValue(provider: VideoProviderAccount): string {
  return provider.default_model?.trim() || ''
}

function applyDraftIfPresent() {
  const draft = loadTaskDraft()
  if (!draft) return
  Object.assign(form, draft)
  selectPreferredProvider()
  syncProviderModel()
  appStore.showInfo(isVideoGatewayDemoMode ? '已复制参数，可调整后重新发起调用。' : '已复制参数，可调整后重新创建任务。')
}

onMounted(async () => {
  await loadProviders()
  applyDraftIfPresent()
})
</script>

<style scoped>
.candidate-prompt {
  display: -webkit-box;
  min-height: 3.75rem;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
}
</style>
