import type {
  CreateOrderRequest,
  CreateOrderResult,
  MethodLimit,
  OrderType,
  WechatJSAPIPayload,
  WechatOAuthInfo,
} from '@/types/payment'

export const PAYMENT_RECOVERY_STORAGE_KEY = 'payment.recovery.current'
export const PAYMENT_SECRET_SESSION_KEY = 'payment.secret.session'

export interface PaymentSecretSession {
  orderId: number
  clientSecret: string
  intentId: string
  resumeToken: string
  createdAt: number
}

const VISIBLE_METHOD_ALIASES = {
  alipay: 'alipay',
  alipay_direct: 'alipay',
  wxpay: 'wxpay',
  wxpay_direct: 'wxpay',
  stripe: 'stripe',
  airwallex: 'airwallex',
} as const

export type VisiblePaymentMethod = 'alipay' | 'wxpay' | 'stripe' | 'airwallex'
export type StripeVisibleMethod = 'alipay' | 'wechat_pay'
export type PaymentLaunchKind =
  | 'qr_waiting'
  | 'redirect_waiting'
  | 'stripe_popup'
  | 'stripe_route'
  | 'airwallex_route'
  | 'wechat_oauth'
  | 'wechat_jsapi'
  | 'unhandled'

export interface PaymentRecoverySnapshot {
  orderId: number
  amount: number
  qrCode: string
  expiresAt: string
  paymentType: string
  payUrl: string
  outTradeNo: string
  clientSecret: string
  intentId: string
  currency: string
  countryCode: string
  paymentEnv: string
  payAmount: number
  orderType: OrderType | ''
  paymentMode: string
  resumeToken: string
  createdAt: number
}

export interface PaymentLaunchContext {
  visibleMethod: string
  orderType: OrderType
  isMobile: boolean
  isWechatBrowser?: boolean
  now?: number
  stripePopupUrl?: string
  stripeRouteUrl?: string
  airwallexRouteUrl?: string
}

export interface PaymentLaunchDecision {
  kind: PaymentLaunchKind
  paymentState: PaymentRecoverySnapshot
  recovery: PaymentRecoverySnapshot
  stripeMethod?: StripeVisibleMethod
  oauth?: WechatOAuthInfo
  jsapi?: WechatJSAPIPayload
}

export interface BuildCreateOrderPayloadInput {
  amount: number
  paymentType: string
  orderType: OrderType
  planId?: number
  origin?: string
  isMobile: boolean
  isWechatBrowser: boolean
}

type CreateOrderFlowResult = CreateOrderResult & {
  resume_token?: string
}

type StorageWriter = Pick<Storage, 'removeItem' | 'setItem'>

export function normalizeVisibleMethod(method: string): VisiblePaymentMethod | '' {
  const normalized = VISIBLE_METHOD_ALIASES[method.trim() as keyof typeof VISIBLE_METHOD_ALIASES]
  return normalized ?? ''
}

export function getVisibleMethods(methods: Record<string, MethodLimit>): Record<string, MethodLimit> {
  const visible: Record<string, MethodLimit> = {}

  Object.entries(methods).forEach(([type, limit]) => {
    const normalized = normalizeVisibleMethod(type)
    if (!normalized) return

    const isCanonical = type === normalized
    const existing = visible[normalized]
    if (!existing || isCanonical) {
      visible[normalized] = { ...limit }
    }
  })

  return visible
}

export function buildCreateOrderPayload(input: BuildCreateOrderPayloadInput): CreateOrderRequest {
  const visibleMethod = normalizeVisibleMethod(input.paymentType) || input.paymentType.trim()
  const normalizedOrigin = (input.origin || '').trim().replace(/\/+$/, '')
  const payload: CreateOrderRequest = {
    amount: input.amount,
    payment_type: visibleMethod,
    order_type: input.orderType,
    is_mobile: input.isMobile,
    payment_source: visibleMethod === 'wxpay' && input.isWechatBrowser
      ? 'wechat_in_app_resume'
      : 'hosted_redirect',
  }

  if (input.planId) {
    payload.plan_id = input.planId
  }
  if (normalizedOrigin) {
    payload.return_url = `${normalizedOrigin}/payment/result`
  }

  return payload
}

