import { describe, expect, it } from 'vitest'
import { resolveDocumentTitle } from '@/router/title'
import { VIDEO_GATEWAY_PRODUCT_NAME } from '@/utils/productMode'

describe('resolveDocumentTitle', () => {
  it('默认产品模式下，路由标题固定使用内部产品名', () => {
    expect(resolveDocumentTitle('Usage Records', 'My Site')).toBe(`Usage Records - ${VIDEO_GATEWAY_PRODUCT_NAME}`)
  })

  it('默认产品模式下，路由无标题时回退内部产品名', () => {
    expect(resolveDocumentTitle(undefined, 'My Site')).toBe(VIDEO_GATEWAY_PRODUCT_NAME)
  })

  it('默认产品模式下，站点名为空时仍使用内部产品名', () => {
    expect(resolveDocumentTitle('Dashboard', '')).toBe(`Dashboard - ${VIDEO_GATEWAY_PRODUCT_NAME}`)
    expect(resolveDocumentTitle(undefined, '   ')).toBe(VIDEO_GATEWAY_PRODUCT_NAME)
  })

  it('默认产品模式下，站点名仍是 upstream 默认值时回退内部产品名', () => {
    expect(resolveDocumentTitle('Login', 'Sub2API')).toBe(`Login - ${VIDEO_GATEWAY_PRODUCT_NAME}`)
    expect(resolveDocumentTitle(undefined, 'Sub2API')).toBe(VIDEO_GATEWAY_PRODUCT_NAME)
  })

  it('默认产品模式下，站点名变更不影响路由标题计算', () => {
    const before = resolveDocumentTitle('Admin Dashboard', 'Alpha')
    const after = resolveDocumentTitle('Admin Dashboard', 'Beta')

    expect(before).toBe(`Admin Dashboard - ${VIDEO_GATEWAY_PRODUCT_NAME}`)
    expect(after).toBe(`Admin Dashboard - ${VIDEO_GATEWAY_PRODUCT_NAME}`)
  })
})
