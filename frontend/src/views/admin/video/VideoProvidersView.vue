<template>
  <AppLayout>
    <div class="video-providers-view space-y-5">
      <header class="video-providers-header">
        <h1 class="video-providers-title text-2xl font-semibold">{{ copy.title }}</h1>
        <p class="video-providers-description text-sm text-gray-500">{{ copy.description }}</p>
      </header>
      <section v-if="contract" class="video-contract card p-5">
        <h2 class="video-contract-title text-base font-semibold">{{ copy.contract }}</h2>
        <dl class="video-contract-grid mt-3 grid gap-3 text-sm md:grid-cols-2">
          <div class="video-contract-field"><dt class="text-gray-500">{{ copy.model }}</dt><dd>{{ contract.default_model }}</dd></div>
          <div class="video-contract-field"><dt class="text-gray-500">{{ copy.endpoint }}</dt><dd class="break-all">{{ contract.base_url }}</dd></div>
        </dl>
      </section>
      <section class="video-provider-form card p-5">
        <h2 class="video-provider-form-title text-base font-semibold">{{ editingId ? copy.edit : copy.add }}</h2>
        <form class="video-provider-form-grid mt-4 grid gap-3 md:grid-cols-2" @submit.prevent="save">
          <input v-model="form.display_name" class="input video-provider-name" required :placeholder="copy.channelName" />
          <select v-model.number="form.group_id" class="input video-provider-group" required>
            <option :value="0">{{ copy.group }}</option>
            <option v-for="group in groups" :key="group.id" :value="group.id">{{ group.name }}</option>
          </select>
          <input v-model="form.api_key" class="input video-provider-secret" type="password" autocomplete="new-password" :required="!editingId" :placeholder="editingId ? copy.secretKeep : copy.secret" />
          <label class="video-provider-enabled flex items-center gap-2 text-sm"><input v-model="form.enabled" type="checkbox" />{{ copy.enableNow }}</label>
          <div class="video-provider-actions flex gap-2"><button class="btn btn-primary" :disabled="saving || !contract">{{ copy.save }}</button><button v-if="editingId" type="button" class="btn btn-secondary" @click="resetForm">{{ copy.cancel }}</button></div>
        </form>
      </section>
      <section class="video-provider-list grid gap-4 xl:grid-cols-2">
        <article v-for="provider in providers" :key="provider.id" class="video-provider-card card p-5">
          <div class="video-provider-summary flex items-start justify-between gap-3"><div class="video-provider-identity"><h2 class="font-semibold text-gray-900 dark:text-white">{{ provider.display_name }}</h2><p class="text-sm text-gray-500">{{ provider.group_name }} / {{ provider.default_model }}</p></div><span class="video-provider-state text-sm">{{ provider.enabled ? copy.enabled : copy.disabled }}</span></div>
          <dl class="video-provider-evidence mt-4 grid grid-cols-2 gap-3 text-sm"><div class="video-provider-secret-state"><dt class="text-gray-500">{{ copy.secretLabel }}</dt><dd>{{ provider.masked_key || copy.notConfigured }}</dd></div><div class="video-provider-grant-state"><dt class="text-gray-500">{{ copy.authorization }}</dt><dd>{{ authLabel(provider) }}</dd></div></dl>
          <div class="video-provider-card-actions mt-4 flex flex-wrap gap-2"><button class="btn btn-secondary" @click="startEdit(provider)">{{ copy.edit }}</button><button class="btn btn-secondary" @click="toggle(provider)">{{ provider.enabled ? copy.disable : copy.enable }}</button><button class="btn btn-primary" :disabled="!provider.enabled || !provider.api_key_configured || !!provider.tiny_real_authorized_at" @click="authorize(provider)">{{ copy.grant }}</button></div>
        </article>
        <p v-if="!providers.length" class="video-provider-empty text-sm text-gray-500">{{ copy.empty }}</p>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { adminAPI } from '@/api/admin'
import type { VideoProviderAccount, VideoProviderContract } from '@/api/admin/video'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'

