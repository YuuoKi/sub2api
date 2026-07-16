<template>
  <section data-testid="gemini-vertex-edit-form" class="space-y-4">
    <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
      <div>
        <label :for="projectIdInputId" class="input-label">Project ID</label>
        <input
          :id="projectIdInputId"
          :value="projectId"
          type="text"
          class="input font-mono"
          readonly
          :placeholder="t('admin.accounts.vertexProjectIdPlaceholder')"
        />
      </div>
      <div>
        <label :for="clientEmailInputId" class="input-label">Client Email</label>
        <input :id="clientEmailInputId" :value="clientEmail" type="email" class="input font-mono" readonly />
      </div>
      <div class="sm:col-span-2">
        <label :for="locationSelectId" class="input-label">Location</label>
        <select
          :id="locationSelectId"
          :value="location"
          required
          class="input font-mono"
          @change="$emit('update:location', selectValue($event))"
        >
          <optgroup v-for="group in VERTEX_LOCATION_OPTIONS" :key="group.label" :label="group.label">
            <option v-for="option in group.options" :key="option.value" :value="option.value">
              {{ option.label }}
            </option>
          </optgroup>
        </select>
        <p class="input-hint">{{ t('admin.accounts.vertexLocationHint') }}</p>
      </div>
    </div>

    <GeminiAccountAdvancedFields
      test-id="gemini-vertex-advanced"
      :id-prefix="advancedFieldsIdPrefix"
      :show-base-url="false"
      base-url=""
      base-url-hint=""
      :proxies="proxies"
      :proxy-id="proxyId"
      :concurrency="concurrency"
      :load-factor="loadFactor"
      :priority="priority"
      :rate-multiplier="rateMultiplier"
      @update:proxy-id="$emit('update:proxyId', $event)"
      @update:concurrency="$emit('update:concurrency', $event)"
      @update:load-factor="$emit('update:loadFactor', $event)"
      @update:priority="$emit('update:priority', $event)"
      @update:rate-multiplier="$emit('update:rateMultiplier', $event)"
    />

    <div
      v-if="review"
      data-testid="gemini-vertex-review"
      role="status"
      class="bg-blue-50 px-4 py-3 text-sm text-blue-800 dark:bg-blue-900/20 dark:text-blue-200"
    >
      <p class="font-medium">Gemini Vertex · {{ t('common.confirm') }}</p>
      <dl class="mt-2 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
        <dt>Project ID</dt>
        <dd class="font-mono">{{ projectId }}</dd>
        <dt>Client Email</dt>
        <dd class="font-mono">{{ clientEmail }}</dd>
        <dt>Location</dt>
        <dd class="font-mono">{{ location }}</dd>
        <dt>{{ t('admin.accounts.vertexSaJsonLabel') }}</dt>
        <dd>{{ hasServiceAccountJson ? '••••' : t('admin.accounts.vertexSaJsonRequired') }}</dd>
      </dl>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, getCurrentInstance } from 'vue'
import { useI18n } from 'vue-i18n'
import GeminiAccountAdvancedFields from './GeminiAccountAdvancedFields.vue'
import { VERTEX_LOCATION_OPTIONS } from '@/constants/account'
import type { Proxy } from '@/types'
const defaultIdPrefix = `gemini-vertex-edit-${getCurrentInstance()?.uid ?? 'default'}`

const props = defineProps<{
  idPrefix?: string
  projectId: string
  clientEmail: string
  location: string
  hasServiceAccountJson: boolean
  proxies: Proxy[]
  proxyId: number | null
  concurrency: number
  loadFactor: number | null
  priority: number
  rateMultiplier: number
  review: boolean
}>()

defineEmits<{
  'update:location': [value: string]
  'update:proxyId': [value: number | null]
  'update:concurrency': [value: number]
  'update:loadFactor': [value: number | null]
  'update:priority': [value: number]
  'update:rateMultiplier': [value: number]
}>()

const { t } = useI18n()
const resolvedIdPrefix = computed(() => props.idPrefix?.trim() || defaultIdPrefix)
const projectIdInputId = computed(() => `${resolvedIdPrefix.value}-project-id`)
const clientEmailInputId = computed(() => `${resolvedIdPrefix.value}-client-email`)
const locationSelectId = computed(() => `${resolvedIdPrefix.value}-location`)
const advancedFieldsIdPrefix = computed(() => `${resolvedIdPrefix.value}-advanced`)

function selectValue(event: Event): string {
  return event.target instanceof HTMLSelectElement ? event.target.value : ''
}
</script>
