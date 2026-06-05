<template>
  <div class="relative flex min-h-screen items-center justify-center bg-slate-50 p-4 dark:bg-dark-950">
    <div class="w-full max-w-md">
      <div class="mb-8 text-center">
        <div
          class="mb-4 inline-flex h-14 w-14 items-center justify-center rounded-lg bg-slate-950 text-white shadow-lg shadow-slate-900/10 dark:bg-emerald-600"
          aria-hidden="true"
        >
          <svg class="h-8 w-8" viewBox="0 0 32 32" fill="none">
            <path d="M7 18.5c2.7-5 5.7-7.5 9-7.5 5.8 0 6.4 10 10 10 1.7 0 3-1.5 4-4" stroke="currentColor" stroke-width="2.3" stroke-linecap="round" />
            <path d="M2 15c1.4-2.7 3-4 4.8-4 4 0 4.5 10 9.2 10 3.1 0 6.1-2.5 9-7.5" stroke="currentColor" stroke-width="2.3" stroke-linecap="round" />
            <circle cx="7" cy="18.5" r="1.6" fill="currentColor" />
            <circle cx="25" cy="13.5" r="1.6" fill="currentColor" />
          </svg>
        </div>
        <h1 class="text-3xl font-semibold tracking-normal text-slate-950 dark:text-white">
          {{ brandName }}
        </h1>
        <p class="mt-2 text-sm font-medium text-slate-500 dark:text-dark-300">
          {{ siteSubtitle }}
        </p>
      </div>

      <div class="rounded-lg border border-slate-200 bg-white p-8 shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <slot />
      </div>

      <div class="mt-6 text-center text-sm">
        <slot name="footer" />
      </div>

      <div class="mt-8 text-center text-xs text-gray-400 dark:text-dark-500">
        <p>{{ networkLabel }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import {
  PUBLIC_AUTH_BRAND_NAME,
  PUBLIC_AUTH_NETWORK_LABEL,
  PUBLIC_AUTH_PRODUCT_SUBTITLE,
} from '@/utils/productMode'

const appStore = useAppStore()

const brandName = computed(() => PUBLIC_AUTH_BRAND_NAME)
const siteSubtitle = computed(() => PUBLIC_AUTH_PRODUCT_SUBTITLE)
const networkLabel = PUBLIC_AUTH_NETWORK_LABEL

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>