const app = useAppStore()
const providers = ref<VideoProviderAccount[]>([])
const groups = ref<Array<{ id: number; name: string }>>([])
const contract = ref<VideoProviderContract>()
const saving = ref(false)
const editingId = ref<number | null>(null)
const form = reactive({ group_id: 0, display_name: 'Seedance 2.0', api_key: '', enabled: false })
const copy = {
  title: '\u89c6\u9891\u751f\u6210\u901a\u9053', description: '\u4ec5\u53ef\u7ed1\u5b9a\u53d7\u63a7\u6807\u51c6\u5458\u5de5\u7ec4\uff0c\u7531\u7cfb\u7edf\u56fa\u5b9a\u552f\u4e00\u6a21\u578b\u548c\u4e0a\u6e38\u5730\u5740\u3002', contract: '\u5f53\u524d\u56fa\u5b9a\u5951\u7ea6', model: '\u6a21\u578b', endpoint: '\u4e0a\u6e38\u5730\u5740', add: '\u65b0\u589e\u901a\u9053', edit: '\u7f16\u8f91\u901a\u9053', channelName: '\u901a\u9053\u540d\u79f0', group: '\u9009\u62e9\u53d7\u63a7\u6807\u51c6\u5458\u5de5\u7ec4', onlyStandardGroup: '\u4ec5\u663e\u793a\u53d7\u63a7\u6807\u51c6\u5458\u5de5\u7ec4', secret: '\u8f93\u5165\u4e0a\u6e38\u5bc6\u94a5', secretKeep: '\u7559\u7a7a\u8868\u793a\u4e0d\u66f4\u6362\u5bc6\u94a5', enableNow: '\u4fdd\u5b58\u540e\u542f\u7528', save: '\u4fdd\u5b58', cancel: '\u53d6\u6d88', enabled: '\u5df2\u542f\u7528', disabled: '\u5df2\u505c\u7528', secretLabel: '\u5bc6\u94a5', notConfigured: '\u672a\u914d\u7f6e', authorization: '\u5355\u6b21\u771f\u5b9e\u9a8c\u6536\u6388\u6743', grant: '\u6388\u6743\u4e00\u6b21\u6700\u5c0f\u771f\u5b9e\u8c03\u7528', enable: '\u542f\u7528', disable: '\u505c\u7528', empty: '\u5c1a\u672a\u914d\u7f6e\u901a\u9053\u3002', pending: '\u5df2\u6388\u6743\uff0c\u5f85\u6267\u884c', consumed: '\u5df2\u6d88\u8d39', notAuthorized: '\u672a\u6388\u6743', confirm: '\u786e\u8ba4\u6388\u6743\u8be5\u901a\u9053\u6267\u884c\u4e00\u6b21\u6700\u5c0f\u771f\u5b9e\u8c03\u7528\uff1f\u6b64\u64cd\u4f5c\u4e0d\u4f1a\u7acb\u5373\u53d1\u8d77\u4e0a\u6e38\u8bf7\u6c42\u3002'
}

async function load() { try { const [contractData, providerData, groupData] = await Promise.all([adminAPI.video.contract(), adminAPI.video.listProviders(), adminAPI.groups.getAll()]); contract.value = contractData; providers.value = providerData.items; groups.value = groupData.filter(group => group.subscription_type === 'standard') } catch (error) { app.showError(extractApiErrorMessage(error, '\u52a0\u8f7d\u89c6\u9891\u901a\u9053\u5931\u8d25')) } }
function resetForm() { editingId.value = null; Object.assign(form, { group_id: 0, display_name: 'Seedance 2.0', api_key: '', enabled: false }) }
function startEdit(provider: VideoProviderAccount) { editingId.value = provider.id; Object.assign(form, { group_id: provider.group_id, display_name: provider.display_name, api_key: '', enabled: provider.enabled }); window.scrollTo({ top: 0, behavior: 'smooth' }) }
async function save() { saving.value = true; try { if (editingId.value) { await adminAPI.video.updateProvider(editingId.value, { group_id: form.group_id, display_name: form.display_name, enabled: form.enabled, ...(form.api_key ? { api_key: form.api_key } : {}) }) } else { await adminAPI.video.createProvider({ ...form, provider: 'seedance' }) } resetForm(); app.showSuccess('\u901a\u9053\u5df2\u4fdd\u5b58'); await load() } catch (error) { app.showError(extractApiErrorMessage(error, '\u4fdd\u5b58\u5931\u8d25')) } finally { saving.value = false } }
async function toggle(provider: VideoProviderAccount) { try { await adminAPI.video.updateProvider(provider.id, { enabled: !provider.enabled }); await load() } catch (error) { app.showError(extractApiErrorMessage(error, '\u66f4\u65b0\u5931\u8d25')) } }
async function authorize(provider: VideoProviderAccount) { if (!window.confirm(copy.confirm)) return; try { await adminAPI.video.authorizeTinyReal(provider.id); app.showSuccess('\u5df2\u8bb0\u5f55\u5355\u6b21\u6388\u6743'); await load() } catch (error) { app.showError(extractApiErrorMessage(error, '\u6388\u6743\u5931\u8d25')) } }
function authLabel(provider: VideoProviderAccount) { if (provider.tiny_real_consumed_at) return copy.consumed; if (provider.tiny_real_authorized_at) return copy.pending; return copy.notAuthorized }
onMounted(load)
</script>
