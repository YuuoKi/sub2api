<template><AppLayout><div class="space-y-5"><RouterLink to="/admin/video/tasks" class="text-sm text-primary-600">{{ copy.back }}</RouterLink><header><h1 class="text-2xl font-semibold">{{ copy.title }} #{{ task?.id }}</h1><p class="text-sm text-gray-500">{{ copy.description }}</p></header><div v-if="task" class="card grid gap-4 p-5 md:grid-cols-2"><div v-for="row in rows" :key="row[0]"><div class="text-xs text-gray-500">{{ row[0] }}</div><div class="mt-1 break-all text-sm">{{ row[1] || '-' }}</div></div><div class="md:col-span-2"><div class="text-xs text-gray-500">Asset URL</div><a v-if="task.result_url" :href="task.result_url" target="_blank" rel="noreferrer" class="break-all text-sm text-primary-600">{{ task.result_url }}</a><span v-else>-</span></div></div></div></AppLayout></template>
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import { adminAPI } from '@/api/admin'
import type { VideoTaskAdmin } from '@/api/admin/video'
const route = useRoute(); const task = ref<VideoTaskAdmin>()
const copy = { back: '\u8fd4\u56de\u4efb\u52a1\u8bc1\u636e', title: '\u89c6\u9891\u4efb\u52a1', description: '\u53ea\u5c55\u793a\u771f\u5b9e\u5b57\u6bb5\uff1b\u5931\u8d25\u8c03\u7528\u4e0d\u4f1a\u964d\u7ea7\u4e3a mock \u6210\u529f\u3002' }
const rows = computed(() => task.value ? [['State', task.value.status], ['Upstream task ID', task.value.upstream_task_id], ['Real dispatch count', String(task.value.real_dispatch_count)], ['Cost', `${task.value.cost_amount} ${task.value.currency || 'USD'}`], ['Dispatch state', task.value.dispatch_state], ['Error', task.value.error_message || task.value.provider_error_message], ['Model', task.value.model], ['Created by', String(task.value.created_by)]] : [])
onMounted(async () => { task.value = await adminAPI.video.getTask(Number(route.params.id)) })
</script>
