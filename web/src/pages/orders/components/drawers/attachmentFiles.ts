import { sha256Bytes } from '@/utils/hash';

export const MAX_ATTACHMENT_SIZE = 100 * 1024 * 1024;

/** 浏览器可安全显示的静态格式；不按后缀放行 HTML、SVG 等活动内容。 */
export function attachmentPreviewType(
  mimeType?: string,
): 'image' | 'pdf' | undefined {
  if (mimeType === 'application/pdf') return 'pdf';
  if (
    [
      'image/png',
      'image/jpeg',
      'image/gif',
      'image/webp',
      'image/bmp',
      'image/avif',
    ].includes(mimeType ?? '')
  ) {
    return 'image';
  }
  return undefined;
}

export function attachmentSize(value?: string | number): string {
  const size = Number(value);
  if (!Number.isFinite(size) || size < 0) return '-';
  if (size >= 1024 * 1024) return `${(size / (1024 * 1024)).toFixed(2)} MB`;
  if (size >= 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${size} 字节`;
}

export async function attachmentChecksum(file: Blob): Promise<string> {
  // 本地局域网 HTTP 入口没有 Web Crypto；统一采用相同字节算法。
  return sha256Bytes(new Uint8Array(await file.arrayBuffer()));
}
