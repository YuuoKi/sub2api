<template>
  <BaseDialog
    :show="show"
    :title="t('admin.users.editUser')"
    width="normal"
    @close="$emit('close')"
  >
    <form v-if="user" id="edit-user-form" @submit.prevent="handleUpdateUser" class="space-y-5">
      <div>
        <label class="input-label">{{ t('admin.users.email') }}</label>
        <input v-model="form.email" type="email" class="input" />
      </div>
      <div>
        <label class="input-label">{{ t('admin.users.username') }}</label>
        <input v-model="form.username" type="text" class="input" />
      </div>
      <div>
        <label class="input-label">{{ t('admin.users.form.roleLabel') }}</label>
        <select v-model="form.role" class="input">
          <option value="user">{{ t('admin.users.roles.user') }}</option>
          <option value="admin">{{ t('admin.users.roles.admin') }}</option>
        </select>
      </div>
      <div>
        <label class="input-label">{{ t('admin.users.notes') }}</label>
        <textarea v-model="form.notes" rows="3" class="input"></textarea>
      </div>
      <div>
        <label class="input-label">{{ t('admin.users.columns.concurrency') }}</label>
        <input v-model.number="form.concurrency" type="number" class="input" />
      </div>
      <div>
        <label class="input-label">{{ t('admin.users.form.rpmLimit') }}</label>
        <input
          v-model.number="form.rpm_limit"
          type="number"
          min="0"
          step="1"
          class="input"
          :placeholder="t('admin.users.form.rpmLimitPlaceholder')"
        />
        <p class="input-hint">{{ t('admin.users.form.rpmLimitHint') }}</p>
      </div>
      <UserAttributeForm v-model="form.customAttributes" :user-id="user?.id" />
    </form>
    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" :disabled="submitting" @click="handleResetPassword">
          重置临时密码
        </button>
        <button @click="$emit('close')" type="button" class="btn btn-secondary">{{ t('common.cancel') }}</button>
        <button type="submit" form="edit-user-form" :disabled="submitting" class="btn btn-primary">
          {{ submitting ? t('admin.users.updating') : t('common.update') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { AdminUser, UserAttributeValuesMap } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import UserAttributeForm from '@/components/user/UserAttributeForm.vue'
import type { InitialCredential } from '@/api/admin/users'

const props = defineProps<{ show: boolean, user: AdminUser | null }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'success'): void
  (event: 'credential', payload: { email: string; credential: InitialCredential }): void
}>()
const { t } = useI18n(); const appStore = useAppStore()

const submitting = ref(false)
const form = reactive({ email: '', username: '', notes: '', role: 'user', concurrency: 1, rpm_limit: 0, customAttributes: {} as UserAttributeValuesMap })

watch(() => props.user, (u) => {
  if (u) {
    Object.assign(form, { email: u.email, username: u.username || '', notes: u.notes || '', role: u.role || 'user', concurrency: u.concurrency, rpm_limit: u.rpm_limit ?? 0, customAttributes: {} })
  }
}, { immediate: true })

const handleResetPassword = async (): Promise<void> => {
  if (!props.user) return
  submitting.value = true
  try {
    const credential = await adminAPI.users.resetPassword(props.user.id)
    emit('credential', { email: props.user.email, credential })
    emit('success')
    emit('close')
  } catch (error: unknown) {
    appStore.showError(error instanceof Error ? error.message : t('admin.users.failedToUpdate'))
  } finally {
    submitting.value = false
  }
}
const handleUpdateUser = async () => {
  if (!props.user) return
  if (!form.email.trim()) {
    appStore.showError(t('admin.users.emailRequired'))
    return
  }
  if (form.concurrency < 1) {
    appStore.showError(t('admin.users.concurrencyMin'))
    return
  }
  submitting.value = true
  try {
    const data = { email: form.email, username: form.username, notes: form.notes, role: form.role as 'admin' | 'user', concurrency: form.concurrency, rpm_limit: form.rpm_limit }
    await adminAPI.users.update(props.user.id, data)
    if (Object.keys(form.customAttributes).length > 0) await adminAPI.userAttributes.updateUserAttributeValues(props.user.id, form.customAttributes)
    appStore.showSuccess(t('admin.users.userUpdated'))
    emit('success'); emit('close')
  } catch (error: unknown) {
    appStore.showError(error instanceof Error ? error.message : t('admin.users.failedToUpdate'))
  } finally { submitting.value = false }
}
</script>
