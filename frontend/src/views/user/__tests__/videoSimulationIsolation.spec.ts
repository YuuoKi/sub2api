import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')

function source(name: string): string {
  return readFileSync(resolve(root, name), 'utf8')
}

describe('employee video simulation view isolation', () => {
  it('never imports admin video APIs from employee simulation views', () => {
    for (const file of ['VideoCreateView.vue', 'VideoTasksView.vue', 'VideoTaskDetailView.vue']) {
      const text = source(file)
      expect(text).not.toMatch(/adminAPI\.video|@\/api\/admin\/video|from ['"]@\/api\/admin['"]/)
      expect(text).toMatch(/video_simulation|@\/api\/user\/video_simulation/)
    }
  })

  it('uses img for image/SVG media_kind previews, not a video element', () => {
    const detail = source('VideoTaskDetailView.vue')
    const list = source('VideoTasksView.vue')
    expect(detail).toMatch(/media_kind\s*===\s*['"]image['"]|mediaKind\s*===\s*['"]image['"]/)
    expect(list).toMatch(/media_kind\s*===\s*['"]image['"]|mediaKind\s*===\s*['"]image['"]/)
    expect(detail).not.toMatch(/<video[\s>]/)
    expect(list).not.toMatch(/<video[\s>]/)
  })
})
