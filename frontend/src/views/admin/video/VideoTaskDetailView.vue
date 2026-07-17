<template>
  <AppLayout>
    <div class="video-task-detail min-w-0 space-y-5 overflow-x-clip">
      <RouterLink to="/admin/video/tasks" class="video-task-back inline-flex text-sm text-primary-600">返回任务证据</RouterLink>
      <header class="video-task-header flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h2 class="video-task-title ui-heading break-words">视频任务证据 #{{ task?.id }}</h2>
          <p class="video-task-description ui-subheading mt-1">只展示管理员接口返回的事实；缺少的审计字段明确标记，不用当前配置推测历史请求。</p>
        </div>
        <TaskProgressRing v-if="task" :phase="taskPhaseLabel(task.status)" :size="40" side-label />
      </header>

      <p v-if="loading" class="video-task-loading text-sm text-gray-500" role="status">正在加载任务证据…</p>
      <p v-else-if="errorMessage" class="video-task-error text-sm text-red-600" role="alert">{{ errorMessage }}</p>

      <template v-else-if="task">
        <section class="video-task-overview ui-panel p-5" aria-labelledby="video-task-overview-title">
          <h2 id="video-task-overview-title" class="text-base font-semibold">单任务事实</h2>
          <dl class="mt-4 grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
            <div v-for="item in overview" :key="item.label" class="video-task-field min-w-0">
              <dt class="text-xs text-gray-500">{{ item.label }}</dt>
              <dd class="mt-1 break-all text-sm text-gray-900 dark:text-white">{{ item.value }}</dd>
            </div>
          </dl>
        </section>

        <section class="video-task-specs ui-panel p-5" aria-labelledby="video-task-specs-title">
          <div class="video-task-specs-heading flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
            <div class="video-task-specs-copy">
              <h2 id="video-task-specs-title" class="text-base font-semibold">三方规格核对</h2>
              <p class="mt-1 text-sm text-gray-500">请求规格来自任务持久化字段；上游和计费规格只展示各自的独立原始事实，缺失时禁止复用请求值。</p>
            </div>
            <span class="video-spec-comparison text-sm font-medium" :class="specComparison.className">{{ specComparison.label }}</span>
          </div>
          <div class="mt-4 grid gap-4 lg:grid-cols-3">
            <article v-for="spec in specifications" :key="spec.title" class="video-task-spec-block min-w-0 bg-gray-50 p-4 dark:bg-dark-800">
              <h3 class="text-sm font-semibold">{{ spec.title }}</h3>
              <dl class="mt-3 space-y-3 text-sm">
                <div v-for="field in spec.fields" :key="field.label" class="video-task-spec-field">
                  <dt class="text-xs text-gray-500">{{ field.label }}</dt>
                  <dd class="mt-1 break-all" :class="fieldClass(field)">{{ field.value }}</dd>
                </div>
              </dl>
            </article>
          </div>
        </section>

        <section class="video-task-assets ui-panel p-5" aria-labelledby="video-task-assets-title">
          <h2 id="video-task-assets-title" class="text-base font-semibold">资产预览</h2>
          <div class="mt-4 max-w-xl">
            <label for="qcanvas-base-url" class="mb-1 block text-sm font-medium">QCanvas 本机地址</label>
            <input
              id="qcanvas-base-url"
              v-model.trim="qcanvasBaseURL"
              type="url"
              class="input w-full"
              placeholder="例如 http://127.0.0.1:5173"
              autocomplete="url"
              @change="persistQCanvasBaseURL"
            />
            <p class="mt-1 text-xs text-gray-500">必须显式填写 loopback origin；仅保存在当前浏览器，不是凭据。地址无效时不会签发票据。</p>
          </div>
          <div class="mt-4 grid gap-5 lg:grid-cols-2">
            <div class="video-task-result min-w-0">
              <h3 class="text-sm font-medium">结果视频</h3>
              <video v-if="task.result_url" class="mt-2 aspect-video w-full bg-black object-contain" controls preload="metadata">
                <source :src="task.result_url" />
                浏览器无法播放此视频，请使用下方链接打开。
              </video>
              <p v-else class="mt-2 text-sm text-gray-500">未返回结果视频。</p>
              <a v-if="task.result_url" :href="task.result_url" target="_blank" rel="noreferrer" class="mt-2 block break-all text-sm text-primary-600">在新窗口打开结果资产</a>
              <button
                v-if="task.status === 'succeeded' && task.result_url"
                type="button"
                class="btn btn-secondary mt-3"
                :disabled="handoffLoading !== null"
                @click="startAssetHandoff('video')"
              >
                {{ handoffLoading === 'video' ? '正在创建交接票据…' : '发送视频到 QCanvas' }}
              </button>
            </div>
            <div class="video-task-tail-frame min-w-0">
              <h3 class="text-sm font-medium">尾帧</h3>
              <img v-if="task.last_frame_url" :src="task.last_frame_url" alt="视频任务尾帧" class="mt-2 aspect-video w-full bg-black object-contain" />
              <p v-else class="mt-2 text-sm text-gray-500">未返回尾帧。</p>
              <a v-if="task.last_frame_url" :href="task.last_frame_url" target="_blank" rel="noreferrer" class="mt-2 block break-all text-sm text-primary-600">在新窗口打开尾帧</a>
              <p v-if="task.last_frame_url" class="mt-2 text-xs text-gray-500">当前历史尾帧是 JPEG/JFIF，不开放 PNG 交接按钮，也不会伪报 MIME。既有 Gemini PNG 由 tapcanvas-api 本地文件导入。</p>
            </div>
          </div>
          <p v-if="handoffError" class="mt-3 text-sm text-red-600" role="alert">{{ handoffError }}</p>
        </section>

        <section v-if="failureReason" class="video-task-failure card p-5" aria-labelledby="video-task-failure-title">
          <h2 id="video-task-failure-title" class="text-base font-semibold">失败证据</h2>
          <p class="mt-2 break-words text-sm text-red-600">{{ failureReason }}</p>
          <p v-if="task.provider_error_code" class="mt-2 break-all font-mono text-xs text-gray-500">原始错误码：{{ task.provider_error_code }}</p>
        </section>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import TaskProgressRing from '@/components/common/TaskProgressRing.vue'
