<template>
  <div
    class="flex flex-col items-center px-6 py-10 text-center"
    data-testid="animated-empty-state"
    :data-variant="variant"
  >
    <!-- 视频任务空态:一条待生成的胶片,中间帧的播放键在呼吸 -->
    <svg
      v-if="variant === 'video-tasks'"
      class="h-24 w-32 text-ui-text-muted"
      viewBox="0 0 128 96"
      fill="none"
      role="img"
      :aria-label="title"
    >
      <g class="ui-anim-float">
        <rect x="14" y="24" width="100" height="48" rx="8" stroke="currentColor" stroke-width="2.5" />
        <line x1="52" y1="24" x2="52" y2="72" stroke="currentColor" stroke-width="2" opacity="0.55" />
        <line x1="76" y1="24" x2="76" y2="72" stroke="currentColor" stroke-width="2" opacity="0.55" />
        <g fill="currentColor" opacity="0.4">
          <rect x="20" y="30" width="7" height="5" rx="1.5" />
          <rect x="20" y="45" width="7" height="5" rx="1.5" />
          <rect x="20" y="60" width="7" height="5" rx="1.5" />
          <rect x="101" y="30" width="7" height="5" rx="1.5" />
          <rect x="101" y="45" width="7" height="5" rx="1.5" />
          <rect x="101" y="60" width="7" height="5" rx="1.5" />
        </g>
        <g class="ui-anim-breathe text-teal-500 dark:text-teal-300">
          <path d="M60 41.5 L71 48 L60 54.5 Z" fill="currentColor" />
        </g>
        <rect x="80" y="40" width="18" height="3" rx="1.5" fill="currentColor" opacity="0.35" />
        <rect x="80" y="47" width="12" height="3" rx="1.5" fill="currentColor" opacity="0.25" />
      </g>
    </svg>

    <!-- 视频总览空态:一块等内容的画板,角落的 teal 光点在呼吸 -->
    <svg
      v-else-if="variant === 'video-dashboard'"
      class="h-24 w-32 text-ui-text-muted"
      viewBox="0 0 128 96"
      fill="none"
      role="img"
      :aria-label="title"
    >
      <g class="ui-anim-float">
        <rect x="18" y="16" width="92" height="58" rx="8" stroke="currentColor" stroke-width="2.5" />
        <line x1="52" y1="82" x2="76" y2="82" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" opacity="0.6" />
        <line x1="64" y1="74" x2="64" y2="82" stroke="currentColor" stroke-width="2.5" opacity="0.6" />
        <rect x="28" y="28" width="34" height="4" rx="2" fill="currentColor" opacity="0.35" />
        <rect x="28" y="38" width="22" height="4" rx="2" fill="currentColor" opacity="0.25" />
        <path d="M30 62 L42 50 L52 57 L62 46" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" opacity="0.55" />
        <g class="ui-anim-breathe text-teal-500 dark:text-teal-300">
          <circle cx="88" cy="34" r="5" fill="currentColor" />
        </g>
      </g>
    </svg>

    <!-- 通用空态:一个空托盘,上方 teal 圆点轻轻漂浮 -->
    <svg
      v-else
      class="h-24 w-32 text-ui-text-muted"
      viewBox="0 0 128 96"
      fill="none"
      role="img"
      :aria-label="title"
    >
      <g class="ui-anim-float">
        <path
          d="M26 44 H102 L94 72 a8 8 0 0 1 -7.7 5.9 H41.7 A8 8 0 0 1 34 72 Z"
          stroke="currentColor"
          stroke-width="2.5"
          stroke-linejoin="round"
        />
        <path d="M26 44 L36 30 H92 L102 44" stroke="currentColor" stroke-width="2.5" stroke-linejoin="round" opacity="0.6" />
        <g class="ui-anim-breathe text-teal-500 dark:text-teal-300">
          <circle cx="64" cy="18" r="4.5" fill="currentColor" />
        </g>
      </g>
    </svg>

    <h3 class="mt-5 text-base font-semibold text-ui-text">{{ title }}</h3>
    <p v-if="description" class="ui-subheading mt-2 max-w-md leading-6">{{ description }}</p>
    <button v-if="actionLabel" class="btn btn-primary mt-5" type="button" @click="emit('action')">
      {{ actionLabel }}
    </button>
  </div>
</template>

<script setup lang="ts">
export type AnimatedEmptyStateVariant = 'video-tasks' | 'video-dashboard' | 'generic'

interface Props {
  variant?: AnimatedEmptyStateVariant
  title: string
  description?: string
  actionLabel?: string
}

withDefaults(defineProps<Props>(), {
  variant: 'generic',
  description: '',
  actionLabel: '',
})

const emit = defineEmits<{ action: [] }>()
</script>
