<template>
  <details
    :data-testid="testId"
    class="group rounded-lg bg-gray-50 dark:bg-dark-700/60"
  >
    <summary
      class="cursor-pointer list-none px-4 py-3 text-sm font-medium text-gray-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:text-gray-200"
    >
      {{ t('admin.accounts.proxy') }} · {{ t('admin.accounts.concurrency') }} ·
      {{ t('admin.accounts.billingRateMultiplier') }}
    </summary>

    <div class="space-y-4 px-4 pb-4">
      <div v-if="showBaseUrl">
        <label :for="controlId('base-url')" class="input-label">{{ t('admin.accounts.baseUrl') }}</label>
        <input
          :id="controlId('base-url')"
          :value="baseUrl"
          type="url"
          class="input"
          placeholder="https://generativelanguage.googleapis.com"
          @input="emitText('update:baseUrl', $event)"
        />
        <p class="input-hint">{{ baseUrlHint }}</p>
      </div>

      <div>
        <label :for="controlId('proxy')" class="input-label">{{ t('admin.accounts.proxy') }}</label>
        <ProxySelector
          :trigger-id="controlId('proxy')"
          :model-value="proxyId"
          :proxies="proxies"
          @update:model-value="$emit('update:proxyId', $event)"
        />
      </div>

      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div>
          <label :for="controlId('concurrency')" class="input-label">{{ t('admin.accounts.concurrency') }}</label>
          <input
            :id="controlId('concurrency')"
            :value="concurrency"
            type="number"
            min="1"
            class="input"
            @input="emitNumber('update:concurrency', $event)"
          />
        </div>
        <div>
          <label :for="controlId('load-factor')" class="input-label">{{ t('admin.accounts.loadFactor') }}</label>
          <input
            :id="controlId('load-factor')"
            :value="loadFactor ?? ''"
            type="number"
            min="1"
            class="input"
            :placeholder="String(concurrency)"
            @input="emitOptionalPositiveNumber('update:loadFactor', $event)"
          />
          <p class="input-hint">{{ t('admin.accounts.loadFactorHint') }}</p>
        </div>
        <div>
          <label :for="controlId('priority')" class="input-label">{{ t('admin.accounts.priority') }}</label>
          <input
            :id="controlId('priority')"
            :value="priority"
            type="number"
            min="1"
            class="input"
            @input="emitNumber('update:priority', $event)"
          />
        </div>
        <div>
          <label :for="controlId('rate-multiplier')" class="input-label">{{ t('admin.accounts.billingRateMultiplier') }}</label>
          <input
            :id="controlId('rate-multiplier')"
            :value="rateMultiplier"
            type="number"
            min="0"
            step="0.001"
            class="input"
            @input="emitNonNegativeNumber('update:rateMultiplier', $event)"
          />
        </div>
      </div>
    </div>
  </details>
</template>

<script setup lang="ts">
import { getCurrentInstance } from 'vue'
import { useI18n } from 'vue-i18n'
import ProxySelector from '@/components/common/ProxySelector.vue'
import type { Proxy } from '@/types'
const defaultIdPrefix = `gemini-account-advanced-${getCurrentInstance()?.uid ?? 'default'}`

const props = defineProps<{
  idPrefix?: string
  testId: string
  showBaseUrl: boolean
  baseUrl: string
  baseUrlHint: string
  proxies: Proxy[]
  proxyId: number | null
  concurrency: number
  loadFactor: number | null
  priority: number
  rateMultiplier: number
}>()

type NumericEvent =
  | 'update:concurrency'
  | 'update:loadFactor'
  | 'update:priority'
  | 'update:rateMultiplier'

const emit = defineEmits<{
  'update:baseUrl': [value: string]
  'update:proxyId': [value: number | null]
  'update:concurrency': [value: number]
  'update:loadFactor': [value: number | null]
  'update:priority': [value: number]
  'update:rateMultiplier': [value: number]
}>()

const { t } = useI18n()

function controlId(suffix: string): string {
  const prefix = props.idPrefix?.trim() || defaultIdPrefix
  return `${prefix}-${suffix}`
}

function inputValue(event: Event): string {
  return event.target instanceof HTMLInputElement ? event.target.value : ''
}

function emitText(event: 'update:baseUrl', source: Event) {
  emit(event, inputValue(source))
}

function emitNumber(event: Extract<NumericEvent, 'update:concurrency' | 'update:priority'>, source: Event) {
  const value = Number(inputValue(source))
  if (event === 'update:concurrency') {
    emit('update:concurrency', value)
  } else {
    emit('update:priority', value)
  }
}

function emitOptionalPositiveNumber(event: 'update:loadFactor', source: Event) {
  const raw = inputValue(source).trim()
  const value = Number(raw)
  emit(event, raw && Number.isFinite(value) && value >= 1 ? value : null)
}

function emitNonNegativeNumber(event: 'update:rateMultiplier', source: Event) {
  const value = Number(inputValue(source))
  emit(event, value)
}
</script>
