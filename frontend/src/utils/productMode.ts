export const DEFAULT_PRODUCT_NAME = '无界 · 企业 AI 管理中台'
export const UPSTREAM_DEFAULT_PRODUCT_NAME = 'Sub2API'

/**
 * 保留管理员自定义站点名；仅将空值或上游默认名归一为无界品牌。
 * 本模块只负责品牌语义，不承载路由、导航或功能隐藏模式。
 */
export function resolveProductName(siteName?: string): string {
  const normalizedSiteName = siteName?.trim()

  if (normalizedSiteName && normalizedSiteName !== UPSTREAM_DEFAULT_PRODUCT_NAME) {
    return normalizedSiteName
  }

  return DEFAULT_PRODUCT_NAME
}
