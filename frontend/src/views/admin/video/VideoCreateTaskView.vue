<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 border-b border-gray-200 pb-4 dark:border-dark-700 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">Create Video Task</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Submit a text, image or reference video task</p>
        </div>
        <RouterLink class="btn btn-outline" to="/admin/video/tasks">
          <Icon name="document" size="sm" />
          Tasks
        </RouterLink>
      </div>

      <div class="grid gap-6 xl:grid-cols-[0.95fr_1.05fr]">
        <section class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
          <form class="space-y-4" @submit.prevent="submitTask">
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Provider</label>
              <select v-model.number="form.provider_account_id" class="input" required @change="syncProviderModel">
                <option :value="0" disabled>Select provider</option>
                <option v-for="provider in enabledProviders" :key="provider.id" :value="provider.id">
                  {{ provider.display_name }} · {{ provider.default_model }}
                </option>
              </select>
            </div>

            <div class="grid gap-4 sm:grid-cols-2">
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Task Type</label>
                <select v-model="form.task_type" class="input">
                  <option v-for="item in taskTypeOptions" :key="item.value" :value="item.value">{{ item.label }}</option>
                </select>
              </div>
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Model</label>
                <input v-model="form.model" class="input" maxlength="200" />
              </div>
            </div>

            <div>
              <div class="mb-1 flex items-center justify-between gap-3">
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">Prompt</label>
                <button class="text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-300" type="button" @click="insertMockFailure">
                  Insert mock:fail
                </button>
              </div>
              <textarea v-model="form.prompt" class="input min-h-36 resize-y" maxlength="8000" required />
            </div>

            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Negative Prompt</label>
              <textarea v-model="form.negative_prompt" class="input min-h-20 resize-y" maxlength="4000" />
            </div>

            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Reference Image URL</label>
              <input v-model="form.reference_image_url" class="input" maxlength="1000" placeholder="https://..." />
            </div>

            <div class="grid gap-4 sm:grid-cols-3">
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Aspect Ratio</label>
                <select v-model="form.aspect_ratio" class="input">
                  <option value="16:9">16:9</option>
                  <option value="9:16">9:16</option>
                  <option value="1:1">1:1</option>
                  <option value="4:3">4:3</option>
                </select>
              </div>
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Duration</label>
                <input v-model.number="form.duration" class="input" type="number" min="1" max="60" />
              </div>
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Resolution</label>
                <select v-model="form.resolution" class="input">
                  <option value="720p">720p</option>
                  <option value="1080p">1080p</option>
                  <option value="480p">480p</option>
                </select>
              </div>
            </div>

            <button class="btn btn-primary" type="submit" :disabled="submitting || !form.provider_account_id">
              <Icon name="play" size="sm" />
              Submit
            </button>
          </form>
        </section>

        <section class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">Provider State</h2>
          <div class="mt-4 space-y-3">
            <div v-for="provider in providers" :key="provider.id" class="flex items-center justify-between gap-3 rounded-lg border border-gray-200 p-3 dark:border-dark-700">
              <div class="min-w-0">
                <span class="inline-flex rounded-md px-2 py-1 text-xs font-medium" :class="providerBadgeClass(provider.provider)">
                  {{ provider.display_name }}
                </span>
                <div class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">{{ provider.default_model || '-' }}</div>
              </div>
              <div class="text-right text-xs">
                <div :class="provider.enabled ? 'text-emerald-600 dark:text-emerald-300' : 'text-gray-500 dark:text-gray-400'">
                  {{ provider.enabled ? 'Enabled' : 'Disabled' }}
                </div>
                <div class="mt-1 text-gray-500 dark:text-gray-400">{{ provider.api_key_configured ? provider.masked_key || 'Configured' : 'No key' }}</div>
              </div>
            </div>
          </div>
          <RouterLink class="btn btn-outline mt-5" to="/admin/video/providers">
            <Icon name="key" size="sm" />
            Configure Providers
          </RouterLink>
        </section>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import { videoTaskAPI, type VideoProviderAccount, type VideoTaskCreatePayload } from '@/api/admin/video'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { providerBadgeClass, taskTypeOptions } from './videoUtils'

const router = useRouter()
const appStore = useAppStore()
const providers = ref<VideoProviderAccount[]>([])
const submitting = ref(false)

const form = reactive<VideoTaskCreatePayload>({
  provider_account_id: 0,
  task_type: 'text_to_video',
  model: '',
  prompt: 'Generate a concise product demo clip for a secure enterprise API console.',
  negative_prompt: '',
  reference_image_url: '',
  aspect_ratio: '16:9',
  duration: 5,
  resolution: '720p',
})

const enabledProviders = computed(() => providers.value.filter((provider) => provider.enabled))

async function loadProviders() {
  try {
    const res = await adminAPI.video.listProviders()
    providers.value = res.items || []
    const mock = enabledProviders.value.find((provider) => provider.provider === 'mock')
    const first = mock || enabledProviders.value[0]
    if (first && !form.provider_account_id) {
      form.provider_account_id = first.id
      form.model = first.default_model
    }
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, 'Failed to load video providers'))
  }
}

function syncProviderModel() {
  const provider = providers.value.find((item) => item.id === form.provider_account_id)
  if (provider?.default_model) {
    form.model = provider.default_model
  }
}

function insertMockFailure() {
  form.prompt = form.prompt.includes('mock:fail') ? form.prompt : `${form.prompt.trim()} mock:fail`.trim()
}

async function submitTask() {
  submitting.value = true
  try {
    const task = await videoTaskAPI.create({
      ...form,
      prompt: form.prompt.trim(),
      negative_prompt: form.negative_prompt?.trim(),
      reference_image_url: form.reference_image_url?.trim(),
      model: form.model?.trim(),
    })
    appStore.showSuccess('Video task created')
    router.push(`/admin/video/tasks/${task.id}`)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, 'Failed to create video task'))
  } finally {
    submitting.value = false
  }
}

onMounted(loadProviders)
</script>
