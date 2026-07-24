import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const source = (path: string) => readFileSync(resolve(process.cwd(), path), 'utf8')

describe('canonical video admin surface', () => {
  it('keeps admin-only routes visible without demo hiding', () => {
    const router = source('src/router/index.ts')
    const sidebar = source('src/components/layout/AppSidebar.vue')
    const roleNav = source('src/components/layout/roleAwareNavigation.ts')
    for (const path of ['/admin/video/providers', '/admin/video/tasks', '/admin/video/system-check']) {
      expect(router).toContain(path)
    }
    // 旧视频通道页由密钥库视频 Tab 接管、视频链路检查并入系统健康页：
    // 侧栏只保留 tasks 入口（三条 URL 均保持可达，深链不受影响）
    for (const path of ['/admin/video/tasks']) {
      expect(roleNav).toContain(path)
    }
    expect(roleNav).not.toContain('/admin/video/providers')
    expect(router).not.toContain('video_gateway_demo')
    expect(sidebar).not.toContain('video_gateway_demo')
    expect(roleNav).not.toContain('video_gateway_demo')
    expect(router.match(/requiresAdmin:\s*true/g)?.length).toBeGreaterThanOrEqual(3)
  })

  it('exposes the operator evidence fields and one-time authorization', () => {
    const api = source('src/api/admin/video.ts')
    for (const field of ['upstream_task_id', 'result_url', 'cost_amount', 'real_dispatch_count', 'tiny_real_authorized_at']) {
      expect(api).toContain(field)
    }
    expect(api).toContain('tiny-real-authorization')
  })

  it('uses the canonical Seedance contract and explicit load failures', () => {
    const providers = source('src/views/admin/video/VideoProvidersView.vue')
    const api = source('src/api/admin/video.ts')
    const detail = source('src/views/admin/video/VideoTaskDetailView.vue')
    const system = source('src/views/admin/video/VideoSystemCheckView.vue')
    expect(providers).toContain('adminAPI.video.contract()')
    expect(providers).toContain("group.subscription_type === 'standard'")
    expect(providers).toContain('onlyStandardGroup')
    expect(providers).not.toContain('v-model="form.default_model"')
    expect(api).toContain('/admin/video/contract')
    expect(detail).toContain('extractApiErrorMessage')
    expect(system).toContain('extractApiErrorMessage')
  })
})
