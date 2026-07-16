import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

// NOTE: repo convention — jsdom's global URL re-resolves relative URLs against
// http://localhost:3000 instead of the file:// base, so use dirname/resolve here
// (same pattern as src/components/layout/__tests__/AppSidebar.spec.ts).
const cssPath = resolve(dirname(fileURLToPath(import.meta.url)), '../style.css')
const css = readFileSync(cssPath, 'utf8')

describe('Apple-like visual token contract', () => {
  it.each([
    '--ui-canvas',
    '--ui-surface',
    '--ui-surface-raised',
    '--ui-border',
    '--ui-text',
    '--ui-text-muted',
    '--ui-accent',
    '--ui-focus'
  ])('defines %s', token => {
    expect(css).toContain(token)
  })

  it.each(['.ui-page', '.ui-panel', '.ui-toolbar', '.ui-heading', '.ui-subheading'])(
    'defines %s',
    selector => expect(css).toContain(selector)
  )

  it('supports reduced motion', () => {
    expect(css).toContain('@media (prefers-reduced-motion: reduce)')
  })
})
