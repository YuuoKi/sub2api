<template><AppLayout><div class="space-y-5"><header><h1 class="text-2xl font-semibold">{{ copy.title }}</h1><p class="text-sm text-gray-500">{{ copy.description }}</p></header><button class="btn btn-secondary" @click="load">{{ copy.run }}</button><div v-if="check" class="grid gap-4 md:grid-cols-3"><div v-for="row in rows" :key="row[0]" class="card p-5"><div class="text-sm text-gray-500">{{ row[0] }}</div><div class="mt-2 text-2xl font-semibold">{{ row[1] }}</div></div></div></div></AppLayout></template>
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { adminAPI } from '@/api/admin'
import type { VideoSystemCheck } from '@/api/admin/video'
const check = ref<VideoSystemCheck>()
const copy = { title: '\u89c6\u9891\u7cfb\u7edf\u68c0\u67e5', description: '\u53ea\u8bfb\u68c0\u67e5\u914d\u7f6e\u3001\u6388\u6743\u3001\u4efb\u52a1\u4e0e\u771f\u5b9e dispatch \u8bc1\u636e\uff0c\u4e0d\u6267\u884c\u4e0a\u6e38\u8bf7\u6c42\u3002', run: '\u91cd\u65b0\u68c0\u67e5' }
const rows = computed(() => check.value ? [['Providers', check.value.provider_count], ['Enabled providers', check.value.enabled_provider_count], ['Pending authorization', check.value.authorized_provider_count], ['Tasks', check.value.task_count], ['Real dispatches', check.value.real_dispatch_count], ['Global tiny_real', check.value.global_tiny_real_consumed ? 'consumed' : 'not consumed']] : [])
async function load() { check.value = await adminAPI.video.systemCheck() }
onMounted(load)
</script>
