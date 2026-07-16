<template>
  <BaseDialog :show="show" title="一次性员工凭据" width="narrow" @close="requestClose">
    <div class="space-y-4" role="status" aria-live="polite">
      <p class="text-sm text-gray-600 dark:text-gray-300">
        {{ email }} 的临时密码只显示这一次，24 小时后失效。员工首次登录后必须立即改密。
      </p>
      <div class="rounded-lg bg-gray-100 p-3 dark:bg-dark-700">
        <label class="input-label" for="initial-temporary-password">临时密码</label>
        <input
          id="initial-temporary-password"
          :value="visibleCredential?.temporary_password ?? ''"
          type="password"
          readonly
          class="input font-mono"
          autocomplete="new-password"
        />
        <p class="mt-2 text-xs text-gray-500">
          失效时间：{{ visibleCredential?.expires_at ?? '-' }}
        </p>
      </div>
      <p v-if="!acknowledged" class="text-xs text-amber-700 dark:text-amber-300">
        请先复制或下载凭据，确认留存后才能关闭。
      </p>
    </div>
    <template #footer>
      <div class="flex flex-wrap justify-end gap-3">
        <button data-test="credential-copy" type="button" class="btn btn-secondary" @click="copyCredential">
          复制凭据
        </button>
        <button data-test="credential-download" type="button" class="btn btn-secondary" @click="downloadCredential">
          下载凭据
        </button>
        <button
          data-test="credential-close"
          type="button"
          class="btn btn-primary"
          :disabled="!acknowledged"
          @click="requestClose"
        >
          已安全保存
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import type { InitialCredential } from '@/api/admin/users'

const props = defineProps<{
  show: boolean
  email: string
  credential: InitialCredential | null
}>()

const emit = defineEmits<{ (event: 'close'): void }>()
const visibleCredential = ref<InitialCredential | null>(null)
const acknowledged = ref(false)

watch(
  () => [props.show, props.credential] as const,
  ([show, credential]) => {
    visibleCredential.value = show && credential ? { ...credential } : null
    acknowledged.value = false
  },
  { immediate: true }
)

const credentialText = (): string => {
  if (!visibleCredential.value) return ''
  return [
    `员工：${props.email}`,
    `临时密码：${visibleCredential.value.temporary_password}`,
    `失效时间：${visibleCredential.value.expires_at}`,
    '首次登录必须立即修改密码。'
  ].join('\n')
}

const copyCredential = async (): Promise<void> => {
  const text = credentialText()
  if (!text) return
  await navigator.clipboard.writeText(text)
  acknowledged.value = true
}

const downloadCredential = (): void => {
  const text = credentialText()
  if (!text) return
  const url = URL.createObjectURL(new Blob([text], { type: 'text/plain;charset=utf-8' }))
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = `employee-credential-${props.email.replace(/[^a-zA-Z0-9.-]/g, '_')}.txt`
  anchor.click()
  URL.revokeObjectURL(url)
  acknowledged.value = true
}

const requestClose = (): void => {
  if (!acknowledged.value) return
  visibleCredential.value = null
  emit('close')
}
</script>
