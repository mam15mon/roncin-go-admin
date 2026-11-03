import { createHash } from 'node:crypto';
import { describe, expect, it } from 'vitest';
import { computeCanonicalSha256, sha256, sha256Bytes } from './hash';

describe('SHA-256 字节与既有字符串契约', () => {
  it.each([0, 1, 55, 56, 63, 64, 65, 127, 128, 1024 * 1024])(
    '对 %i 字节二进制与标准实现一致',
    (size) => {
      const bytes = Uint8Array.from(
        { length: size },
        (_, i) => (i * 19 + 255) % 256,
      );
      expect(sha256Bytes(bytes)).toBe(
        createHash('sha256').update(bytes).digest('hex'),
      );
    },
  );
  it.each(['', 'abc', '订单附件📎', 'a'.repeat(128)])(
    '保持 UTF-8 字符串哈希：%s',
    (input) => {
      expect(sha256(input)).toBe(
        createHash('sha256').update(input, 'utf8').digest('hex'),
      );
    },
  );
  it('保持拒绝不完整 UTF-16 输入的既有行为', () => {
    expect(() => sha256('\ud800')).toThrow(URIError);
  });
  it('保持规范 JSON 排序与忽略 undefined 的既有行为', () => {
    expect(
      computeCanonicalSha256({ b: '订单', a: [1, null], c: undefined }),
    ).toBe(
      createHash('sha256').update('{"a":[1,null],"b":"订单"}').digest('hex'),
    );
  });
});
