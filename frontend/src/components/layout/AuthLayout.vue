<template>
  <div class="relative flex min-h-screen items-center justify-center overflow-hidden p-4">
    <!-- Background -->
    <div
      class="absolute inset-0 bg-gradient-to-br from-gray-50 via-primary-50/30 to-gray-100 dark:from-dark-950 dark:via-dark-900 dark:to-dark-950"
    ></div>

    <!-- Decorative Elements -->
    <div class="pointer-events-none absolute inset-0 overflow-hidden">
      <!-- Gradient Orbs -->
      <div
        class="absolute -right-40 -top-40 h-80 w-80 rounded-full bg-primary-400/20 blur-3xl"
      ></div>
      <div
        class="absolute -bottom-40 -left-40 h-80 w-80 rounded-full bg-primary-500/15 blur-3xl"
      ></div>
      <div
        class="absolute left-1/2 top-1/2 h-96 w-96 -translate-x-1/2 -translate-y-1/2 rounded-full bg-primary-300/10 blur-3xl"
      ></div>

      <!-- Grid Pattern -->
      <div
        class="absolute inset-0 bg-[linear-gradient(rgba(20,184,166,0.03)_1px,transparent_1px),linear-gradient(90deg,rgba(20,184,166,0.03)_1px,transparent_1px)] bg-[size:64px_64px]"
      ></div>
    </div>

    <!-- Content Container -->
    <div class="relative z-10 w-full max-w-md">
      <!-- Logo/Brand -->
      <div class="mb-8 text-center">
        <!-- Custom Logo or Default Logo -->
        <div
          class="mb-4 inline-flex h-16 w-16 items-center justify-center overflow-hidden rounded-2xl shadow-lg shadow-primary-500/30"
        >
          <img
            :src="siteLogo || '/logo.png'"
            alt="企业 AI 视频 API 调度中台"
            class="h-full w-full object-contain"
          />
        </div>
        <h1 class="text-gradient mb-2 text-3xl font-bold">
          {{ siteName }}
        </h1>
        <p class="text-sm font-medium text-gray-600 dark:text-dark-300">
          {{ siteSubtitle }}
        </p>
        <div class="mt-3 flex flex-wrap justify-center gap-2 text-[11px] font-semibold">
          <span
            v-for="item in identityLabels"
            :key="item"
            class="rounded border border-gray-200 bg-white/70 px-2 py-1 text-gray-600 dark:border-dark-700 dark:bg-dark-800/70 dark:text-dark-300"
          >
            {{ item }}
          </span>
        </div>
      </div>

      <!-- Card Container -->
      <div class="card-glass rounded-2xl p-8 shadow-glass">
        <slot />
      </div>

      <!-- Footer Links -->
      <div class="mt-6 text-center text-sm">
        <slot name="footer" />
      </div>

      <!-- Copyright -->
      <div class="mt-8 space-y-1 text-center text-xs text-gray-400 dark:text-dark-500">
        <p>&copy; {{ currentYear }} {{ siteName }}. {{ networkLabel }}.</p>
        <p>{{ productionStatus }} · {{ commercialStatus }} · {{ realProviderStatus }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'
import {
  PUBLIC_AUTH_COMMERCIAL_STATUS,
  PUBLIC_AUTH_NETWORK_LABEL,
  PUBLIC_AUTH_PRODUCT_NAME,
  PUBLIC_AUTH_PRODUCT_SUBTITLE,
  PUBLIC_AUTH_PRODUCTION_STATUS,
  PUBLIC_AUTH_REAL_PROVIDER_STATUS,
  PUBLIC_AUTH_SAFE_DEMO_LABEL,
} from '@/utils/productMode'

const appStore = useAppStore()

const siteName = computed(() => PUBLIC_AUTH_PRODUCT_NAME)
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => PUBLIC_AUTH_PRODUCT_SUBTITLE)
const networkLabel = PUBLIC_AUTH_NETWORK_LABEL
const productionStatus = PUBLIC_AUTH_PRODUCTION_STATUS
const commercialStatus = PUBLIC_AUTH_COMMERCIAL_STATUS
const realProviderStatus = PUBLIC_AUTH_REAL_PROVIDER_STATUS
const identityLabels = [
  PUBLIC_AUTH_NETWORK_LABEL,
  PUBLIC_AUTH_SAFE_DEMO_LABEL,
  PUBLIC_AUTH_PRODUCTION_STATUS,
  PUBLIC_AUTH_COMMERCIAL_STATUS,
  PUBLIC_AUTH_REAL_PROVIDER_STATUS,
]

const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
.text-gradient {
  @apply bg-gradient-to-r from-primary-600 to-primary-500 bg-clip-text text-transparent;
}
</style>
