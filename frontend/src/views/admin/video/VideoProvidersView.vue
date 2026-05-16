<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 border-b border-gray-200 pb-4 dark:border-dark-700 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">Video Providers</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Mock, Seedance 2.0 and Kling upstream accounts</p>
        </div>
        <button class="btn btn-outline" type="button" :disabled="loading" @click="loadProviders">
          <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
          Refresh
        </button>
      </div>

      <div class="grid gap-6 xl:grid-cols-[1.25fr_0.75fr]">
        <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <div class="overflow-x-auto">
            <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
              <thead class="bg-gray-50 text-left text-xs uppercase text-gray-500 dark:bg-dark-700/40 dark:text-gray-400">
                <tr>
                  <th class="px-5 py-3 font-medium">Provider</th>
                  <th class="px-5 py-3 font-medium">Status</th>
                  <th class="px-5 py-3 font-medium">API Key</th>
                  <th class="px-5 py-3 font-medium">Model</th>
                  <th class="px-5 py-3 font-medium">Updated</th>
                  <th class="px-5 py-3 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                <tr v-for="provider in providers" :key="provider.id" :class="selected?.id === provider.id ? 'bg-gray-50 dark:bg-dark-700/40' : ''">
                  <td class="px-5 py-3">
                    <span class="inline-flex rounded-md px-2 py-1 text-xs font-medium" :class="providerBadgeClass(provider.provider)">
                      {{ provider.display_name }}
                    </span>
                    <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ provider.base_url || '-' }}</div>
                  </td>
                  <td class="px-5 py-3">
                    <button
                      type="button"
                      class="inline-flex rounded-md px-2 py-1 text-xs font-medium"
                      :class="provider.enabled ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300' : 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-200'"
                      :disabled="savingId === provider.id"
                      @click="toggleEnabled(provider)"
                    >
                      {{ provider.enabled ? 'Enabled' : 'Disabled' }}
                    </button>
                  </td>
                  <td class="px-5 py-3 text-gray-700 dark:text-gray-200">
                    {{ provider.api_key_configured ? provider.masked_key || 'Configured' : 'Not set' }}
                  </td>
                  <td class="px-5 py-3 text-gray-700 dark:text-gray-200">{{ provider.default_model || '-' }}</td>
                  <td class="px-5 py-3 text-gray-500 dark:text-gray-400">{{ formatDate(provider.updated_at) }}</td>
                  <td class="px-5 py-3">
                    <div class="flex gap-2">
                      <button class="btn btn-sm btn-outline" type="button" @click="selectProvider(provider)">
                        <Icon name="edit" size="xs" />
                        Edit
                      </button>
                      <button class="btn btn-sm btn-outline" type="button" :disabled="testingId === provider.id" @click="testProvider(provider.id)">
                        <Icon name="beaker" size="xs" />
                        Test
                      </button>
                    </div>
                  </td>
                </tr>
                <tr v-if="!loading && !providers.length">
                  <td colspan="6" class="px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400">No providers</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <section class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">Configuration</h2>
          <form v-if="selected" class="mt-4 space-y-4" @submit.prevent="saveProvider">
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Display Name</label>
              <input v-model="form.display_name" class="input" maxlength="120" />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Base URL</label>
              <input v-model="form.base_url" class="input" maxlength="500" />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Default Model</label>
              <input v-model="form.default_model" class="input" maxlength="200" />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">API Key</label>
              <input v-model="form.api_key" class="input" type="password" autocomplete="off" placeholder="Leave blank to keep current key" maxlength="4000" />
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Current: {{ selected.api_key_configured ? selected.masked_key || 'Configured' : 'Not set' }}</p>
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Rate Limit / minute</label>
              <input v-model.number="form.rate_limit_per_minute" class="input" type="number" min="0" max="100000" />
            </div>
            <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
              <input v-model="form.enabled" type="checkbox" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
              Enabled
            </label>
            <div class="flex flex-wrap gap-2">
              <button class="btn btn-primary" type="submit" :disabled="saving">
                <Icon name="check" size="sm" />
                Save
              </button>
              <button class="btn btn-outline" type="button" :disabled="testingId === selected.id" @click="testProvider(selected.id)">
                <Icon name="beaker" size="sm" />
                Test
              </button>
            </div>
          </form>
          <div v-else class="mt-6 text-sm text-gray-500 dark:text-gray-400">Select a provider</div>

          <div v-if="testResult" class="mt-5 rounded-lg border border-gray-200 bg-gray-50 p-4 text-sm dark:border-dark-700 dark:bg-dark-700/40">
            <div class="flex items-center justify-between gap-3">
              <span class="font-medium text-gray-900 dark:text-white">Last Test</span>
              <span class="rounded-md px-2 py-1 text-xs font-medium" :class="testResult.reachable ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300' : 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-200'">
                {{ testResult.reachable ? 'Reachable' : 'Skipped' }}
              </span>
            </div>
            <p class="mt-2 text-gray-600 dark:text-gray-300">{{ testResult.message }}</p>
            <details class="mt-3">
              <summary class="cursor-pointer text-xs font-medium text-gray-500 dark:text-gray-400">Payload preview</summary>
              <pre class="mt-2 max-h-64 overflow-auto rounded-md bg-white p-3 text-xs text-gray-700 dark:bg-dark-900 dark:text-gray-200">{{ JSON.stringify(testResult.payload_preview || {}, null, 2) }}</pre>
            </details>
          </div>
        </section>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type { VideoProviderAccount, VideoProviderTestResult } from '@/api/admin/video'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDate, providerBadgeClass } from './videoUtils'

