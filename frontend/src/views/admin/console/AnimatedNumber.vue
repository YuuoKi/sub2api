<template>
  <span>{{ formatted }}</span>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    value: number
    /** 动画时长（毫秒） */
    duration?: number
    /** 格式化函数，默认按整数千分位 */
    format?: (value: number) => string
  }>(),
  { duration: 900 }
)

const displayValue = ref(0)
let rafId: number | null = null

function animateTo(target: number) {
  if (rafId !== null) cancelAnimationFrame(rafId)
  const from = displayValue.value
  const delta = target - from
  if (Math.abs(delta) < 1e-9) {
    displayValue.value = target
    return
  }
  const startTs = performance.now()
  const step = (now: number) => {
    const progress = Math.min(1, (now - startTs) / props.duration)
    // easeOutCubic：数字滚动前快后慢，观感更像仪表盘
    const eased = 1 - Math.pow(1 - progress, 3)
    displayValue.value = from + delta * eased
    if (progress < 1) {
      rafId = requestAnimationFrame(step)
    } else {
      displayValue.value = target
      rafId = null
    }
  }
  rafId = requestAnimationFrame(step)
}

watch(
  () => props.value,
  (v) => animateTo(Number.isFinite(v) ? v : 0),
  { immediate: true }
)

onBeforeUnmount(() => {
  if (rafId !== null) cancelAnimationFrame(rafId)
})

const formatted = computed(() => {
  if (props.format) return props.format(displayValue.value)
  return Math.round(displayValue.value).toLocaleString()
})
</script>
