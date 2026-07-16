<template><AppLayout><div class="video-system-check space-y-5"><header class="video-system-header"><h1 class="video-system-title text-2xl font-semibold">{{ copy.title }}</h1><p class="video-system-description text-sm text-gray-500">{{ copy.description }}</p></header><button class="video-system-refresh btn btn-secondary" :disabled="loading" @click="load">{{ copy.run }}</button><p v-if="loading" class="video-system-loading text-sm text-gray-500">{{ copy.loading }}</p><p v-else-if="errorMessage" class="video-system-error text-sm text-red-600">{{ errorMessage }}</p><div v-else-if="check" class="video-system-grid grid gap-4 md:grid-cols-3"><div v-for="row in rows" :key="row[0]" class="video-system-card card p-5"><div class="video-system-label text-sm text-gray-500">{{ row[0] }}</div><div class="video-system-value mt-2 text-2xl font-semibold">{{ row[1] }}</div></div></div></div></AppLayout></template>
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { adminAPI } from '@/api/admin'
import type { VideoSystemCheck } from '@/api/admin/video'
import { extractApiErrorMessage } from '@/utils/apiError'
const check = ref<VideoSystemCheck>(); const loading = ref(false); const errorMessage = ref('')
const copy = { title: '\u89c6\u9891\u7cfb\u7edf\u68c0\u67e5', description: '\u53ea\u8bfb\u68c0\u67e5\u914d\u7f6e\u3001\u6388\u6743\u3001\u4efb\u52a1\u4e0e\u771f\u5b9e\u8c03\u5ea6\u8bc1\u636e\uff0c\u4e0d\u6267\u884c\u4e0a\u6e38\u8bf7\u6c42\u3002', run: '\u91cd\u65b0\u68c0\u67e5', loading: '\u6b63\u5728\u68c0\u67e5\u2026' }
const rows = computed(() => check.value ? [['\u901a\u9053\u6570', check.value.provider_count], ['\u5df2\u542f\u7528\u901a\u9053', check.value.enabled_provider_count], ['\u5f85\u6267\u884c\u6388\u6743', check.value.authorized_provider_count], ['\u4efb\u52a1\u6570', check.value.task_count], ['\u771f\u5b9e\u8c03\u5ea6\u6b21\u6570', check.value.real_dispatch_count], ['\u5168\u5c40\u5355\u6b21\u95e8\u7981', check.value.global_tiny_real_consumed ? '\u5df2\u6d88\u8d39' : '\u672a\u6d88\u8d39']] : [])
async function load() { loading.value = true; errorMessage.value = ''; try { check.value = await adminAPI.video.systemCheck() } catch (error) { check.value = undefined; errorMessage.value = extractApiErrorMessage(error, '\u7cfb\u7edf\u68c0\u67e5\u5931\u8d25') } finally { loading.value = false } }
onMounted(load)
</script>
