import { describe, expect, it } from 'vitest'
import { isVideoGatewayDemoRoute, VIDEO_GATEWAY_HOME_PATH } from '../productMode'

describe('productMode', () => {
  it('allows the internal generation content ledger in video gateway demo mode', () => {
    expect(isVideoGatewayDemoRoute('/admin/generation-content')).toBe(true)
  })

  it('keeps the video gateway dashboard as the demo home path', () => {
    expect(VIDEO_GATEWAY_HOME_PATH).toBe('/admin/video/dashboard')
  })
})