const appStore = useAppStore()
const loading = ref(false)
const saving = ref(false)
const savingId = ref<number | null>(null)
const testingId = ref<number | null>(null)
const providers = ref<VideoProviderAccount[]>([])
const selected = ref<VideoProviderAccount | null>(null)
const testResult = ref<VideoProviderTestResult | null>(null)

const form = reactive({
  display_name: '',
  enabled: false,
  api_key: '',
  base_url: '',
  default_model: '',
  rate_limit_per_minute: 60,
})

function selectProvider(provider: VideoProviderAccount) {
  selected.value = provider
  form.display_name = provider.display_name
  form.enabled = provider.enabled
  form.api_key = ''
  form.base_url = provider.base_url
  form.default_model = provider.default_model
  form.rate_limit_per_minute = provider.rate_limit_per_minute
  testResult.value = null
}

async function loadProviders() {
  loading.value = true
  try {
    const res = await adminAPI.video.listProviders()
    providers.value = res.items || []
    const current = selected.value ? providers.value.find((item) => item.id === selected.value?.id) : providers.value[0]
    if (current) selectProvider(current)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, 'Failed to load video providers'))
  } finally {
    loading.value = false
  }
}

async function saveProvider() {
  if (!selected.value) return
  saving.value = true
  try {
    await adminAPI.video.updateProvider(selected.value.id, {
      display_name: form.display_name,
      enabled: form.enabled,
      base_url: form.base_url,
      default_model: form.default_model,
      rate_limit_per_minute: form.rate_limit_per_minute,
      ...(form.api_key.trim() ? { api_key: form.api_key.trim() } : {}),
    })
    form.api_key = ''
    appStore.showSuccess('Provider saved')
    await loadProviders()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, 'Failed to save provider'))
  } finally {
    saving.value = false
  }
}

async function toggleEnabled(provider: VideoProviderAccount) {
  savingId.value = provider.id
  try {
    await adminAPI.video.updateProvider(provider.id, { enabled: !provider.enabled })
    await loadProviders()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, 'Failed to update provider'))
  } finally {
    savingId.value = null
  }
}

async function testProvider(id: number) {
  testingId.value = id
  try {
    testResult.value = await adminAPI.video.testProvider(id)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, 'Provider test failed'))
  } finally {
    testingId.value = null
  }
}

onMounted(loadProviders)
</script>
