<template>
  <BaseDialog
    :show="show"
    title="确认单次 Seedance 授权"
    width="normal"
    :show-close-button="false"
    @close="emit('cancel')"
  >
    <div class="seedance-authorization-dialog space-y-4">
      <p class="seedance-authorization-warning text-sm text-gray-600 dark:text-dark-300">
        此操作只记录一次真实调用授权，不会立即请求上游。执行时仍由后端原子门禁校验预算与授权状态。
      </p>
      <dl class="seedance-authorization-evidence grid gap-3 text-sm sm:grid-cols-2">
        <div class="seedance-authorization-field">
          <dt class="text-gray-500">固定模型</dt>
          <dd class="mt-1 break-all font-medium text-gray-900 dark:text-white">{{ contract.default_model }}</dd>
        </div>
        <div class="seedance-authorization-field">
          <dt class="text-gray-500">固定规格</dt>
          <dd class="mt-1 font-medium text-gray-900 dark:text-white">{{ contract.duration_seconds }} 秒 / {{ contract.resolution }}</dd>
        </div>
        <div class="seedance-authorization-field">
          <dt class="text-gray-500">员工 / 分组</dt>
          <dd class="mt-1 font-medium text-gray-900 dark:text-white">员工：{{ unavailable }}；分组：{{ provider.group_name || unavailable }}</dd>
        </div>
        <div class="seedance-authorization-field">
          <dt class="text-gray-500">授权人</dt>
          <dd class="mt-1 font-medium text-gray-900 dark:text-white">{{ authorizationActor }}</dd>
        </div>
        <div class="seedance-authorization-field">
          <dt class="text-gray-500">预算上限</dt>
          <dd class="mt-1 font-medium text-gray-900 dark:text-white">{{ unavailable }}</dd>
        </div>
        <div class="seedance-authorization-field">
          <dt class="text-gray-500">剩余授权次数</dt>
          <dd class="mt-1 font-medium text-gray-900 dark:text-white">{{ remainingCount }}</dd>
        </div>
        <div class="seedance-authorization-field sm:col-span-2">
          <dt class="text-gray-500">消费后状态</dt>
          <dd class="mt-1 font-medium text-gray-900 dark:text-white">{{ consumptionState }}</dd>
        </div>
      </dl>
    </div>
    <template #footer>
      <div class="seedance-authorization-actions flex w-full flex-col-reverse gap-2 sm:flex-row sm:justify-end">
        <button
          type="button"
          class="btn btn-secondary"
          data-testid="cancel-video-authorization"
          :disabled="submitting"
          @click="emit('cancel')"
        >
          取消
        </button>
        <button
          type="button"
          class="btn btn-primary"
          data-testid="confirm-video-authorization"
          :disabled="submitting"
          @click="emit('confirm')"
        >
          {{ submitting ? '正在记录授权…' : '确认授权一次' }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import type { VideoProviderAccount, VideoProviderContract } from '@/api/admin/video'

const props = defineProps<{
  show: boolean
  provider: VideoProviderAccount
  contract: VideoProviderContract
  submitting: boolean
}>()

const emit = defineEmits<{
  (event: 'cancel'): void
  (event: 'confirm'): void
}>()

const unavailable = '不可用（后端未提供）'
const remainingCount = computed(() => {
  if (props.provider.tiny_real_consumed_at) return '0（已消费）'
  if (props.provider.tiny_real_authorized_at) return '1（待消费）'
  return '当前 0；本次授权成功后为 1'
})
const authorizationActor = computed(() => props.provider.tiny_real_authorized_by ? `管理员 #${props.provider.tiny_real_authorized_by}` : unavailable)
const consumptionState = computed(() => {
  if (props.provider.tiny_real_consumed_at) return `已消费（${props.provider.tiny_real_consumed_at}）`
  if (props.provider.tiny_real_authorized_at) return '待消费；成功消费后剩余次数变为 0'
  return '尚未授权；本次确认后进入“待消费”'
})
</script>
