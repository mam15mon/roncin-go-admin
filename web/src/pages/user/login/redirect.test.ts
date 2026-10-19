import { describe, expect, it } from 'vitest';
import { safeLoginRedirect } from './redirect';

describe('登录站内回跳', () => {
  it('保留站内业务路径、查询和锚点', () => {
    expect(safeLoginRedirect('/orders/123?tab=fees#cost')).toBe(
      '/orders/123?tab=fees#cost',
    );
  });
  it.each([
    null,
    '',
    'https://evil.example/a',
    '//evil.example',
    '/\\evil.example',
    '/%5Cevil.example',
    '/user/login',
    '/user/login/?redirect=%2Forders',
    '/user/login/dingtalk/callback?authCode=secret',
    '/USER/LOGIN',
    '/user/%6cogin',
    '/login',
    '/a/../user/login',
    '/%E0%A4%A',
  ])('拒绝外跳或登录循环 %s', (value) => {
    expect(safeLoginRedirect(value)).toBe('/');
  });
});
