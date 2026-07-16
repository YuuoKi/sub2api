import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const read = (name: string) =>
  readFileSync(resolve(dirname(fileURLToPath(import.meta.url)), `../${name}`), 'utf8')

describe('shared app shell visual contract', () => {
  it('exposes stable shell landmarks', () => {
    expect(read('AppLayout.vue')).toContain('data-testid="app-shell"')
    expect(read('AppLayout.vue')).toContain('data-testid="app-main"')
    expect(read('AppHeader.vue')).toContain('data-testid="app-header"')
    expect(read('AppSidebar.vue')).toContain('data-testid="app-sidebar"')
  })

  it('keeps layout motion accessible', () => {
    expect(read('AppLayout.vue')).toContain('motion-reduce:transition-none')
  })
})
