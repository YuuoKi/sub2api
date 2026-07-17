import { readdirSync, readFileSync, statSync } from 'node:fs'
import { dirname, join, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../..')
const srcRoot = resolve(frontendRoot, 'src')

/**
 * User-visible upstream brand markers.
 * - Sub2API / Wei-Shaw / domains are always forbidden outside allowlist
 * - lowercase `sub2api` as a product/token word is also forbidden in scanned
 *   user-facing sources; storage-key / protocol allowlist lines stay exempt
 */
const FORBIDDEN =
  /Sub2API|Wei-Shaw|weishaw|wei-shaw|github\.com\/Wei-Shaw|sub2api\.io|sub2api\.org|(?<![A-Za-z0-9_])sub2api(?![A-Za-z0-9_])/g

/**
 * Explicit allowlist: internal compat constants, banner comments, and test
 * fixtures that intentionally assert upstream remaps. Paths are relative to
 * frontend/.
 */
const ALLOWLIST_PATH_PREFIXES = [
  'src/utils/productMode.ts',
  // Detector patterns intentionally mention upstream tokens
  'src/utils/complianceBrand.ts',
  'src/utils/__tests__/',
  'src/__tests__/brand-scan.spec.ts',
  'src/__tests__/integration/brand-admin-surface.spec.ts',
  'src/router/__tests__/title.spec.ts',
  'src/stores/__tests__/app.spec.ts',
  'src/stores/__tests__/adminCompliance.spec.ts',
  'src/views/auth/__tests__/',
  'src/views/__tests__/KeyUsageView.spec.ts',
  'src/views/__tests__/HomeView.brand.spec.ts',
  'src/views/admin/__tests__/SettingsView.spec.ts',
  'src/components/admin/__tests__/AdminComplianceDialog.spec.ts',
  'src/router/__tests__/wechat-route.spec.ts',
  'src/router/__tests__/feature-access.spec.ts',
  'src/utils/__tests__/ccswitchImport.spec.ts',
  'src/components/user/profile/__tests__/',
  'src/api/__tests__/'
] as const

/** Line-level allowlist for otherwise scanned files (comments / storage / protocols). */
const ALLOWLIST_LINE = [
  /^\s*\/[/*]/,
  /^\s*\*/,
  /<!--.*-->/,
  /UPSTREAM_DEFAULT_PRODUCT_NAME\s*=\s*['"]Sub2API['"]/,
  /site_name:\s*['"]Sub2API['"]/,
  /siteName:\s*['"]Sub2API['"]/,
  /providerName:\s*['"]Sub2API['"]/,
  /params\?\.siteName\s*\?\?\s*['"]Sub2API['"]/,
  /expect\([^)]*Sub2API/,
  /toBe\([^)]*Sub2API/,
  /toContain\([^)]*Sub2API/,
  /not\.toContain\([^)]*Sub2API/,
  /qr_code_url:.*Sub2API/,
  /split\(['"]Sub2API['"]\)/,
  // Storage keys / IndexedDB / cache ids
  /['"`]sub2api[_:-][A-Za-z0-9_:$\{\}.()-]+/,
  /`sub2api-ui-/,
  /`sub2api-ui-retry-/,
  /PREVIEW_CACHE_DB_NAME\s*=\s*['"]sub2api-/,
  /LOCALE_KEY\s*=\s*['"]sub2api_locale['"]/,
  /LOGIN_AGREEMENT_STORAGE_KEY\s*=\s*['"]sub2api_login_agreement_consent['"]/,
  /CACHE_STORAGE_KEY\s*=\s*['"]sub2api:/,
  // Wire protocols / export format markers (not product display)
  /OPS_WS_BASE_PROTOCOL\s*=\s*['"]sub2api-admin['"]/,
  /sub2api-asset-handoff-ticket/,
  /SUPPORTED_DATA_TYPES\s*=\s*\[[^\]]*sub2api-(data|bundle)/,
  /filename\s*=\s*`sub2api-(proxy|account)-/,
  /name:\s*sub2api-batch-image/,
  /providerName\s*=\s*\([^)]*\|\|\s*['"]sub2api['"]\)/,
  /\|\|\s*['"]sub2api['"]\)\.trim\(\)\s*\|\|\s*['"]sub2api['"]/,
  /dbname:\s*['"]sub2api['"]/,
  /placeholder=["']sub2api["']/,
  /sub2apipay/i
] as const

const SCANNED_EXTENSIONS = new Set(['.vue', '.ts', '.tsx', '.js', '.jsx', '.html'])

/** Must remain in the scanned set (brand fail-open surfaces). */
const MUST_SCAN = [
  'src/stores/adminCompliance.ts',
  'src/components/admin/AdminComplianceDialog.vue',
  'src/utils/complianceBrand.ts'
] as const

/** Paths allowlisted only for detector internals — still asserted as scanned. */
const DETECTOR_ALLOWLIST = new Set(['src/utils/complianceBrand.ts'])

function isAllowlistedPath(relPath: string): boolean {
  const normalized = relPath.replace(/\\/g, '/')
  return ALLOWLIST_PATH_PREFIXES.some(
    prefix => normalized === prefix || normalized.startsWith(prefix)
  )
}

function walk(dir: string, out: string[] = []): string[] {
  for (const entry of readdirSync(dir)) {
    if (entry === 'node_modules' || entry === 'dist' || entry === '.git') continue
    const full = join(dir, entry)
    const st = statSync(full)
    if (st.isDirectory()) {
      walk(full, out)
      continue
    }
    const ext = entry.includes('.') ? `.${entry.split('.').pop()}` : ''
    if (SCANNED_EXTENSIONS.has(ext)) out.push(full)
  }
  return out
}

function isAllowlistedLine(line: string): boolean {
  return ALLOWLIST_LINE.some(pattern => pattern.test(line))
}

describe('brand scan (source-level)', () => {
  it('canonical product name is 无界 · 企业 AI 中台', () => {
    const productMode = readFileSync(resolve(srcRoot, 'utils/productMode.ts'), 'utf8')
    expect(productMode).toContain("DEFAULT_PRODUCT_NAME = '无界 · 企业 AI 中台'")
    expect(productMode).toContain("UPSTREAM_DEFAULT_PRODUCT_NAME = 'Sub2API'")
  })

  it('scans compliance store and dialog sources', () => {
    const indexHtml = resolve(frontendRoot, 'index.html')
    const files = [indexHtml, ...walk(srcRoot)].map(f =>
      relative(frontendRoot, f).replace(/\\/g, '/')
    )
    for (const required of MUST_SCAN) {
      expect(files).toContain(required)
      if (!DETECTOR_ALLOWLIST.has(required)) {
        expect(isAllowlistedPath(required)).toBe(false)
      }
    }
  })

  it('rejects user-visible upstream brand hits outside allowlist', () => {
    const indexHtml = resolve(frontendRoot, 'index.html')
    const files = [indexHtml, ...walk(srcRoot)]
    const violations: string[] = []

    for (const file of files) {
      const rel = relative(frontendRoot, file).replace(/\\/g, '/')
      if (isAllowlistedPath(rel)) continue

      const content = readFileSync(file, 'utf8')
      const lines = content.split(/\r?\n/)
      lines.forEach((line, index) => {
        FORBIDDEN.lastIndex = 0
        if (!FORBIDDEN.test(line)) return
        FORBIDDEN.lastIndex = 0
        if (isAllowlistedLine(line)) return
        violations.push(`${rel}:${index + 1}: ${line.trim()}`)
      })
    }

    expect(violations, violations.join('\n')).toEqual([])
  })

  it('fails admin i18n help text that uses lowercase sub2api as product name', () => {
    const sample =
      "usageWindowsHint: 'not configured by sub2api, cannot lift from within sub2api.'"
    FORBIDDEN.lastIndex = 0
    expect(FORBIDDEN.test(sample)).toBe(true)
    expect(isAllowlistedLine(sample)).toBe(false)
  })
})