export function decidePaymentLaunch(
  result: CreateOrderFlowResult,
  context: PaymentLaunchContext,
): PaymentLaunchDecision {
  const visibleMethod = normalizeVisibleMethod(context.visibleMethod) || context.visibleMethod
  const baseState = createPaymentRecoverySnapshot({
    orderId: result.order_id,
    amount: result.amount,
    qrCode: result.qr_code || '',
    expiresAt: result.expires_at || '',
    paymentType: visibleMethod,
    payUrl: result.pay_url || '',
    outTradeNo: result.out_trade_no || '',
    clientSecret: result.client_secret || '',
    intentId: result.intent_id || '',
    currency: result.currency || '',
    countryCode: result.country_code || '',
    paymentEnv: result.payment_env || '',
    payAmount: result.pay_amount,
    orderType: context.orderType,
    paymentMode: (result.payment_mode || '').trim(),
    resumeToken: result.resume_token || '',
  }, context.now)

  if (visibleMethod === 'airwallex' && baseState.clientSecret && baseState.intentId) {
    if (!context.airwallexRouteUrl) {
      return { kind: 'unhandled', paymentState: baseState, recovery: baseState }
    }
    const paymentState = { ...baseState, payUrl: context.airwallexRouteUrl || '' }
    return { kind: 'airwallex_route', paymentState, recovery: paymentState }
  }

  if (baseState.clientSecret) {
    // visibleMethod === 'stripe' means the user clicked the dedicated Stripe button
    // and should land on the full Payment Element to choose a sub-method themselves.
    const isStripeButton = visibleMethod === 'stripe'
    const stripeMethod: StripeVisibleMethod | undefined = isStripeButton
      ? undefined
      : visibleMethod === 'wxpay' ? 'wechat_pay' : 'alipay'
    const kind: PaymentLaunchKind = stripeMethod === 'alipay' && !context.isMobile
      ? 'stripe_popup'
      : 'stripe_route'
    const payUrl = kind === 'stripe_popup'
      ? context.stripePopupUrl || context.stripeRouteUrl || ''
      : context.stripeRouteUrl || context.stripePopupUrl || ''
    const paymentState = { ...baseState, payUrl }
    return { kind, paymentState, recovery: paymentState, stripeMethod }
  }

  if (result.result_type === 'oauth_required' && result.oauth?.authorize_url) {
    return { kind: 'wechat_oauth', paymentState: baseState, recovery: baseState, oauth: result.oauth }
  }

  const jsapiPayload = result.jsapi ?? result.jsapi_payload
  if (result.result_type === 'jsapi_ready' && jsapiPayload) {
    return { kind: 'wechat_jsapi', paymentState: baseState, recovery: baseState, jsapi: jsapiPayload }
  }

  const normalizedPaymentMode = baseState.paymentMode.trim().toLowerCase()
  const prefersRedirect = normalizedPaymentMode === 'redirect'
    || normalizedPaymentMode === 'popup'
    || (context.isMobile && !!baseState.payUrl)
  const prefersQr = normalizedPaymentMode === 'qrcode'
    || normalizedPaymentMode === 'native'
    || (!prefersRedirect && !!baseState.qrCode)

  if (visibleMethod === 'wxpay' && context.isWechatBrowser && baseState.payUrl && !baseState.qrCode) {
    return { kind: 'redirect_waiting', paymentState: baseState, recovery: baseState }
  }

  if (prefersRedirect && baseState.payUrl) {
    return { kind: 'redirect_waiting', paymentState: baseState, recovery: baseState }
  }

  if (prefersQr && baseState.qrCode) {
    return { kind: 'qr_waiting', paymentState: baseState, recovery: baseState }
  }

  if (baseState.payUrl) {
    return { kind: 'redirect_waiting', paymentState: baseState, recovery: baseState }
  }

  return { kind: 'unhandled', paymentState: baseState, recovery: baseState }
}

export function createPaymentRecoverySnapshot(
  state: Omit<PaymentRecoverySnapshot, 'createdAt'>,
  now = Date.now(),
): PaymentRecoverySnapshot {
  return {
    ...state,
    createdAt: now,
  }
}

export function writePaymentSecretSession(
  sessionStorage: StorageWriter | undefined,
  snapshot: Pick<PaymentRecoverySnapshot, 'orderId' | 'clientSecret' | 'intentId' | 'resumeToken'>,
): void {
  const clientSecret = snapshot.clientSecret.trim()
  if (!sessionStorage || !clientSecret || snapshot.orderId <= 0) {
    return
  }
  const payload: PaymentSecretSession = {
    orderId: snapshot.orderId,
    clientSecret,
    intentId: snapshot.intentId || '',
    resumeToken: snapshot.resumeToken || '',
    createdAt: Date.now(),
  }
  sessionStorage.setItem(PAYMENT_SECRET_SESSION_KEY, JSON.stringify(payload))
}

