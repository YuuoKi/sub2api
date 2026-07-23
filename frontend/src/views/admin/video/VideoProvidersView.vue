<template>
  <AppLayout>
    <div class="video-providers-view min-w-0 space-y-5 overflow-x-clip">
      <header class="video-providers-header">
        <h2 class="video-providers-title ui-heading">视频生成通道</h2>
        <p class="video-providers-description ui-subheading mt-1">仅绑定受控标准员工组；模型、时长、分辨率与上游地址由系统固定。</p>
      </header>

      <section v-if="contract" class="video-contract ui-panel p-5" aria-labelledby="video-contract-title">
        <h2 id="video-contract-title" class="video-contract-title text-base font-semibold">当前固定契约</h2>
        <dl class="video-contract-grid mt-3 grid gap-3 text-sm sm:grid-cols-2">
          <div class="video-contract-field"><dt class="text-gray-500">模型</dt><dd class="break-all">{{ contract.default_model }}</dd></div>
          <div class="video-contract-field"><dt class="text-gray-500">固定规格</dt><dd>{{ contract.duration_seconds }} 秒 / {{ contract.resolution }}</dd></div>
          <div class="video-contract-field sm:col-span-2"><dt class="text-gray-500">上游地址</dt><dd class="break-all">{{ contract.base_url }}</dd></div>
        </dl>
      </section>

      <section class="video-provider-form ui-panel p-5" aria-labelledby="video-provider-form-title">
        <h2 id="video-provider-form-title" class="video-provider-form-title text-base font-semibold">{{ editingId ? '编辑通道' : '新增通道' }}</h2>
        <form class="video-provider-form-grid mt-4 grid gap-4 md:grid-cols-2" @submit.prevent="save">
          <div class="video-provider-form-field">
            <label for="video-provider-name" class="mb-1 block text-sm font-medium">通道名称</label>
            <input id="video-provider-name" v-model="form.display_name" class="input video-provider-name w-full" required autocomplete="off" />
          </div>
          <div class="video-provider-form-field">
            <label for="video-provider-group" class="mb-1 block text-sm font-medium">受控标准员工组</label>
            <select id="video-provider-group" v-model.number="form.group_id" class="input video-provider-group w-full" required aria-describedby="video-provider-group-help">
              <option :value="0" disabled>选择员工组</option>
              <option v-for="group in groups" :key="group.id" :value="group.id">{{ group.name }}</option>
            </select>
            <p id="video-provider-group-help" class="mt-1 text-xs text-gray-500">{{ onlyStandardGroup }}</p>
          </div>
          <div class="video-provider-form-field">
            <label for="video-provider-secret" class="mb-1 block text-sm font-medium">上游 API Key</label>
            <input id="video-provider-secret" v-model="form.api_key" class="input video-provider-secret w-full" type="password" autocomplete="new-password" :required="!editingId" :placeholder="editingId ? '留空表示不更换' : '输入上游密钥'" />
            <p class="mt-1 text-xs text-gray-500">保存后只显示脱敏摘要，不回显明文。</p>
          </div>
          <div class="video-provider-toggles flex flex-col gap-2 self-end">
            <label class="video-provider-enabled flex min-h-6 items-center gap-2 text-sm">
              <input v-model="form.enabled" type="checkbox" />
              保存后启用
            </label>
            <label class="video-provider-authorize flex min-h-6 items-center gap-2 text-sm" data-test="authorize-after-save">
              <input v-model="form.authorize_after_save" type="checkbox" />
              保存后自动授权一次最小真实调用
            </label>
            <p class="text-xs text-gray-500">{{ AUTHORIZE_HINT }}</p>
          </div>
          <div class="video-provider-actions flex flex-col gap-2 sm:flex-row md:col-span-2">
            <button class="btn btn-primary" :disabled="saving || !contract">{{ saving ? '正在保存…' : '保存' }}</button>
            <button v-if="editingId" type="button" class="btn btn-secondary" @click="resetForm">取消编辑</button>
          </div>
        </form>
      </section>

      <section class="video-provider-list grid gap-4 xl:grid-cols-2" aria-label="已保存通道">
        <article v-for="provider in providers" :key="provider.id" class="video-provider-card ui-panel min-w-0 p-5">
          <div class="video-provider-summary flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
            <div class="video-provider-identity min-w-0">
              <h2 class="break-words font-semibold text-gray-900 dark:text-white">{{ provider.display_name }}</h2>
              <p class="break-all text-sm text-gray-500">{{ provider.group_name }} / {{ provider.default_model }}</p>
            </div>
            <span class="video-provider-state text-sm">{{ provider.enabled ? '已启用' : '已停用' }}</span>
          </div>
          <h3 class="mt-4 text-sm font-medium">保存后摘要</h3>
          <dl class="video-provider-evidence mt-2 grid gap-3 text-sm sm:grid-cols-2">
            <div class="video-provider-secret-state"><dt class="text-gray-500">密钥</dt><dd class="break-all">{{ provider.masked_key || '未配置' }}</dd></div>
            <div class="video-provider-group-state"><dt class="text-gray-500">员工组</dt><dd>{{ provider.group_name || '后端未提供' }}</dd></div>
            <div class="video-provider-contract-state"><dt class="text-gray-500">固定规格</dt><dd>{{ contract ? `${contract.duration_seconds} 秒 / ${contract.resolution}` : '加载失败' }}</dd></div>
            <div class="video-provider-grant-state"><dt class="text-gray-500">授权状态</dt><dd>{{ authLabel(provider) }}</dd></div>
          </dl>
          <div class="video-provider-card-actions mt-4 flex flex-col gap-2 sm:flex-row sm:flex-wrap">
            <button type="button" class="btn btn-secondary" @click="startEdit(provider)">编辑</button>
            <button type="button" class="btn btn-secondary" @click="toggle(provider)">{{ provider.enabled ? '停用' : '启用' }}</button>
            <button
              type="button"
              class="btn btn-primary"
              :data-testid="`authorize-provider-${provider.id}`"
              :disabled="!!authorizationDisabledReason(provider)"
              :aria-describedby="`video-auth-reason-${provider.id}`"
              @click="openAuthorization(provider)"
            >
              授权一次最小真实调用
            </button>
          </div>
          <p :id="`video-auth-reason-${provider.id}`" class="mt-2 text-xs text-gray-500" aria-live="polite">
            {{ authorizationDisabledReason(provider) || '授权前会再次展示模型、规格、预算门禁与影响范围。' }}
          </p>
        </article>
        <div v-if="!providers.length" class="video-provider-empty ui-panel md:col-span-2">
          <AnimatedEmptyState
            variant="generic"
            title="尚未配置通道"
            description="新增一条受控 Seedance 通道后即可开始授权与调度。"
          />
        </div>
      </section>

      <SeedanceAuthorizationDialog
        v-if="selectedProvider && contract"
        :show="authorizationOpen"
        :provider="selectedProvider"
        :contract="contract"
        :submitting="authorizing"
        @cancel="closeAuthorization"
        @confirm="confirmAuthorization"
      />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import AnimatedEmptyState from '@/components/common/AnimatedEmptyState.vue'
