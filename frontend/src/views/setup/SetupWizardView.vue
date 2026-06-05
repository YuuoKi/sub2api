<template>
  <div class="flex min-h-screen items-center justify-center bg-gray-50 p-4 dark:bg-dark-900">
    <div class="w-full max-w-3xl">
      <div class="mb-8 text-center">
        <div
          class="mb-4 inline-flex h-16 w-16 items-center justify-center rounded-2xl bg-slate-950 text-xl font-semibold text-white shadow-lg dark:bg-emerald-600"
          role="img"
          aria-label="无界互娱"
        >
          无界
        </div>
        <p class="mb-2 text-sm font-semibold text-primary-600 dark:text-primary-400">内部试运行</p>
        <h1 class="text-3xl font-bold text-gray-900 dark:text-white">无界互娱 API 控制台</h1>
        <p class="mt-2 text-gray-500 dark:text-dark-400">AI 中剧 / 短剧生产网关</p>
      </div>

      <div class="mb-8 overflow-x-auto">
        <div class="flex min-w-[640px] items-center justify-center">
          <template v-for="(step, index) in steps" :key="step.id">
            <div class="flex items-center">
              <div
                :class="[
                  'flex h-10 w-10 items-center justify-center rounded-full text-sm font-semibold transition-all',
                  currentStep > index
                    ? 'bg-primary-500 text-white'
                    : currentStep === index
                      ? 'bg-primary-500 text-white ring-4 ring-primary-100 dark:ring-primary-900'
                      : 'bg-gray-200 text-gray-500 dark:bg-dark-700 dark:text-dark-400'
                ]"
              >
                <Icon v-if="currentStep > index" name="check" size="md" :stroke-width="2" />
                <span v-else>{{ index + 1 }}</span>
              </div>
              <span
                class="ml-2 text-sm font-medium"
                :class="currentStep >= index ? 'text-gray-900 dark:text-white' : 'text-gray-400 dark:text-dark-500'"
              >
                {{ step.title }}
              </span>
            </div>
            <div
              v-if="index < steps.length - 1"
              class="mx-3 h-0.5 w-12"
              :class="currentStep > index ? 'bg-primary-500' : 'bg-gray-200 dark:bg-dark-700'"
            ></div>
          </template>
        </div>
      </div>

      <div class="rounded-lg bg-white p-8 shadow-xl dark:bg-dark-800">
        <div v-if="currentStep === 0" class="space-y-6">
          <div class="text-center">
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">初始化本地控制台</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">首次使用前，先连接本机数据服务</p>
          </div>

          <div class="rounded-lg border border-gray-200 bg-gray-50 p-4 text-sm text-gray-600 dark:border-dark-700 dark:bg-dark-700/40 dark:text-gray-300">
            <div class="font-medium text-gray-900 dark:text-white">当前步骤</div>
            <p class="mt-1">系统会检查本机数据服务是否可用。老板和员工无需理解底层配置。</p>
          </div>

          <details class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
            <summary class="cursor-pointer text-sm font-semibold text-gray-900 dark:text-white">高级设置</summary>
            <div class="mt-4 grid gap-4 sm:grid-cols-2">
              <div>
                <label class="input-label">数据服务地址</label>
                <input v-model="formData.database.host" type="text" class="input" placeholder="localhost" />
              </div>
              <div>
                <label class="input-label">端口</label>
                <input v-model.number="formData.database.port" type="number" class="input" placeholder="5432" />
              </div>
              <div>
                <label class="input-label">用户名</label>
                <input v-model="formData.database.user" type="text" class="input" placeholder="postgres" />
              </div>
              <div>
                <label class="input-label">密码</label>
                <input v-model="formData.database.password" type="password" class="input" placeholder="密码" />
              </div>
              <div>
                <label class="input-label">数据名称</label>
                <input v-model="formData.database.dbname" type="text" class="input" placeholder="wujie_api" />
              </div>
              <div>
                <label class="input-label">连接方式</label>
                <select v-model="formData.database.sslmode" class="input">
                  <option value="disable">本机默认</option>
                  <option value="require">安全连接</option>
                  <option value="verify-ca">证书校验</option>
                  <option value="verify-full">完整校验</option>
                </select>
              </div>
            </div>
          </details>

          <button @click="testDatabaseConnection" :disabled="testingDb" class="btn btn-primary w-full">
            <Icon v-if="dbConnected" name="check" size="md" class="mr-2" :stroke-width="2" />
            {{ testingDb ? '正在检测...' : dbConnected ? '已连接，继续' : '检测并继续' }}
          </button>
        </div>

        <div v-if="currentStep === 1" class="space-y-6">
          <div class="text-center">
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">准备运行环境</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">检查本机运行队列和加速服务</p>
          </div>

          <div class="rounded-lg border border-gray-200 bg-gray-50 p-4 text-sm text-gray-600 dark:border-dark-700 dark:bg-dark-700/40 dark:text-gray-300">
            <div class="font-medium text-gray-900 dark:text-white">当前步骤</div>
            <p class="mt-1">系统会确认任务队列、状态缓存和后台运行依赖可用。</p>
          </div>

          <details class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
            <summary class="cursor-pointer text-sm font-semibold text-gray-900 dark:text-white">高级设置</summary>
            <div class="mt-4 grid gap-4 sm:grid-cols-2">
              <div>
                <label class="input-label">加速服务地址</label>
                <input v-model="formData.redis.host" type="text" class="input" placeholder="localhost" />
              </div>
              <div>
                <label class="input-label">端口</label>
                <input v-model.number="formData.redis.port" type="number" class="input" placeholder="6379" />
              </div>
              <div>
                <label class="input-label">密码</label>
                <input v-model="formData.redis.password" type="password" class="input" placeholder="可留空" />
              </div>
              <div>
                <label class="input-label">编号</label>
                <input v-model.number="formData.redis.db" type="number" class="input" placeholder="0" />
              </div>
            </div>
            <label class="mt-4 flex items-center justify-between rounded-lg border border-gray-200 p-3 text-sm dark:border-dark-700">
              <span>
                <span class="block font-medium text-gray-900 dark:text-white">加密连接</span>
                <span class="block text-xs text-gray-500 dark:text-dark-400">仅在管理员明确要求时开启</span>
              </span>
              <input v-model="formData.redis.enable_tls" type="checkbox" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
            </label>
          </details>

          <button @click="testRedisConnection" :disabled="testingRedis" class="btn btn-primary w-full">
            <Icon v-if="redisConnected" name="check" size="md" class="mr-2" :stroke-width="2" />
            {{ testingRedis ? '正在检测...' : redisConnected ? '已准备，继续' : '检测并继续' }}
          </button>
        </div>

        <div v-if="currentStep === 2" class="space-y-6">
          <div class="text-center">
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">创建管理员</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">设置第一个管理账号，用于进入控制台</p>
          </div>

          <div>
            <label class="input-label">邮箱</label>
            <input v-model="formData.admin.email" type="email" class="input" placeholder="admin@wujie.local" />
          </div>

          <div>
            <label class="input-label">密码</label>
            <input v-model="formData.admin.password" type="password" class="input" placeholder="至少 8 个字符" />
          </div>

          <div>
            <label class="input-label">确认密码</label>
            <input v-model="confirmPassword" type="password" class="input" placeholder="再次输入密码" />
            <p v-if="confirmPassword && formData.admin.password !== confirmPassword" class="input-error-text">
              两次输入的密码不一致
            </p>
          </div>
        </div>

        <div v-if="currentStep === 3" class="space-y-6">
          <div class="text-center">
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">完成安装</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">确认后即可进入内部试运行</p>
          </div>

          <div class="space-y-4">
            <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-700">
              <h3 class="mb-2 text-sm font-medium text-gray-500 dark:text-dark-400">本地数据</h3>
              <p class="break-words text-gray-900 dark:text-white">
                {{ formData.database.user }}@{{ formData.database.host }}:{{ formData.database.port }}/{{ formData.database.dbname }}
              </p>
            </div>

            <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-700">
              <h3 class="mb-2 text-sm font-medium text-gray-500 dark:text-dark-400">运行环境</h3>
              <p class="break-words text-gray-900 dark:text-white">{{ formData.redis.host }}:{{ formData.redis.port }}</p>
            </div>

            <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-700">
              <h3 class="mb-2 text-sm font-medium text-gray-500 dark:text-dark-400">管理员邮箱</h3>
              <p class="break-words text-gray-900 dark:text-white">{{ formData.admin.email }}</p>
            </div>
          </div>
        </div>

        <div v-if="errorMessage" class="mt-6 rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-800/50 dark:bg-red-900/20">
          <div class="flex items-start gap-3">
            <Icon name="exclamationCircle" size="md" class="flex-shrink-0 text-red-500" />
            <p class="text-sm text-red-700 dark:text-red-400">{{ errorMessage }}</p>
          </div>
        </div>

        <div v-if="installSuccess" class="mt-6 rounded-lg border border-green-200 bg-green-50 p-4 dark:border-green-800/50 dark:bg-green-900/20">
          <div class="flex items-start gap-3">
            <Icon name="checkCircle" size="md" class="flex-shrink-0 text-green-500" />
            <div>
              <p class="text-sm font-medium text-green-700 dark:text-green-400">安装完成</p>
              <p class="mt-1 text-sm text-green-600 dark:text-green-500">
                {{ serviceReady ? '正在跳转到登录页面...' : '服务正在准备，请稍候...' }}
              </p>
            </div>
          </div>
        </div>

        <div class="mt-8 flex justify-between">
          <button v-if="currentStep > 0 && !installSuccess" @click="currentStep--" class="btn btn-secondary">
            <Icon name="chevronLeft" size="sm" class="mr-2" :stroke-width="2" />
            上一步
          </button>
          <div v-else></div>

          <button v-if="currentStep === 2" @click="nextStep" :disabled="!canProceed" class="btn btn-primary">
            下一步
            <Icon name="chevronRight" size="sm" class="ml-2" :stroke-width="2" />
          </button>

          <button v-else-if="currentStep === 3 && !installSuccess" @click="performInstall" :disabled="installing" class="btn btn-primary">
            {{ installing ? '正在安装...' : '完成安装' }}
          </button>
        </div>
      </div>

      <footer class="mt-6 text-center text-xs text-gray-500 dark:text-dark-400">
        <p>© 2026 无界互娱 API 控制台</p>
        <p class="mt-1">内部试运行 · 正式上线未开启</p>
      </footer>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { testDatabase, testRedis, install, type InstallRequest } from '@/api/setup'
