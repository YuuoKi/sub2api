<template>
  <main class="flex min-h-screen items-center justify-center bg-gray-50 px-4 dark:bg-dark-900">
    <section class="w-full max-w-md space-y-6 rounded-xl bg-white p-6 shadow-sm dark:bg-dark-800" aria-labelledby="force-password-title">
      <div class="space-y-2">
        <h1 id="force-password-title" class="text-xl font-semibold text-gray-900 dark:text-white">首次登录，请修改临时密码</h1>
        <p class="text-sm text-gray-600 dark:text-gray-300">完成改密前只能查看当前身份、修改密码或退出登录。</p>
      </div>
      <form class="space-y-4" @submit.prevent="submitPasswordChange">
        <div>
          <label for="temporary-password" class="input-label">当前临时密码</label>
          <input id="temporary-password" v-model="form.currentPassword" type="password" autocomplete="current-password" required class="input" />
        </div>
        <div>
          <label for="new-permanent-password" class="input-label">新密码</label>
          <input id="new-permanent-password" v-model="form.newPassword" type="password" autocomplete="new-password" minlength="8" required class="input" />
        </div>
        <div>
          <label for="confirm-permanent-password" class="input-label">确认新密码</label>
          <input id="confirm-permanent-password" v-model="form.confirmPassword" type="password" autocomplete="new-password" minlength="8" required class="input" />
        </div>
        <p v-if="errorMessage" class="text-sm text-red-600" role="alert">{{ errorMessage }}</p>
        <button type="submit" class="btn btn-primary w-full" :disabled="submitting">
          {{ submitting ? '正在修改…' : '修改密码并重新登录' }}
        </button>
      </form>
      <button type="button" class="btn btn-secondary w-full" @click="logout">退出登录</button>
    </section>
  </main>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { userAPI } from '@/api/user'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()
const submitting = ref(false)
const errorMessage = ref('')
const form = reactive({ currentPassword: '', newPassword: '', confirmPassword: '' })

const submitPasswordChange = async (): Promise<void> => {
  errorMessage.value = ''
  if (form.newPassword !== form.confirmPassword) {
    errorMessage.value = '两次输入的新密码不一致。'
    return
  }
  submitting.value = true
  try {
    await userAPI.changePassword(form.currentPassword, form.newPassword)
    await authStore.logout()
    await router.replace({ path: '/login', query: { passwordChanged: '1' } })
  } catch (error: unknown) {
    errorMessage.value = error instanceof Error ? error.message : '修改密码失败，请检查临时密码后重试。'
  } finally {
    submitting.value = false
  }
}

const logout = async (): Promise<void> => {
  await authStore.logout()
  await router.replace('/login')
}
</script>
