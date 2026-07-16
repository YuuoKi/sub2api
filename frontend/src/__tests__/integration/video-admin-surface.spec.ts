import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const source = (path: string) => readFileSync(resolve(process.cwd(), path), 'utf8')

describe('canonical video admin surface', () => {
  it('keeps admin-only routes visible without demo hiding', () => {
    const router = source('src/router/index.ts')
    const sidebar = source('src/components/layout/AppSidebar.vue')
    for (const path of ['/admin/video/providers', '/admin/video/tasks', '/admin/video/system-check']) {
      expect(router).toContain(path)
      expect(sidebar).toContain(path)
    }
    expect(router).not.toContain('video_gateway_demo')
    expect(sidebar).not.toContain('video_gateway_demo')
    expect(router.match(/requiresAdmin:\s*true/g)?.length).toBeGreaterThanOrEqual(3)
  })

  it('exposes the operator evidence fields and one-time authorization', () => {
    const api = source('src/api/admin/video.ts')
    for (const field of ['upstream_task_id', 'result_url', 'cost_amount', 'real_dispatch_count', 'tiny_real_authorized_at']) {
      expect(api).toContain(field)
    }
    expect(api).toContain('tiny-real-authorization')
  })
})