import Icon from '@/components/icons/Icon.vue'

const steps = [
  { id: 'database', title: '连接本地数据' },
  { id: 'runtime', title: '准备运行环境' },
  { id: 'admin', title: '创建管理员' },
  { id: 'complete', title: '完成安装' },
]

const currentStep = ref(0)
const errorMessage = ref('')
const installSuccess = ref(false)
const testingDb = ref(false)
const testingRedis = ref(false)
const dbConnected = ref(false)
const redisConnected = ref(false)
const installing = ref(false)
const confirmPassword = ref('')
const serviceReady = ref(false)

const getCurrentPort = (): number => {
  const port = window.location.port
  if (port) return parseInt(port, 10)
  return window.location.protocol === 'https:' ? 443 : 80
}

const formData = reactive<InstallRequest>({
  database: {
    host: 'localhost',
    port: 5432,
    user: 'postgres',
    password: '',
    dbname: 'wujie_api',
    sslmode: 'disable'
  },
  redis: {
    host: 'localhost',
    port: 6379,
    password: '',
    db: 0,
    enable_tls: false
  },
  admin: {
    email: '',
    password: ''
  },
  server: {
    host: '0.0.0.0',
    port: getCurrentPort(),
    mode: 'release'
  }
})

const canProceed = computed(() => (
  formData.admin.email &&
  formData.admin.password.length >= 8 &&
  formData.admin.password === confirmPassword.value
))