import { adminAPI } from '@/api/admin'
import type { VideoProviderAccount, VideoProviderContract } from '@/api/admin/video'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import SeedanceAuthorizationDialog from './SeedanceAuthorizationDialog.vue'

const app = useAppStore()
const providers = ref<VideoProviderAccount[]>([])
const groups = ref<Array<{ id: number; name: string }>>([])
const contract = ref<VideoProviderContract>()
const saving = ref(false)
const authorizing = ref(false)
const editingId = ref<number | null>(null)
const authorizationOpen = ref(false)
const selectedProvider = ref<VideoProviderAccount>()
const form = reactive({ group_id: 0, display_name: 'Seedance 2.0', api_key: '', enabled: true, authorize_after_save: true })
// 保存后自动完成一次性授权（后端门禁语义不变：授权只是记账，第一次真实出片才消费）
const AUTHORIZE_HINT = '保存后自动授权一次最小真实调用：这只是记账，不会马上扣费；等第一次真实出片后通道永久可用。'
const onlyStandardGroup = '仅显示受控标准员工组。'

async function load() {
  try {
    const [contractData, providerData, groupData] = await Promise.all([
      adminAPI.video.contract(),
      adminAPI.video.listProviders(),
      adminAPI.groups.getAll()
    ])
    contract.value = contractData
    providers.value = providerData.items
    groups.value = groupData.filter(group => group.subscription_type === 'standard')
  } catch (error) {
    app.showError(extractApiErrorMessage(error, '加载视频通道失败'))
  }
}

