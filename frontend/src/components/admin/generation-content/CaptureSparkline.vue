<template>
  <svg
    :viewBox="`0 0 ${width} ${height}`"
    :width="width"
    :height="height"
    class="text-primary-500 dark:text-primary-400"
    preserveAspectRatio="none"
    role="img"
    aria-hidden="true"
  >
    <polyline
      :points="points"
      fill="none"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
    <circle v-if="lastPoint" :cx="lastPoint.x" :cy="lastPoint.y" r="2" fill="currentColor" />
  </svg>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    series: number[]
    width?: number
    height?: number
  }>(),
  {
    width: 120,
    height: 32
  }
)

const PAD = 4 // 垂直内边距，避免顶/底点贴边

// 把 series 映射为 SVG 坐标点；n<2 时退化为一条水平基线。
const coords = computed(() => {
  const s = props.series ?? []
  const n = s.length
  const max = Math.max(...s, 1)
  const usableH = props.height - PAD * 2

  if (n < 2) {
    const y = props.height - PAD
    return [
      { x: 0, y },
      { x: props.width, y }
    ]
  }

  return s.map((v, i) => {
    const x = (i * props.width) / (n - 1)
    const y = props.height - PAD - (Math.max(v, 0) / max) * usableH
    return { x, y }
  })
})

const points = computed(() => coords.value.map((p) => `${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(' '))

const lastPoint = computed(() => {
  if ((props.series?.length ?? 0) < 1) return null
  return coords.value[coords.value.length - 1]
})
</script>