export function readPaymentSecretSession(
  sessionStorage: Pick<Storage, 'getItem'> | undefined,
  orderId: number,
  resumeToken?: string,
): PaymentSecretSession | null {
  if (!sessionStorage || orderId <= 0) {
    return null
  }
  const raw = sessionStorage.getItem(PAYMENT_SECRET_SESSION_KEY)
  if (!raw) {
    return null
  }
  try {
    const parsed = JSON.parse(raw) as Partial<PaymentSecretSession>
    if (
      typeof parsed.orderId !== 'number'
      || typeof parsed.clientSecret !== 'string'
      || parsed.clientSecret.trim() === ''
      || typeof parsed.resumeToken !== 'string'
      || typeof parsed.createdAt !== 'number'
    ) {
      return null
    }
    if (parsed.orderId !== orderId) {
      return null
    }
    if (resumeToken && parsed.resumeToken !== resumeToken) {
      return null
    }
    return {
      orderId: parsed.orderId,
      clientSecret: parsed.clientSecret,
      intentId: typeof parsed.intentId === 'string' ? parsed.intentId : '',
      resumeToken: parsed.resumeToken,
      createdAt: parsed.createdAt,
    }
  } catch {
    return null
  }
}

export function clearPaymentSecretSession(sessionStorage: Pick<Storage, 'removeItem'> | undefined): void {
  sessionStorage?.removeItem(PAYMENT_SECRET_SESSION_KEY)
}

export function writePaymentRecoverySnapshot(
  storage: StorageWriter,
  snapshot: PaymentRecoverySnapshot,
  key = PAYMENT_RECOVERY_STORAGE_KEY,
  sessionStorage?: StorageWriter,
): void {
  writePaymentSecretSession(sessionStorage, snapshot)
  const redacted: PaymentRecoverySnapshot = {
    ...snapshot,
    clientSecret: '',
  }
  storage.setItem(key, JSON.stringify(redacted))
}

export function clearPaymentRecoverySnapshot(
  storage: Pick<Storage, 'removeItem'>,
  key = PAYMENT_RECOVERY_STORAGE_KEY,
  sessionStorage?: Pick<Storage, 'removeItem'>,
): void {
  storage.removeItem(key)
  clearPaymentSecretSession(sessionStorage)
}

export function readPaymentRecoverySnapshot(
  raw: string | null | undefined,
  options: { now?: number; resumeToken?: string; sessionStorage?: Pick<Storage, 'getItem'> } = {},
): PaymentRecoverySnapshot | null {
  if (!raw) return null

  try {
    const parsed = JSON.parse(raw) as Partial<PaymentRecoverySnapshot>
    if (
      typeof parsed.orderId !== 'number'
      || typeof parsed.amount !== 'number'
      || typeof parsed.qrCode !== 'string'
      || typeof parsed.expiresAt !== 'string'
      || typeof parsed.paymentType !== 'string'
      || typeof parsed.payUrl !== 'string'
      || (parsed.outTradeNo != null && typeof parsed.outTradeNo !== 'string')
      || typeof parsed.clientSecret !== 'string'
      || (parsed.intentId != null && typeof parsed.intentId !== 'string')
      || (parsed.currency != null && typeof parsed.currency !== 'string')
      || (parsed.countryCode != null && typeof parsed.countryCode !== 'string')
      || (parsed.paymentEnv != null && typeof parsed.paymentEnv !== 'string')
      || typeof parsed.payAmount !== 'number'
      || typeof parsed.paymentMode !== 'string'
      || typeof parsed.resumeToken !== 'string'
      || typeof parsed.createdAt !== 'number'
    ) {
      return null
    }

    const now = options.now ?? Date.now()
    const expiresAt = Date.parse(parsed.expiresAt)
    if (Number.isFinite(expiresAt) && expiresAt <= now) {
      return null
    }
    if (options.resumeToken && parsed.resumeToken !== options.resumeToken) {
      return null
    }

    const secretSession = readPaymentSecretSession(
      options.sessionStorage,
      parsed.orderId,
      options.resumeToken,
    )

    return {
      orderId: parsed.orderId,
      amount: parsed.amount,
      qrCode: parsed.qrCode,
      expiresAt: parsed.expiresAt,
      paymentType: parsed.paymentType,
      payUrl: parsed.payUrl,
      outTradeNo: parsed.outTradeNo || '',
      clientSecret: secretSession?.clientSecret || '',
      intentId: secretSession?.intentId || parsed.intentId || '',
      currency: parsed.currency || '',
      countryCode: parsed.countryCode || '',
      paymentEnv: parsed.paymentEnv || '',
      payAmount: parsed.payAmount,
      orderType: parsed.orderType === 'subscription' ? 'subscription' : 'balance',
      paymentMode: parsed.paymentMode,
      resumeToken: parsed.resumeToken,
      createdAt: parsed.createdAt,
    }
  } catch {
    return null
  }
}