async function testDatabaseConnection() {
  testingDb.value = true
  errorMessage.value = ''
  dbConnected.value = false

  try {
    await testDatabase(formData.database)
    dbConnected.value = true
    currentStep.value = 1
  } catch (error: unknown) {
    const err = error as { response?: { data?: { detail?: string; message?: string } }; message?: string }
    errorMessage.value = err.response?.data?.detail || err.response?.data?.message || err.message || '连接失败'
  } finally {
    testingDb.value = false
  }
}

async function testRedisConnection() {
  testingRedis.value = true
  errorMessage.value = ''
  redisConnected.value = false

  try {
    await testRedis(formData.redis)
    redisConnected.value = true
    currentStep.value = 2
  } catch (error: unknown) {
    const err = error as { response?: { data?: { detail?: string; message?: string } }; message?: string }
    errorMessage.value = err.response?.data?.detail || err.response?.data?.message || err.message || '连接失败'
  } finally {
    testingRedis.value = false
  }
}

function nextStep() {
  if (canProceed.value) {
    errorMessage.value = ''
    currentStep.value = 3
  }
}

async function performInstall() {
  installing.value = true
  errorMessage.value = ''

  try {
    await install(formData)
    installSuccess.value = true
    waitForServiceRestart()
  } catch (error: unknown) {
    const err = error as { response?: { data?: { detail?: string; message?: string } }; message?: string }
    errorMessage.value = err.response?.data?.detail || err.response?.data?.message || err.message || '安装失败'
  } finally {
    installing.value = false
  }
}

async function waitForServiceRestart() {
  const maxAttempts = 60
  const interval = 1000

  await new Promise((resolve) => setTimeout(resolve, 3000))

  for (let attempt = 0; attempt < maxAttempts; attempt++) {
    try {
      const response = await fetch('/setup/status', {
        method: 'GET',
        cache: 'no-store'
      })

      if (response.ok) {
        const data = await response.json()
        if (data.data && !data.data.needs_setup) {
          serviceReady.value = true
          setTimeout(() => {
            window.location.href = '/login'
          }, 1500)
          return
        }
      }
    } catch {
      // Service may be unavailable while restarting.
    }

    await new Promise((resolve) => setTimeout(resolve, interval))
  }

  errorMessage.value = '服务准备时间超出预期，请手动刷新页面。'
}
</script>
