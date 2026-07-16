<template>
  <section data-testid="gemini-api-key-edit-form" class="space-y-4">
    <div>
      <label :for="apiKeyInputId" class="input-label">{{ t('admin.accounts.apiKey') }}</label>
      <input
        :id="apiKeyInputId"
        :value="apiKey"
        data-testid="gemini-api-key-input"
        type="password"
        class="input font-mono"
        autocomplete="new-password"
        data-1p-ignore
        data-lpignore="true"
        data-bwignore="true"
        placeholder="AIza..."
        @input="$emit('update:apiKey', inputValue($event))"
      />
      <p class="input-hint">{{ t('admin.accounts.leaveEmptyToKeep') }}</p>
    </div>

    <GeminiAccountAdvancedFields
      test-id="gemini-api-key-advanced"
      :id-prefix="advancedFieldsIdPrefix"
      :show-base-url="true"
      :base-url="baseUrl"
      :base-url-hint="baseUrlHint"
      :proxies="proxies"
      :proxy-id="proxyId"
      :concurrency="concurrency"
      :load-factor="loadFactor"
      :priority="priority"
      :rate-multiplier="rateMultiplier"
      @update:base-url="$emit('update:baseUrl', $event)"
      @update:proxy-id="$emit('update:proxyId', $event)"
      @update:concurrency="$emit('update:concurrency', $event)"
      @update:load-factor="$emit('update:loadFactor', $event)"
      @update:priority="$emit('update:priority', $event)"
      @update:rate-multiplier="$emit('update:rateMultiplier', $event)"
    />

    <div
      v-if="review"
      data-testid="gemini-api-key-review"
      role="status"
      class="bg-blue-50 px-4 py-3 text-sm text-blue-800 dark:bg-blue-900/20 dark:text-blue-200"
    >
      <p class="font-medium">Gemini API Key · {{ t('common.confirm') }}</p>
      <dl class="mt-2 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
        <dt>{{ t('admin.accounts.apiKey') }}</dt>
        <dd class="font-mono">{{ maskedApiKey }}</dd>
        <dt>{{ t('admin.accounts.baseUrl') }}</dt>
        <dd class="break-all">{{ baseUrl }}</dd>
        <dt>{{ t('admin.accounts.concurrency') }}</dt>
        <dd>{{ concurrency }}</dd>
        <dt>{{ t('admin.accounts.billingRateMultiplier') }}</dt>
        <dd>{{ rateMultiplier }}</dd>
      </dl>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, getCurrentInstance } from 'vue'
import { useI18n } from 'vue-i18n'
import GeminiAccountAdvancedFields from './GeminiAccountAdvancedFields.vue'
import type { Proxy } from '@/types'
const defaultIdPrefix = `gemini-api-key-edit-${getCurrentInstance()?.uid ?? 'default'}`

const props = defineProps<{
  idPrefix?: string
  apiKey: string
  hasExistingApiKey: boolean
  baseUrl: string
  baseUrlHint: string
  proxies: Proxy[]
  proxyId: number | null
  concurrency: number
  loadFactor: number | null
  priority: number
  rateMultiplier: number
  review: boolean
}>()

defineEmits<{
  'update:apiKey': [value: string]
  'update:baseUrl': [value: string]
  'update:proxyId': [value: number | null]
  'update:concurrency': [value: number]
  'update:loadFactor': [value: number | null]
  'update:priority': [value: number]
  'update:rateMultiplier': [value: number]
}>()

const { t } = useI18n()
const resolvedIdPrefix = computed(() => props.idPrefix?.trim() || defaultIdPrefix)
const apiKeyInputId = computed(() => `${resolvedIdPrefix.value}-api-key`)
const advancedFieldsIdPrefix = computed(() => `${resolvedIdPrefix.value}-advanced`)
const maskedApiKey = computed(() => {
  const value = props.apiKey.trim()
  if (value) return `••••${value.slice(-4)}`
  return props.hasExistingApiKey ? '••••' : t('admin.accounts.apiKeyIsRequired')
})

function inputValue(event: Event): string {
  return event.target instanceof HTMLInputElement ? event.target.value : ''
}
</script>
