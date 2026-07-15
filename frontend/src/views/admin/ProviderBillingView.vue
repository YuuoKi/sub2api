<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import {
  getPeriodSummary,
  importStatement,
  listImports,
  listMatches,
  previewImport,
  type ProviderBillingImport,
  type ProviderBillingImportHeader,
  type ProviderBillingMatch,
  type ProviderBillingPeriodSummary
} from '@/api/admin/providerBilling'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const imports = ref<ProviderBillingImport[]>([])
const summaries = ref<ProviderBillingPeriodSummary[]>([])
const selectedImportId = ref<number | null>(null)
const matches = ref<ProviderBillingMatch[]>([])
const previewLines = ref<unknown[]>([])
const previewSHA = ref('')
const duplicateBlocked = ref(false)

const form = ref<ProviderBillingImportHeader>({
  provider: 'seedance',
  provider_account_id: '',
  billing_period_start: new Date(new Date().getFullYear(), new Date().getMonth(), 1).toISOString(),
  billing_period_end: new Date(new Date().getFullYear(), new Date().getMonth() + 1, 0, 23, 59, 59).toISOString(),
  timezone: 'UTC',
  original_currency: 'USD',
  source_type: 'csv',
  invoice_number: ''
})
const file = ref<File | null>(null)

const diffQueue = computed(() => matches.value.filter((m) => m.match_status !== 'matched'))

async function refresh() {
  loading.value = true
  try {
    const [imp, sum] = await Promise.all([listImports(), getPeriodSummary()])
    imports.value = imp.items || []
    summaries.value = sum.items || []
  } catch (err) {
    appStore.showError(err instanceof Error ? err.message : String(err))
  } finally {
    loading.value = false
  }
}

async function onPreview() {
  if (!file.value) {
    appStore.showError(t('admin.providerBilling.fileRequired'))
    return
  }
  try {
    const result = await previewImport(form.value, file.value)
    previewLines.value = result.lines || []
    previewSHA.value = result.file_sha256
    duplicateBlocked.value = false
  } catch (err: any) {
    const code = err?.response?.data?.error?.reason || err?.response?.data?.reason || ''
    if (String(code).includes('DUPLICATE_FILE') || String(err?.message || '').includes('DUPLICATE_FILE')) {
      duplicateBlocked.value = true
    }
    appStore.showError(err instanceof Error ? err.message : String(err))
  }
}

async function onImport() {
  if (!file.value) {
    appStore.showError(t('admin.providerBilling.fileRequired'))
    return
  }
  try {
    const result = await importStatement(form.value, file.value)
    selectedImportId.value = result.import.id
    matches.value = result.matches || []
    duplicateBlocked.value = false
    await refresh()
  } catch (err: any) {
    const code = err?.response?.data?.error?.reason || err?.response?.data?.reason || ''
    if (String(code).includes('DUPLICATE')) {
      duplicateBlocked.value = true
    }
    appStore.showError(err instanceof Error ? err.message : String(err))
  }
}

async function openImport(id: number) {
  selectedImportId.value = id
  try {
    const result = await listMatches(id)
    matches.value = result.items || []
  } catch (err) {
    appStore.showError(err instanceof Error ? err.message : String(err))
  }
}

function onFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  file.value = input.files?.[0] || null
  if (file.value?.name.toLowerCase().endsWith('.xlsx')) {
    form.value.source_type = 'xlsx'
  } else {
    form.value.source_type = 'csv'
  }
}

function exportCSV() {
  if (!selectedImportId.value) return
  window.open(`/api/v1/admin/provider-billing/imports/${selectedImportId.value}/matches.csv`, '_blank')
}

onMounted(refresh)
</script>

