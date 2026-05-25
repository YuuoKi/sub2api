<template>
  <main class="min-h-screen bg-slate-50 text-slate-950 dark:bg-dark-950 dark:text-white">
    <section class="border-b border-slate-200 bg-white dark:border-dark-800 dark:bg-dark-900">
      <div class="mx-auto flex max-w-6xl flex-col gap-8 px-6 py-10 lg:flex-row lg:items-end lg:justify-between">
        <div class="max-w-3xl">
          <p class="text-sm font-semibold uppercase tracking-normal text-teal-700 dark:text-teal-300">LAN-only 内部试运行</p>
          <h1 class="mt-3 text-3xl font-semibold tracking-normal text-slate-950 dark:text-white md:text-4xl">
            企业 AI 视频 API 调度中台
          </h1>
          <p class="mt-3 text-xl font-medium text-slate-800 dark:text-slate-100">
            AI 中剧 / AI 短剧生产网关
          </p>
          <p class="mt-4 max-w-2xl text-sm leading-6 text-slate-600 dark:text-slate-300">
            这是 Phase 4B.5 Day0 的内部试运行入口，只面向公司局域网和 safe demo / mock 验证；真实 provider、商业化和生产状态仍未就绪。
          </p>
        </div>

        <div class="grid w-full max-w-md grid-cols-2 gap-2 text-sm sm:grid-cols-3">
          <div v-for="item in readiness" :key="item.label" class="rounded border border-slate-200 bg-slate-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-800">
            <div class="font-semibold text-slate-900 dark:text-white">{{ item.label }}</div>
            <div :class="item.className" class="mt-1 text-xs font-semibold">{{ item.value }}</div>
          </div>
        </div>
      </div>
    </section>

    <section class="mx-auto max-w-6xl px-6 py-8">
      <div class="grid gap-3 md:grid-cols-5">
        <RouterLink
          v-for="action in primaryActions"
          :key="action.label"
          :to="action.to"
          class="inline-flex min-h-12 items-center justify-center gap-2 rounded border border-slate-300 bg-white px-3 py-2 text-center text-sm font-semibold text-slate-900 transition hover:border-teal-500 hover:text-teal-700 dark:border-dark-700 dark:bg-dark-900 dark:text-white dark:hover:border-teal-400 dark:hover:text-teal-200"
        >
          <Icon :name="action.icon" size="sm" />
          <span>{{ action.label }}</span>
        </RouterLink>
        <a
          href="#api-client"
          class="inline-flex min-h-12 items-center justify-center gap-2 rounded border border-slate-300 bg-white px-3 py-2 text-center text-sm font-semibold text-slate-900 transition hover:border-teal-500 hover:text-teal-700 dark:border-dark-700 dark:bg-dark-900 dark:text-white dark:hover:border-teal-400 dark:hover:text-teal-200"
        >
          <Icon name="terminal" size="sm" />
          <span>API client / n8n 调用说明</span>
        </a>
        <a
          href="#review-package"
          class="inline-flex min-h-12 items-center justify-center gap-2 rounded border border-slate-300 bg-white px-3 py-2 text-center text-sm font-semibold text-slate-900 transition hover:border-teal-500 hover:text-teal-700 dark:border-dark-700 dark:bg-dark-900 dark:text-white dark:hover:border-teal-400 dark:hover:text-teal-200"
        >
          <Icon name="document" size="sm" />
          <span>Phase 4B.5 审查包说明</span>
        </a>
      </div>
    </section>

    <section class="mx-auto grid max-w-6xl gap-5 px-6 pb-10 lg:grid-cols-[1.05fr_0.95fr]">
      <div class="rounded border border-slate-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900">
        <h2 class="text-base font-semibold text-slate-950 dark:text-white">Day0 入口口径</h2>
        <div class="mt-4 overflow-x-auto">
          <table class="min-w-full divide-y divide-slate-200 text-sm dark:divide-dark-700">
            <thead class="bg-slate-50 text-left text-xs uppercase text-slate-500 dark:bg-dark-800 dark:text-slate-400">
              <tr>
                <th class="px-3 py-2 font-semibold">路径</th>
                <th class="px-3 py-2 font-semibold">结论</th>
                <th class="px-3 py-2 font-semibold">用途</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-100 dark:divide-dark-800">
              <tr v-for="route in routeSummary" :key="route.path">
                <td class="px-3 py-3 font-mono text-xs text-slate-700 dark:text-slate-200">{{ route.path }}</td>
                <td class="px-3 py-3">
                  <span :class="route.className" class="rounded px-2 py-1 text-xs font-semibold">{{ route.status }}</span>
                </td>
                <td class="px-3 py-3 text-slate-600 dark:text-slate-300">{{ route.note }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div id="api-client" class="rounded border border-slate-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900">
        <h2 class="text-base font-semibold text-slate-950 dark:text-white">API client / n8n 统一口径</h2>
        <div class="mt-4 space-y-3 text-sm text-slate-600 dark:text-slate-300">
          <p><code>/api/v1/video/tasks</code> 用于通用视频任务：文生视频、图生视频、参考视频生成。</p>
          <p><code>/api/v1/drama/tasks</code> 用于 AI 中剧 / AI 短剧生产网关：剧种、镜头目标、prompt 结构、参考资产和 Skill 学习沉淀。</p>
          <p>内部试运行主入口使用 <code>/api/v1/drama/tasks</code>；通用视频任务保留 <code>/api/v1/video/tasks</code>。</p>
        </div>
        <pre class="mt-4 overflow-x-auto rounded bg-slate-950 p-4 text-xs leading-6 text-slate-100"><code>POST http://&lt;LAN_HOST&gt;:8080/api/v1/drama/tasks
Authorization: Bearer &lt;INTERNAL_PILOT_API_KEY&gt;
Content-Type: application/json</code></pre>
      </div>
    </section>

    <section id="review-package" class="mx-auto max-w-6xl px-6 pb-12">
      <div class="rounded border border-amber-200 bg-amber-50 p-5 text-sm text-amber-950 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-100">
        <h2 class="text-base font-semibold">Phase 4B.5 审查包说明</h2>
        <p class="mt-2">
          Day0 修复审查包路径：<code>03_审查包/PHASE_4B_5_DAY0_PUBLIC_AUTH_IDENTITY_FIX_REVIEW.html</code>。
          <code>/home</code> 已改为进入本内部试运行身份页，不再显示 upstream 默认页。
        </p>
      </div>
    </section>
  </main>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'

const readiness = [
  { label: 'Safe demo', value: 'mock only', className: 'text-teal-700 dark:text-teal-300' },
  { label: 'Production', value: 'NOT_READY', className: 'text-red-700 dark:text-red-300' },
  { label: 'Commercial', value: 'NOT_READY', className: 'text-red-700 dark:text-red-300' },
  { label: 'Real Provider', value: 'NOT_READY', className: 'text-red-700 dark:text-red-300' },
  { label: 'Network', value: 'LAN-only', className: 'text-amber-700 dark:text-amber-300' },
  { label: 'Scope', value: 'Phase 4B.5', className: 'text-slate-700 dark:text-slate-300' },
]

const primaryActions: Array<{ label: string; to: string; icon: 'login' | 'chart' | 'play' }> = [
  { label: '登录 / 管理后台', to: '/login', icon: 'login' },
  { label: '视频任务', to: '/admin/video/tasks', icon: 'chart' },
  { label: '短剧任务 / Drama Gateway', to: '/admin/video/create', icon: 'play' },
]

const routeSummary = [
  {
    path: '/',
    status: '内部入口',
    note: 'Day0 默认浏览器入口，进入本页。',
    className: 'bg-teal-100 text-teal-800 dark:bg-teal-500/20 dark:text-teal-200',
  },
  {
    path: '/internal-pilot',
    status: '内部入口',
    note: 'Phase 4B.5 LAN-only 入口说明和跳转面板。',
    className: 'bg-teal-100 text-teal-800 dark:bg-teal-500/20 dark:text-teal-200',
  },
  {
    path: '/home',
    status: '内部入口',
    note: '重定向到 /internal-pilot，不显示 upstream 营销页。',
    className: 'bg-teal-100 text-teal-800 dark:bg-teal-500/20 dark:text-teal-200',
  },
  {
    path: '/admin/video/*',
    status: '需登录',
    note: '视频网关驾驶舱、通道池、创建任务、任务列表和详情。',
    className: 'bg-slate-100 text-slate-800 dark:bg-dark-700 dark:text-slate-200',
  },
]
</script>
