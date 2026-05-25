export const VIDEO_GATEWAY_DEMO_MODE = 'video_gateway_demo'
export const PUBLIC_AUTH_PRODUCT_NAME = '企业 AI 视频 API 调度中台'
export const PUBLIC_AUTH_PRODUCT_SUBTITLE = 'AI 中剧 / AI 短剧生产网关'
export const PUBLIC_AUTH_NETWORK_LABEL = 'LAN-only 内部试运行'
export const PUBLIC_AUTH_SAFE_DEMO_LABEL = 'Safe demo / mock only'
export const PUBLIC_AUTH_PRODUCTION_STATUS = 'Production NOT_READY'
export const PUBLIC_AUTH_COMMERCIAL_STATUS = 'Commercial NOT_READY'
export const PUBLIC_AUTH_REAL_PROVIDER_STATUS = 'Real Provider NOT_READY'
export const VIDEO_GATEWAY_PRODUCT_NAME = PUBLIC_AUTH_PRODUCT_NAME
export const VIDEO_GATEWAY_ADMIN_NAME = '平台管理员'
export const VIDEO_GATEWAY_HOME_PATH = '/admin/video/dashboard'

export const isVideoGatewayDemoMode = import.meta.env.VITE_PRODUCT_MODE === VIDEO_GATEWAY_DEMO_MODE

const demoRoutePrefixes = [
  '/internal-pilot',
  '/admin/video',
  '/login',
  '/setup',
  '/auth',
  '/payment/result',
]

export function isVideoGatewayDemoRoute(path: string): boolean {
  return path === '/' || path === '/admin' || demoRoutePrefixes.some((prefix) => path === prefix || path.startsWith(`${prefix}/`))
}

export function videoGatewayDisplayProvider(provider: string, displayName?: string): string {
  if (isVideoGatewayDemoMode) {
    if (provider === 'mock') return '演示通道'
    if (provider === 'seedance') return 'Seedance 2.0'
    if (provider === 'kling') return 'Kling'
    return provider || 'API 通道'
  }

  if (displayName?.trim()) return displayName.trim()
  if (provider === 'seedance') return 'Seedance 2.0'
  if (provider === 'kling') return 'Kling'
  return provider || '模型通道'
}