<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
            {{ t('admin.providerBilling.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.providerBilling.subtitle') }}
          </p>
        </div>
        <button class="btn btn-secondary" :disabled="loading" @click="refresh">
          {{ t('common.refresh') }}
        </button>
      </div>

      <section class="card p-4">
        <h2 class="mb-3 text-lg font-medium">{{ t('admin.providerBilling.periodSummary') }}</h2>
        <div v-if="summaries.length === 0" class="text-sm text-gray-500">{{ t('common.noData') }}</div>
        <div v-else class="overflow-x-auto">
          <table class="min-w-full text-sm">
            <thead>
              <tr class="text-left text-gray-500">
                <th class="py-2 pr-4">Provider</th>
                <th class="py-2 pr-4">Account</th>
                <th class="py-2 pr-4">Period</th>
                <th class="py-2 pr-4">Conclusion</th>
                <th class="py-2 pr-4">Matched</th>
                <th class="py-2 pr-4">Diff</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(row, idx) in summaries" :key="idx" class="border-t border-gray-100 dark:border-dark-600">
                <td class="py-2 pr-4">{{ row.provider }}</td>
                <td class="py-2 pr-4">{{ row.provider_account_id }}</td>
                <td class="py-2 pr-4">{{ row.billing_period_start }} → {{ row.billing_period_end }}</td>
                <td class="py-2 pr-4 font-medium">{{ row.conclusion }}</td>
                <td class="py-2 pr-4">{{ row.matched }}</td>
                <td class="py-2 pr-4">{{ row.has_diff }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section class="card space-y-3 p-4">
        <h2 class="text-lg font-medium">{{ t('admin.providerBilling.importPreview') }}</h2>
        <div class="grid gap-3 md:grid-cols-2">
          <label class="text-sm">
            Provider
            <select v-model="form.provider" class="input mt-1 w-full">
              <option value="seedance">seedance</option>
              <option value="gemini">gemini</option>
            </select>
          </label>
          <label class="text-sm">
            Provider account ID
            <input v-model="form.provider_account_id" class="input mt-1 w-full" />
          </label>
          <label class="text-sm">
            Period start (RFC3339)
            <input v-model="form.billing_period_start" class="input mt-1 w-full" />
          </label>
          <label class="text-sm">
            Period end (RFC3339)
            <input v-model="form.billing_period_end" class="input mt-1 w-full" />
          </label>
          <label class="text-sm">
            Timezone
            <input v-model="form.timezone" class="input mt-1 w-full" />
          </label>
          <label class="text-sm">
            Currency
            <select v-model="form.original_currency" class="input mt-1 w-full">
              <option value="USD">USD</option>
              <option value="CNY">CNY</option>
            </select>
          </label>
          <label class="text-sm">
            Invoice number
            <input v-model="form.invoice_number" class="input mt-1 w-full" />
          </label>
          <label class="text-sm">
            Statement file (CSV/XLSX)
            <input type="file" accept=".csv,.xlsx" class="mt-1 block w-full text-sm" @change="onFileChange" />
          </label>
        </div>
        <div v-if="duplicateBlocked" class="rounded border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-800">
          {{ t('admin.providerBilling.duplicateBlocked') }}
        </div>
        <div class="flex gap-2">
          <button class="btn btn-secondary" @click="onPreview">{{ t('admin.providerBilling.preview') }}</button>
          <button class="btn btn-primary" @click="onImport">{{ t('admin.providerBilling.import') }}</button>
        </div>
        <p v-if="previewSHA" class="text-xs text-gray-500">SHA-256: {{ previewSHA }} · lines {{ previewLines.length }}</p>
      </section>

      <section class="card p-4">
        <h2 class="mb-3 text-lg font-medium">{{ t('admin.providerBilling.imports') }}</h2>
        <div class="overflow-x-auto">
          <table class="min-w-full text-sm">
            <thead>
              <tr class="text-left text-gray-500">
                <th class="py-2 pr-4">ID</th>
                <th class="py-2 pr-4">Provider</th>
                <th class="py-2 pr-4">SHA-256</th>
                <th class="py-2 pr-4">Lines</th>
                <th class="py-2 pr-4"></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in imports" :key="item.id" class="border-t border-gray-100 dark:border-dark-600">
                <td class="py-2 pr-4">{{ item.id }}</td>
                <td class="py-2 pr-4">{{ item.provider }} / {{ item.provider_account_id }}</td>
                <td class="py-2 pr-4 font-mono text-xs">{{ item.file_sha256.slice(0, 16) }}…</td>
                <td class="py-2 pr-4">{{ item.line_count }}</td>
                <td class="py-2 pr-4">
                  <button class="btn btn-secondary btn-sm" @click="openImport(item.id)">
                    {{ t('admin.providerBilling.viewMatches') }}
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section v-if="selectedImportId" class="card space-y-3 p-4">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-medium">
            {{ t('admin.providerBilling.matchDetail') }} #{{ selectedImportId }}
          </h2>
          <button class="btn btn-secondary" @click="exportCSV">{{ t('admin.providerBilling.exportCsv') }}</button>
        </div>
        <div>
          <h3 class="mb-2 text-sm font-medium text-gray-600">{{ t('admin.providerBilling.diffQueue') }}</h3>
          <div v-if="diffQueue.length === 0" class="text-sm text-gray-500">{{ t('admin.providerBilling.noDiff') }}</div>
          <ul v-else class="space-y-2 text-sm">
            <li v-for="m in diffQueue" :key="`${m.external_line_id}-${m.internal_ref_id}-${m.match_status}`" class="rounded border border-gray-200 p-2 dark:border-dark-600">
              <div class="font-medium">{{ m.match_status }} · {{ m.match_mode }}</div>
              <div class="text-gray-500">
                line={{ m.external_line_id || '-' }} internal={{ m.internal_ref_id || '-' }}
                amount {{ m.provider_amount }} / {{ m.internal_amount }}
              </div>
            </li>
          </ul>
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full text-sm">
            <thead>
              <tr class="text-left text-gray-500">
                <th class="py-2 pr-3">Status</th>
                <th class="py-2 pr-3">Mode</th>
                <th class="py-2 pr-3">External</th>
                <th class="py-2 pr-3">Internal</th>
                <th class="py-2 pr-3">Provider amt</th>
                <th class="py-2 pr-3">Internal amt</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="m in matches" :key="`${m.id}-${m.external_line_id}-${m.internal_ref_id}`" class="border-t border-gray-100 dark:border-dark-600">
                <td class="py-2 pr-3">{{ m.match_status }}</td>
                <td class="py-2 pr-3">{{ m.match_mode }}</td>
                <td class="py-2 pr-3">{{ m.external_line_id || '-' }}</td>
                <td class="py-2 pr-3">{{ m.internal_ref_id || '-' }}</td>
                <td class="py-2 pr-3">{{ m.provider_amount }}</td>
                <td class="py-2 pr-3">{{ m.internal_amount }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  </AppLayout>
</template>
