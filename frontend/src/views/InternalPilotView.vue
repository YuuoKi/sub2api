<template>
  <main class="min-h-screen bg-slate-50 text-slate-950 dark:bg-dark-950 dark:text-white">
    <section class="mx-auto flex min-h-screen max-w-4xl flex-col items-center justify-center px-6 py-12 text-center">
      <div class="mb-5 inline-flex h-14 w-14 items-center justify-center rounded-lg bg-slate-950 text-white dark:bg-emerald-600" aria-hidden="true">
        <svg class="h-8 w-8" viewBox="0 0 32 32" fill="none">
          <path d="M7 18.5c2.7-5 5.7-7.5 9-7.5 5.8 0 6.4 10 10 10 1.7 0 3-1.5 4-4" stroke="currentColor" stroke-width="2.3" stroke-linecap="round" />
          <path d="M2 15c1.4-2.7 3-4 4.8-4 4 0 4.5 10 9.2 10 3.1 0 6.1-2.5 9-7.5" stroke="currentColor" stroke-width="2.3" stroke-linecap="round" />
        </svg>
      </div>
      <p class="text-sm font-semibold text-teal-700 dark:text-teal-300">内部使用</p>
      <h1 class="mt-3 text-3xl font-semibold tracking-normal md:text-4xl">AI 生产控制台</h1>
      <p class="mt-3 text-lg text-slate-600 dark:text-slate-300">中剧 / 短剧生产网关</p>
      <div class="mt-8 flex flex-wrap justify-center gap-3">
        <RouterLink class="btn btn-primary" to="/login">
          <Icon name="login" size="sm" />
          进入系统
        </RouterLink>
        <RouterLink class="btn btn-outline" to="/admin/video/system-check">
          <Icon name="shield" size="sm" />
          系统检查
        </RouterLink>
      </div>
    </section>
  </main>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'
</script>
