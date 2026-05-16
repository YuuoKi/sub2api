export const VIDEO_GATEWAY_DEMO_MODE = 'video_gateway_demo'
export const VIDEO_GATEWAY_PRODUCT_NAME = '企业 AI 视频网关'
export const VIDEO_GATEWAY_ADMIN_NAME = '视频管理员'
export const VIDEO_GATEWAY_HOME_PATH = '/admin/video/dashboard'

export const isVideoGatewayDemoMode = import.meta.env.VITE_PRODUCT_MODE === VIDEO_GATEWAY_DEMO_MODE

const demoRoutePrefixes = [
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
    return provider || '模型通道'
  }

  if (displayName?.trim()) return displayName.trim()
  if (provider === 'seedance') return 'Seedance 2.0'
  if (provider === 'kling') return 'Kling'
  return provider || '模型通道'
}
