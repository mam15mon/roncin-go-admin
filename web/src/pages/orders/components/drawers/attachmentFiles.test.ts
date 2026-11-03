import { createHash } from 'node:crypto';
import { describe, expect, it } from 'vitest';
import {
  attachmentChecksum,
  attachmentPreviewType,
  attachmentSize,
} from './attachmentFiles';

describe('附件文件边界', () => {
  it('允许静态图片/PDF预览，拒绝 HTML/SVG 和非预览格式', () => {
    expect(attachmentPreviewType('image/png')).toBe('image');
    expect(attachmentPreviewType('application/pdf')).toBe('pdf');
    for (const type of [
      'image/svg+xml',
      'text/html',
      'application/octet-stream',
      'application/vnd.ms-excel',
      undefined,
    ]) {
      expect(attachmentPreviewType(type)).toBeUndefined();
    }
  });
  it('没有 Web Crypto 的 HTTP 页面仍可计算真实二进制校验值', async () => {
    const bytes = new Uint8Array([0, 128, 255, 37, 0]);
    expect(await attachmentChecksum(new Blob([bytes]))).toBe(
      createHash('sha256').update(bytes).digest('hex'),
    );
  });
  it('文件大小使用可读单位', () => {
    expect(attachmentSize('1024')).toBe('1.0 KB');
    expect(attachmentSize(1048576)).toBe('1.00 MB');
    expect(attachmentSize(undefined)).toBe('-');
  });
});