function resetForm() {
  editingId.value = null
  Object.assign(form, { group_id: 0, display_name: 'Seedance 2.0', api_key: '', enabled: true, authorize_after_save: true })
}

function startEdit(provider: VideoProviderAccount) {
  editingId.value = provider.id
  Object.assign(form, { group_id: provider.group_id, display_name: provider.display_name, api_key: '', enabled: provider.enabled, authorize_after_save: true })
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

// 保存成功后按勾选自动完成一次性授权；授权失败不掩盖「通道已保存」这个事实
async function save() {
  saving.value = true
  try {
    let savedId: number
    if (editingId.value) {
      savedId = editingId.value
      await adminAPI.video.updateProvider(editingId.value, {
        group_id: form.group_id,
        display_name: form.display_name,
        enabled: form.enabled,
        ...(form.api_key ? { api_key: form.api_key } : {})
      })
    } else {
      const created = await adminAPI.video.createProvider({
        group_id: form.group_id,
        display_name: form.display_name,
        enabled: form.enabled,
        api_key: form.api_key,
        provider: 'seedance'
      })
      savedId = created.id
    }
    const wantAuthorize = form.authorize_after_save
    resetForm()
    await load()
    if (wantAuthorize) {
      const saved = providers.value.find((p) => p.id === savedId)
      if (saved && !authorizationDisabledReason(saved)) {
        try {
          await adminAPI.video.authorizeTinyReal(savedId)
          app.showSuccess('通道已保存并记录单次授权；等第一次真实出片后通道永久可用')
          await load()
          return
        } catch (authError) {
          app.showError(`通道已保存，但自动授权失败：${extractApiErrorMessage(authError, '可在卡片上手动授权')}`)
          return
        }
      }
    }
    app.showSuccess('通道已保存；密钥明文不会再次回显')
  } catch (error) {
    app.showError(extractApiErrorMessage(error, '保存失败'))
  } finally {
    saving.value = false
  }
}

async function toggle(provider: VideoProviderAccount) {
  try {
    await adminAPI.video.updateProvider(provider.id, { enabled: !provider.enabled })
    app.showSuccess(provider.enabled ? `通道「${provider.display_name}」已停用` : `通道「${provider.display_name}」已启用`)
    await load()
  } catch (error) {
    app.showError(extractApiErrorMessage(error, '更新失败'))
  }
}

function authorizationDisabledReason(provider: VideoProviderAccount): string {
  if (!contract.value) return '固定契约尚未加载，不能授权。'
  if (!provider.enabled) return '通道已停用；先启用通道。'
  if (!provider.api_key_configured) return '密钥未配置；先保存上游密钥。'
  if (provider.tiny_real_consumed_at) return '该通道的一次性授权已经消费。'
  if (provider.tiny_real_authorized_at) return '该通道已有待消费授权，不能重复授权。'
  return ''
}

function openAuthorization(provider: VideoProviderAccount) {
  if (authorizationDisabledReason(provider)) return
  selectedProvider.value = provider
  authorizationOpen.value = true
}

function closeAuthorization() {
  if (authorizing.value) return
  authorizationOpen.value = false
}

async function confirmAuthorization() {
  if (!selectedProvider.value || authorizing.value) return
  authorizing.value = true
  try {
    await adminAPI.video.authorizeTinyReal(selectedProvider.value.id)
    authorizationOpen.value = false
    app.showSuccess('已记录单次授权；尚未发起上游请求')
    await load()
  } catch (error) {
    app.showError(extractApiErrorMessage(error, '授权失败'))
  } finally {
    authorizing.value = false
  }
}

function authLabel(provider: VideoProviderAccount): string {
  if (provider.tiny_real_consumed_at) return `已开通（首次真实出片完成于 ${provider.tiny_real_consumed_at}）`
  if (provider.tiny_real_authorized_at) return '待消费 = 等第一次真实出片，之后通道永久可用'
  return '未授权'
}

onMounted(load)
</script>
