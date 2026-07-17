import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const read = (name: string) =>
  readFileSync(resolve(dirname(fileURLToPath(import.meta.url)), `../${name}`), 'utf8')

describe('HomeView / KeyUsageView product name contract', () => {
  it('routes siteName through resolveProductName instead of raw cached site_name', () => {
    const home = read('HomeView.vue')
    const keyUsage = read('KeyUsageView.vue')

    expect(home).toContain("import { resolveProductName } from '@/utils/productMode'")
    expect(keyUsage).toContain("import { resolveProductName } from '@/utils/productMode'")
    expect(home).toMatch(/siteName\s*=\s*computed\(\s*\(\)\s*=>\s*resolveProductName\(/)
    expect(keyUsage).toMatch(/siteName\s*=\s*computed\(\s*\(\)\s*=>\s*resolveProductName\(/)
    expect(home).not.toMatch(
      /cachedPublicSettings\?\.site_name\s*\|\|\s*appStore\.siteName\s*\|\|\s*['"]无界/
    )
    expect(keyUsage).not.toMatch(
      /cachedPublicSettings\?\.site_name\s*\|\|\s*appStore\.siteName\s*\|\|\s*['"]无界/
    )
  })
})
