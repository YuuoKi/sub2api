<template>
  <BaseDialog
    :show="show"
    :title="title"
    width="narrow"
    :initial-focus="`#${inputId}`"
    @close="handleCancel"
  >
    <form class="space-y-4" @submit.prevent="handleConfirm">
      <p v-if="message" class="text-sm text-gray-600 dark:text-gray-400">
        {{ message }}
      </p>
      <div class="space-y-1.5">
        <label :for="inputId" class="block text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ label }}
        </label>
        <input
          :id="inputId"
          ref="inputRef"
          v-model="inputValue"
          :type="inputType"
          :placeholder="placeholder"
          :required="required"
          class="input w-full"
          autocomplete="off"
        />
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="handleCancel">
          {{ cancelText }}
        </button>
        <button
          type="button"
          class="btn btn-danger"
          :disabled="required && inputValue.length === 0"
          @click="handleConfirm"
        >
          {{ confirmText }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import BaseDialog from './BaseDialog.vue'

let promptInputCounter = 0
const inputId = `app-text-prompt-${++promptInputCounter}`

const props = withDefaults(defineProps<{
  show: boolean
  title: string
  message?: string
  label: string
  inputType?: 'text' | 'password'
  placeholder?: string
  required?: boolean
  confirmText: string
  cancelText: string
}>(), {
  message: '',
  inputType: 'text',
  placeholder: '',
  required: true
})

const emit = defineEmits<{
  (event: 'confirm', value: string): void
  (event: 'cancel'): void
}>()

const inputRef = ref<HTMLInputElement | null>(null)
const inputValue = ref('')

watch(
  () => props.show,
  async (show) => {
    if (!show) {
      inputValue.value = ''
      return
    }
    await nextTick()
    inputRef.value?.focus()
  }
)

function handleConfirm(): void {
  if (props.required && inputValue.value.length === 0) return
  emit('confirm', inputValue.value)
}

function handleCancel(): void {
  emit('cancel')
}
</script>
