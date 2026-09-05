/**
 * 生成符合 RFC 4122 v4 标准的 UUID。
 *
 * 兼容在非安全上下文（例如通过局域网 IP / 公网 IP 经纯 HTTP 访问）下
 * 现代浏览器禁用 crypto.randomUUID 的环境。
 */
export function generateUUID(): string {
  // 1. 优先使用原生 randomUUID（安全上下文、HTTPS、localhost 环境）
  if (
    typeof globalThis !== 'undefined' &&
    globalThis.crypto &&
    typeof globalThis.crypto.randomUUID === 'function'
  ) {
    return globalThis.crypto.randomUUID();
  }

  // 2. 降级使用 crypto.getRandomValues（部分非安全上下文中仍然提供）
  if (
    typeof globalThis !== 'undefined' &&
    globalThis.crypto &&
    typeof globalThis.crypto.getRandomValues === 'function'
  ) {
    const bytes = new Uint8Array(16);
    globalThis.crypto.getRandomValues(bytes);
    // RFC 4122 v4: version 4 (0100) & variant RFC 4122 (10xx)
    bytes[6] = (bytes[6] & 0x0f) | 0x40;
    bytes[8] = (bytes[8] & 0x3f) | 0x80;
    const hex = Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('');
    return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20, 32)}`;
  }

  // 3. 兜底使用 Math.random
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}
