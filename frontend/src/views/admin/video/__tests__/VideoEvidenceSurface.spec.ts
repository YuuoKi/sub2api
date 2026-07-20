import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const source = (path: string) => readFileSync(resolve(process.cwd(), path), 'utf8')

describe('video task evidence surfaces', () => {
  it('keeps horizontal scrolling inside an announced table region with sticky key columns', () => {
    const list = source('src/views/admin/video/VideoTasksView.vue')
    expect(list).toContain('video-task-table-shell')
    expect(list).toContain('tabindex="0"')
    expect(list).toContain('aria-describedby="video-task-scroll-hint"')
    expect(list).toContain('position: sticky')
    expect(list).toContain('overflow-x: clip')
  })

  it('shows boss-readable evidence, previews, and explicit DTO gaps without inventing values', () => {
    const detail = source('src/views/admin/video/VideoTaskDetailView.vue')
    for (const label of ['请求人', '本地任务 ID', '上游任务 ID', '资产预览', '余额差分', '授权消费状态']) {
      expect(detail).toContain(label)
    }
    for (const heading of ['请求规格', '上游回传规格', '计费规格']) {
      expect(detail).toContain(heading)
    }
    expect(detail).toContain('后端未提供')
    expect(detail).toContain('<video')
    expect(detail).toContain('<img')
    expect(detail).toContain('video-spec-incomplete')
    expect(detail).toContain('task.value.request_duration_seconds')
    expect(detail).toContain('task.value.request_resolution')
    expect(detail).toContain('task.value.usage_total_tokens')
    expect(detail).toContain('task.value.reserved_cost_usd')
    expect(detail).toContain('task.value.reservation_state')
    expect(detail).toContain('task.value.provider_actual_cost_usd')
  })

  it('does not use native confirmation or prompt dialogs in the video admin views', () => {
    const providers = source('src/views/admin/video/VideoProvidersView.vue')
    const tasks = source('src/views/admin/video/VideoTasksView.vue')
    const detail = source('src/views/admin/video/VideoTaskDetailView.vue')
    for (const view of [providers, tasks, detail]) {
      expect(view).not.toContain('window.confirm')
      expect(view).not.toContain('window.prompt')
    }
  })

  it('keeps asset preview/download but removes the legacy QCanvas launcher and handoff API', () => {
    const detail = source('src/views/admin/video/VideoTaskDetailView.vue')
    expect(detail).not.toContain('发送视频到 QCanvas')
    expect(detail).not.toContain('QCanvas 本机地址')
    expect(detail).not.toContain('createAssetHandoff')
    expect(detail).not.toContain('startQCanvasAssetHandoffTransfer')
    expect(detail).toContain('在新窗口打开结果资产')
    expect(detail).toContain('在新窗口打开尾帧')
    expect(detail).not.toContain('sourceUrl=')
    expect(detail).not.toContain('apiKey=')
  })
})
