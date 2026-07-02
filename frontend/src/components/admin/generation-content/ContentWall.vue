<template>
  <div class="space-y-3">
    <div v-if="props.isLive && props.samples.length" class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
      <div v-for="(s, idx) in props.samples" :key="sampleKey(s, idx)" class="card space-y-3 p-4">
        <div class="flex flex-wrap items-center justify-between gap-1">
          <div class="flex flex-wrap items-center gap-1">
            <span class="badge badge-gray">{{ s.employee_name }}</span>
            <span class="badge badge-purple">{{ s.team_name }}</span>
            <span class="badge badge-primary">{{ s.model }}</span>
            <span v-if="s.task_id" class="badge badge-success">#{{ s.task_id }}</span>
          </div>
          <span class="text-xs text-gray-400 dark:text-dark-400">
            {{ formatRelativeTime(s.created_at) }}
          </span>
        </div>

        <div class="flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
          <span v-if="s.video_status" class="badge badge-gray">{{ s.video_status }}</span>
          <span>{{ formatCurrency(s.cost_estimate ?? 0) }}</span>
          <span>{{ formatBytes(s.total_bytes) }}</span>
          <span v-if="s.truncated" class="badge badge-warning">
            {{ t('admin.generationContent.truncated') }}
          </span>
        </div>

        <div>
          <p class="mb-0.5 text-[11px] uppercase tracking-wide text-gray-400">
            {{ t('admin.generationContent.promptLabel') }}
          </p>
          <p class="line-clamp-2 text-sm text-gray-700 dark:text-gray-300">
            {{ s.prompt_preview || '-' }}
          </p>
        </div>

        <div>
          <p class="mb-0.5 text-[11px] uppercase tracking-wide text-gray-400">
            {{ t('admin.generationContent.responseLabel') }}
          </p>
          <p class="line-clamp-2 font-mono text-xs text-gray-500 dark:text-dark-400">
            {{ s.response_preview || '-' }}
          </p>
        </div>

        <div v-if="s.task_id" class="grid grid-cols-1 gap-2 border-t border-gray-100 pt-3 dark:border-dark-700">
          <div class="grid grid-cols-[minmax(0,1fr)_88px] gap-2">
            <select v-model="drafts[sampleKey(s, idx)].adoption_status" class="input h-9">
              <option value="pending">pending</option>
              <option value="adopted">adopted</option>
              <option value="rejected">rejected</option>
            </select>
            <input
              v-model.number="drafts[sampleKey(s, idx)].quality_score"
              type="number"
              min="0"
              max="1"
              step="0.01"
              class="input h-9"
            />
          </div>
          <textarea
            v-model="drafts[sampleKey(s, idx)].notes"
            rows="2"
            class="input resize-none text-xs"
            maxlength="2048"
          />
          <button
            type="button"
            class="btn btn-secondary h-9"
            :disabled="saving[sampleKey(s, idx)]"
            @click="saveAdoption(s, idx)"
          >
            Save
          </button>
        </div>
      </div>
    </div>

    <div v-else class="card p-6 text-center">
      <p class="text-sm font-medium text-gray-600 dark:text-gray-300">
        {{ t('admin.generationContent.exampleBannerTitle') }}
      </p>
      <p class="mt-1 text-xs text-gray-400 dark:text-dark-400">
        {{ t('admin.generationContent.exampleBannerDesc') }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { formatBytes, formatCurrency, formatRelativeTime } from '@/utils/format'
import type { AdoptionStatus, GenerationSample } from '@/api/admin/generation_content'

interface Draft {
  adoption_status: AdoptionStatus
  quality_score: number | null | ''
  notes: string
}

const props = defineProps<{
  samples: GenerationSample[]
  isLive: boolean
}>()

const emit = defineEmits<{
  updated: []
}>()

const { t } = useI18n()
const drafts = reactive<Record<string, Draft>>({})
const saving = reactive<Record<string, boolean>>({})

function sampleKey(sample: GenerationSample, idx: number): string {
  return sample.task_id ? `task:${sample.task_id}` : `row:${idx}`
}

function syncDrafts() {
  props.samples.forEach((sample, idx) => {
    const key = sampleKey(sample, idx)
    if (!drafts[key]) {
      drafts[key] = {
        adoption_status: sample.adoption_status || 'pending',
        quality_score: sample.quality_score ?? null,
        notes: sample.adoption_notes || ''
      }
    }
  })
}

async function saveAdoption(sample: GenerationSample, idx: number) {
  if (!sample.task_id) return
  const key = sampleKey(sample, idx)
  const draft = drafts[key]
  if (!draft) return
  saving[key] = true
  try {
    await adminAPI.generationContent.updateAdoption(sample.task_id, {
      adoption_status: draft.adoption_status,
      quality_score: draft.quality_score === '' ? null : draft.quality_score,
      notes: draft.notes
    })
    emit('updated')
  } finally {
    saving[key] = false
  }
}

watch(() => props.samples, syncDrafts, { immediate: true })
</script>
