<template>
  <div class="space-y-3">
    <!-- 实时态:真实脱敏样本墙 -->
    <div
      v-if="isLive && samples.length"
      class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3"
    >
      <div v-for="(s, idx) in samples" :key="idx" class="card space-y-2 p-4">
        <!-- 归因头部 -->
        <div class="flex flex-wrap items-center justify-between gap-1">
          <div class="flex flex-wrap items-center gap-1">
            <span class="badge badge-gray">{{ s.employee_name }}</span>
            <span class="badge badge-purple">{{ s.team_name }}</span>
            <span class="badge badge-primary">{{ s.model }}</span>
          </div>
          <span class="text-xs text-gray-400 dark:text-dark-400">{{
            formatRelativeTime(s.created_at)
          }}</span>
        </div>

        <!-- prompt 摘要 -->
        <div>
          <p class="mb-0.5 text-[11px] uppercase tracking-wide text-gray-400">
            {{ t('admin.generationContent.promptLabel') }}
          </p>
          <p class="line-clamp-2 text-sm text-gray-700 dark:text-gray-300">
            {{ s.prompt_preview || '—' }}
          </p>
        </div>

        <!-- response 摘要 -->
        <div>
          <p class="mb-0.5 text-[11px] uppercase tracking-wide text-gray-400">
            {{ t('admin.generationContent.responseLabel') }}
          </p>
          <p class="line-clamp-2 font-mono text-xs text-gray-500 dark:text-dark-400">
            {{ s.response_preview || '—' }}
          </p>
        </div>

        <!-- 底部:体量 + 截断标 -->
        <div class="flex items-center justify-between pt-1">
          <span class="text-xs text-gray-400">{{ formatBytes(s.total_bytes) }}</span>
          <span v-if="s.truncated" class="badge badge-warning">{{
            t('admin.generationContent.truncated')
          }}</span>
        </div>
      </div>
    </div>

    <!-- 空态/示例:未开启采集时,诚实展示占位,绝不伪造样本内容 -->
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
import { useI18n } from 'vue-i18n'
import { formatBytes, formatRelativeTime } from '@/utils/format'
import type { GenerationSample } from '@/api/admin/generation_content'

defineProps<{
  samples: GenerationSample[]
  isLive: boolean
}>()

const { t } = useI18n()
</script>
