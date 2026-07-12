import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const routerPath = resolve(dirname(fileURLToPath(import.meta.url)), '../index.ts')
const routerSource = readFileSync(routerPath, 'utf8')

function routeBlock(path: string): string {
  const escapedPath = path.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const match = routerSource.match(new RegExp(`path: '${escapedPath}',[\\s\\S]*?\\n  },`))
  expect(match, `route ${path} should exist`).not.toBeNull()
  return match?.[0] ?? ''
}

describe('video route role boundaries', () => {
  it('keeps trial creation and task records available to authenticated employees', () => {
    expect(routeBlock('/admin/video/create')).toContain('requiresAdmin: false')
    expect(routeBlock('/admin/video/tasks')).toContain('requiresAdmin: false')
  })

  it('restricts system check to administrators', () => {
    expect(routeBlock('/admin/video/system-check')).toContain('requiresAdmin: true')
  })
})
