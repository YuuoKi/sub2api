export const buildQCanvasProjectsURL = (configuredBaseURL?: string): string => {
  const baseURL = configuredBaseURL?.trim()
  if (!baseURL) {
    throw new Error('QCanvas 地址未配置（请设置 VITE_QCANVAS_BASE_URL）')
  }

  const url = new URL(baseURL)

  if (url.protocol !== 'http:' && url.protocol !== 'https:') {
    throw new Error('QCanvas 地址必须使用 HTTP 或 HTTPS')
  }
  if (url.username || url.password) {
    throw new Error('QCanvas 地址不得包含用户名或密码')
  }
  if (url.pathname !== '/' || url.search || url.hash) {
    throw new Error('QCanvas 地址必须是纯 origin，不得包含路径、查询或片段')
  }

  return new URL('/projects', url.origin).toString()
}

export const tryBuildQCanvasProjectsURL = (configuredBaseURL?: string): string | null => {
  try {
    return buildQCanvasProjectsURL(configuredBaseURL)
  } catch {
    return null
  }
}
