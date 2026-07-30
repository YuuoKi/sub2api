/**
 * Client-side idempotency token for one UI open/submit session.
 *
 * `crypto.randomUUID()` is secure-context-only in Chromium. Sub2 admin is
 * intentionally served on plain HTTP (`http://IP`), so that API is often
 * missing and must not crash open-handlers.
 *
 * @param cryptoApi - Optional Crypto (defaults to globalThis.crypto; pass a
 *   window's crypto when minting for a cross-window handshake).
 */
export function createIdempotencyKey(cryptoApi: Crypto = globalThis.crypto): string {
  if (typeof cryptoApi?.randomUUID === 'function') {
    return cryptoApi.randomUUID()
  }
  if (typeof cryptoApi?.getRandomValues === 'function') {
    const bytes = new Uint8Array(16)
    cryptoApi.getRandomValues(bytes)
    // RFC 4122 version 4
    bytes[6] = (bytes[6] & 0x0f) | 0x40
    bytes[8] = (bytes[8] & 0x3f) | 0x80
    const hex = Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('')
    return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`
  }
  throw new Error('crypto.getRandomValues unavailable; cannot mint idempotency key')
}
