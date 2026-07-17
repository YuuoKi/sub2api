<template>
  <span class="inline-flex items-center gap-2" role="status" :aria-label="ariaLabel">
    <svg
      :width="size"
      :height="size"
      viewBox="0 0 40 40"
      fill="none"
      aria-hidden="true"
    >
      <!-- 底环 -->
      <circle
        cx="20"
        cy="20"
        r="16"
        stroke="currentColor"
        stroke-width="4"
        class="text-gray-200 dark:text-dark-600"
      />
      <!--
        进度弧:不确定态用一段缺口弧整圈旋转(约 1.8s 一圈),
        只有父组件给出真实 progress(0-1)时才按 stroke-dashoffset 显示确定进度,
        绝不伪造百分比。
      -->
      <circle
        cx="20"
        cy="20"
        r="16"
        stroke="currentColor"
        stroke-width="4"
        stroke-linecap="round"
        class="text-teal-500 dark:text-teal-300"
        :class="{ 'ui-anim-ring': !isDeterminate }"
        :stroke-dasharray="arcDasharray"
        :stroke-dashoffset="arcDashoffset"
        :style="isDeterminate ? 'transition: stroke-dashoffset 400ms ease' : undefined"
      />
      <text
        v-if="centerText"
        x="20"
        y="24"
        text-anchor="middle"
        class="fill-current text-ui-text-muted"
        :font-size="isDeterminate ? 11 : 8.5"
        font-weight="600"
      >{{ centerText }}</text>
    </svg>
    <span v-if="sideLabel" class="text-xs text-ui-text-muted">{{ phase }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  /** 阶段文案,如 '排队中' / '生成中' / '即将完成';不定态时显示在环中心 */
  phase?: string
  /** 真实进度 0-1;仅在父组件确有进度数据时传入,缺省为不定态 */
  progress?: number
  /** 环直径(px) */
  size?: number
  /** 环右侧是否重复阶段文案(小尺寸环内文字过小时使用) */
  sideLabel?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  phase: '',
  progress: undefined,
  size: 36,
  sideLabel: false,
})

const RADIUS = 16
const CIRCUMFERENCE = 2 * Math.PI * RADIUS

const clampedProgress = computed(() => {
  if (typeof props.progress !== 'number' || Number.isNaN(props.progress)) return null
  return Math.min(1, Math.max(0, props.progress))
})

const isDeterminate = computed(() => clampedProgress.value !== null)

const arcDasharray = computed(() => (
  isDeterminate.value
    ? `${CIRCUMFERENCE}`
    : `${CIRCUMFERENCE * 0.28} ${CIRCUMFERENCE * 0.72}`
))

const arcDashoffset = computed(() => (
  isDeterminate.value
    ? CIRCUMFERENCE * (1 - (clampedProgress.value ?? 0))
    : 0
))

const centerText = computed(() => {
  if (isDeterminate.value) return `${Math.round((clampedProgress.value ?? 0) * 100)}%`
  return props.sideLabel ? '' : props.phase
})

const ariaLabel = computed(() => {
  if (isDeterminate.value) return `${props.phase || '进度'} ${Math.round((clampedProgress.value ?? 0) * 100)}%`
  return props.phase || '进行中'
})
</script>