import { adminAPI } from '@/api/admin'
import { buildQCanvasAssetHandoffTargetURL, startQCanvasAssetHandoffTransfer } from '@/api/admin/video'
import type { AssetHandoffKind, VideoTaskAdmin } from '@/api/admin/video'
import { extractApiErrorMessage } from '@/utils/apiError'

interface EvidenceField {
  label: string
  value: string
  available: boolean
  comparison?: SpecComparisonState
}

type SpecComparisonState = 'match' | 'mismatch' | 'incomplete' | 'none'
type SpecDimension = 'model' | 'duration' | 'resolution'

const unavailable = '不可用（后端未提供）'
const route = useRoute()
const task = ref<VideoTaskAdmin>()
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

const loading = ref(true)
const errorMessage = ref('')
const handoffError = ref('')
const handoffLoading = ref<AssetHandoffKind | null>(null)
const qcanvasBaseURL = ref(
  window.localStorage.getItem('wujie-qcanvas-base-url') || import.meta.env.VITE_QCANVAS_BASE_URL || ''
)

const persistQCanvasBaseURL = (): void => {
  const value = qcanvasBaseURL.value.trim()
  if (value) window.localStorage.setItem('wujie-qcanvas-base-url', value)
  else window.localStorage.removeItem('wujie-qcanvas-base-url')
}

const startAssetHandoff = async (kind: AssetHandoffKind): Promise<void> => {
  if (!task.value || handoffLoading.value !== null) return
  handoffError.value = ''
  try {
    buildQCanvasAssetHandoffTargetURL(qcanvasBaseURL.value)
    persistQCanvasBaseURL()
  } catch (error) {
    handoffError.value = extractApiErrorMessage(error, '请先填写有效的 QCanvas 本机地址')
    return
  }
  const targetWindow = window.open('', '_blank')
  if (!targetWindow) {
    handoffError.value = '浏览器阻止了新窗口，未创建交接票据。'
    return
  }
  handoffLoading.value = kind
  try {
    const issued = await adminAPI.video.createAssetHandoff(task.value.id, kind)
    startQCanvasAssetHandoffTransfer(targetWindow, issued.ticket, qcanvasBaseURL.value)
  } catch (error) {
    targetWindow.close()
    handoffError.value = extractApiErrorMessage(error, '创建 QCanvas 交接票据失败')
  } finally {
    handoffLoading.value = null
  }
}

const balanceEvidence = computed(() => {
  if (!task.value || task.value.balance_before_usd == null || task.value.balance_after_usd == null || task.value.balance_delta_usd == null) {
    return unavailable
  }
  return `${task.value.balance_before_usd} → ${task.value.balance_after_usd}（${task.value.balance_delta_usd} USD）`
})

const authorizationEvidence = computed(() => {
  if (!task.value?.authorization_consumed_at) return unavailable
  const actor = task.value.authorization_consumed_by == null ? '授权人不可用' : `授权人 #${task.value.authorization_consumed_by}`
  return `已消费（${actor}，${task.value.authorization_consumed_at}）`
})

