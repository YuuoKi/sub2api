<template>
  <div
    class="ui-page relative flex min-h-screen items-center justify-center overflow-hidden px-4"
  >
    <!-- Background Decorations -->
    <div class="pointer-events-none absolute inset-0 overflow-hidden">
      <div
        class="absolute -right-40 -top-40 h-80 w-80 rounded-full bg-ui-accent/10 blur-3xl"
      ></div>
      <div
        class="absolute -bottom-40 -left-40 h-80 w-80 rounded-full bg-ui-accent/10 blur-3xl"
      ></div>
    </div>

    <div class="relative z-10 w-full max-w-md text-center">
      <p class="mb-4 text-sm font-semibold tracking-wide text-ui-accent" data-testid="not-found-brand">
        {{ productName }}
      </p>

      <!-- 404 Display -->
      <div class="mb-8">
        <div class="relative inline-block">
          <span class="text-[12rem] font-bold leading-none text-gray-100 dark:text-dark-800"
            >404</span
          >
          <div class="absolute inset-0 flex items-center justify-center">
            <div
              class="flex h-24 w-24 items-center justify-center rounded-2xl bg-gradient-to-br from-ui-accent to-teal-600 shadow-lg shadow-teal-500/30"
            >
              <svg
                class="h-12 w-12 text-white"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="1.5"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z"
                />
              </svg>
            </div>
          </div>
        </div>
      </div>

      <!-- Text Content -->
      <div class="mb-8">
        <h1 class="mb-3 text-2xl font-bold text-ui-text">
          {{ t('errors.pageNotFound') }}
        </h1>
        <p class="text-ui-text-muted">
          {{ t('errors.pageNotFoundDesc') }}
        </p>
      </div>

      <!-- Action Buttons -->
      <div class="flex flex-col justify-center gap-3 sm:flex-row">
        <button @click="goBack" class="btn btn-secondary">
          <Icon name="arrowLeft" size="md" class="mr-2" />
          {{ t('errors.goBack') }}
        </button>
        <router-link to="/dashboard" class="btn btn-primary">
          <Icon name="home" size="md" class="mr-2" />
          {{ t('errors.goToDashboard') }}
        </router-link>
      </div>

      <!-- Help Link -->
      <p class="mt-8 text-sm text-ui-text-muted">
        {{ t('errors.needHelp') }}
        <a
          href="#"
          class="text-ui-accent transition-colors hover:opacity-80"
        >
          {{ t('errors.contactSupport') }}
        </a>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { resolveProductName } from '@/utils/productMode'

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()
const productName = computed(() => resolveProductName(appStore.siteName))

function goBack(): void {
  router.back()
}
</script>
