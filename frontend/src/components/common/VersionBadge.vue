<template>
  <div class="relative">
    <template v-if="isAdmin">
      <button
        type="button"
        @click="toggleDropdown"
        class="flex items-center gap-1.5 rounded-lg bg-gray-100 px-2 py-1 text-xs text-gray-600 transition-colors hover:bg-gray-200 dark:bg-dark-800 dark:text-dark-400 dark:hover:bg-dark-700"
        :title="t('version.currentDeployVersion')"
      >
        <span v-if="displayVersion" class="font-medium" :title="displayVersion">{{
          displayVersion
        }}</span>
        <span
          v-else
          class="h-3 w-12 animate-pulse rounded bg-gray-200 font-medium dark:bg-dark-600"
        ></span>
      </button>

      <transition name="dropdown">
        <div
          v-if="dropdownOpen"
          ref="dropdownRef"
          class="absolute left-0 z-50 mt-2 w-72 overflow-hidden whitespace-normal rounded-xl border border-gray-200 bg-white shadow-lg transition-all duration-200 dark:border-dark-700 dark:bg-dark-800"
        >
          <div class="border-b border-gray-100 px-4 py-3 dark:border-dark-700">
            <span class="text-sm font-medium text-gray-700 dark:text-dark-300">{{
              t('version.currentDeployVersion')
            }}</span>
          </div>

          <div class="space-y-3 p-4">
            <div class="text-center">
              <button
                v-if="displayVersion"
                type="button"
                class="text-lg font-bold text-gray-900 dark:text-white"
                :title="t('version.copyFullVersion')"
                @click="copyToClipboard(displayVersion)"
              >
                {{ displayVersion }}
              </button>
              <span v-else class="text-lg font-bold text-gray-400 dark:text-dark-500">--</span>
              <p v-if="copied" class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                {{ t('version.copied') }}
              </p>
            </div>

            <dl class="space-y-2 text-left text-xs text-gray-600 dark:text-dark-300">
              <div class="flex flex-col gap-0.5">
                <dt class="text-gray-400 dark:text-dark-500">{{ t('version.buildCommit') }}</dt>
                <dd class="break-all font-mono text-gray-700 dark:text-dark-200">
                  {{ buildCommit || '--' }}
                </dd>
              </div>
              <div class="flex flex-col gap-0.5">
                <dt class="text-gray-400 dark:text-dark-500">{{ t('version.buildDate') }}</dt>
                <dd class="font-mono text-gray-700 dark:text-dark-200">{{ buildDate || '--' }}</dd>
              </div>
            </dl>
          </div>
        </div>
      </transition>
    </template>

    <span v-else-if="displayVersion" class="text-xs text-gray-500 dark:text-dark-400" :title="displayVersion">
      {{ displayVersion }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import { useClipboard } from '@/composables/useClipboard'

const { t } = useI18n()

const props = defineProps<{
  version?: string
}>()

const authStore = useAuthStore()
const appStore = useAppStore()

const isAdmin = computed(() => authStore.isAdmin)
const dropdownOpen = ref(false)
const dropdownRef = ref<HTMLElement | null>(null)

const displayVersion = computed(
  () => appStore.currentVersion || props.version || appStore.siteVersion || ''
)
const buildCommit = computed(() => appStore.buildCommit || '')
const buildDate = computed(() => appStore.buildDate || '')

const { copied, copyToClipboard } = useClipboard()

function toggleDropdown() {
  dropdownOpen.value = !dropdownOpen.value
}

function closeDropdown() {
  dropdownOpen.value = false
}

function handleClickOutside(event: MouseEvent) {
  const target = event.target as Node
  const button = (event.target as Element).closest('button')
  if (dropdownRef.value && !dropdownRef.value.contains(target) && !button?.contains(target)) {
    closeDropdown()
  }
}

onMounted(() => {
  if (isAdmin.value) {
    void appStore.fetchVersion(false)
  }
  document.addEventListener('click', handleClickOutside)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style scoped>
.dropdown-enter-active,
.dropdown-leave-active {
  transition: all 0.2s ease;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: scale(0.95) translateY(-4px);
}
</style>