const overview = computed(() => {
  if (!task.value) return []
  return [
    { label: '请求人', value: `员工 #${task.value.created_by}（姓名后端未提供）` },
    { label: '分组 ID', value: task.value.group_id > 0 ? String(task.value.group_id) : unavailable },
    { label: '员工 Key ID', value: task.value.api_key_id > 0 ? String(task.value.api_key_id) : unavailable },
    { label: '模型', value: task.value.request_model || unavailable },
    { label: '终态', value: task.value.status || unavailable },
    { label: '本地任务 ID', value: String(task.value.id) },
    { label: '上游任务 ID', value: task.value.upstream_task_id || '未返回' },
    { label: '调度状态', value: task.value.dispatch_state || unavailable },
    { label: '真实调用次数', value: String(task.value.real_dispatch_count) },
    { label: 'Tokens', value: task.value.usage_total_tokens == null ? unavailable : String(task.value.usage_total_tokens) },
    { label: '预留预算上限', value: `${task.value.reserved_cost_usd} USD` },
    { label: '预留结算状态', value: task.value.reservation_state || unavailable },
    { label: '预留时间', value: task.value.reserved_at || unavailable },
    { label: '结算费用', value: `${task.value.cost_amount} ${task.value.currency || 'USD'}` },
    { label: '上游实际费用', value: `${task.value.provider_actual_cost_usd} USD` },
    { label: '余额差分', value: balanceEvidence.value },
    { label: '授权消费状态', value: authorizationEvidence.value },
    { label: '创建时间', value: task.value.created_at || unavailable },
    { label: '完成时间', value: task.value.completed_at || '尚未完成' }
  ]
})

const dimensionComparison = computed<Record<SpecDimension, SpecComparisonState>>(() => {
  if (!task.value) return { model: 'incomplete', duration: 'incomplete', resolution: 'incomplete' }
  const compare = (values: readonly (string | number | null | undefined)[]): SpecComparisonState => {
    if (values.some((value) => value == null || value === '')) return 'incomplete'
    return values.every((value) => value === values[0]) ? 'match' : 'mismatch'
  }
  return {
    model: compare([task.value.request_model, task.value.upstream_model, task.value.billing_model]),
    duration: compare([task.value.request_duration_seconds, task.value.upstream_duration_seconds, task.value.billing_duration_seconds]),
    resolution: compare([task.value.request_resolution, task.value.upstream_resolution, task.value.billing_resolution]),
  }
})

const specComparison = computed(() => {
  const states = Object.values(dimensionComparison.value)
  if (states.includes('mismatch')) {
    return { label: '规格不一致', className: 'video-spec-mismatch text-red-700 dark:text-red-300' }
  }
  if (states.includes('incomplete')) {
    return { label: '规格证据不完整，无法判定一致性', className: 'video-spec-incomplete text-amber-700 dark:text-amber-300' }
  }
  return { label: '规格一致', className: 'video-spec-match text-emerald-700 dark:text-emerald-300' }
})

const fieldClass = (field: EvidenceField): string => {
  if (!field.available || field.comparison === 'incomplete') return 'video-spec-value--incomplete text-amber-700 dark:text-amber-300'
  if (field.comparison === 'mismatch') return 'video-spec-value--mismatch text-red-700 dark:text-red-300'
  if (field.comparison === 'match') return 'video-spec-value--match text-emerald-700 dark:text-emerald-300'
  return ''
}

const specifications = computed(() => {
  if (!task.value) return []
  const field = (label: string, value: string | number | null, dimension?: SpecDimension, suffix = ''): EvidenceField => ({
    label,
    value: value == null || value === '' ? unavailable : `${value}${suffix}`,
    available: value != null && value !== '',
    comparison: dimension ? dimensionComparison.value[dimension] : 'none',
  })
  return [
    {
      title: '请求规格',
      fields: [
        field('原始模型字段', task.value.request_model, 'model'),
        field('原始时长字段', task.value.request_duration_seconds, 'duration', ' 秒'),
        field('原始分辨率字段', task.value.request_resolution, 'resolution')
      ]
    },
    {
      title: '上游回传规格',
      fields: [
        { label: '原始上游任务 ID', value: task.value.upstream_task_id || '未返回', available: Boolean(task.value.upstream_task_id) },
        field('原始模型字段', task.value.upstream_model, 'model'),
        field('原始时长字段', task.value.upstream_duration_seconds, 'duration', ' 秒'),
        field('原始分辨率字段', task.value.upstream_resolution, 'resolution')
      ]
    },
    {
      title: '计费规格',
      fields: [
        { label: '原始预留金额字段', value: `${task.value.reserved_cost_usd} USD`, available: true },
        { label: '原始预留状态字段', value: task.value.reservation_state || unavailable, available: Boolean(task.value.reservation_state) },
        { label: '原始结算费用字段', value: `${task.value.cost_amount} ${task.value.currency || 'USD'}`, available: true },
        { label: '原始上游费用字段', value: `${task.value.provider_actual_cost_usd} USD`, available: true },
        { label: '原始计费币种字段', value: task.value.currency || unavailable, available: Boolean(task.value.currency) },
        field('原始计费模型字段', task.value.billing_model, 'model'),
        field('原始计费时长字段', task.value.billing_duration_seconds, 'duration', ' 秒'),
        field('原始计费分辨率字段', task.value.billing_resolution, 'resolution')
      ]
    }
  ]
})

const failureReason = computed(() => task.value?.error_message || task.value?.provider_error_message || '')

onMounted(async () => {
  try {
    task.value = await adminAPI.video.getTask(Number(route.params.id))
  } catch (error) {
    errorMessage.value = extractApiErrorMessage(error, '加载任务证据失败')
  } finally {
    loading.value = false
  }
})
</script>
