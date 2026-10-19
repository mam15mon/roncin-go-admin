import { describe, expect, it } from 'vitest';
import { notificationActionUrl } from './notification-test-action';

const uuid = '12345678-1234-4234-8234-123456789abc';
const origin = 'https://configured.example.test';

describe('通知动作目标边界', () => {
  it.each([
    '/welcome',
    '/orders',
    '/finance/exchange-rates',
    `/orders/sea-export/${uuid}/fees`,
    `/commission-adjustments/${uuid}/my-supplement-source`,
  ])('仅接受服务端固定业务路由 %s', (path) => {
    expect(notificationActionUrl('查看详情', `${origin}${path}`)).toBe(
      `${origin}${path}`,
    );
  });

  it('允许服务端配置的HTTP开发站点，不把浏览器origin作为第二套配置', () => {
    expect(
      notificationActionUrl('打开系统首页', 'http://localhost:8000/welcome'),
    ).toBe('http://localhost:8000/welcome');
  });

  it.each([
    'https://Admin.Example.Com/welcome',
    'https://configured.example.test:443/welcome',
    'https://Admin.Example.Com:443/welcome',
    'http://Admin.Example.Com:80/welcome',
  ])('接受合法配置站点的大小写和默认端口 %s', (url) => {
    expect(notificationActionUrl('打开系统首页', url)).toBe(url);
  });

  it.each([
    'javascript:alert(1)',
    'data:text/html,<b>test</b>',
    '//configured.example.test/welcome',
    '/welcome',
    'https://user:password@configured.example.test/welcome',
    'https://@configured.example.test/welcome',
    'https:///configured.example.test/welcome',
    `${origin}/welcome?redirect=https://evil.test`,
    `${origin}/welcome#target`,
    `${origin}/welcome?`,
    `${origin}/welcome#`,
    `${origin}/welcome/`,
    `${origin}/orders/../welcome`,
    `${origin}/orders/%2e%2e/welcome`,
    'https://Admin.Example.Com:443/orders/../welcome',
    'https://Admin.Example.Com:443/orders/%2e%2e/welcome',
    'https://Admin.Example.Com:443/%77elcome',
    `${origin}/%77elcome`,
    `${origin}/orders/sea-export/not-a-uuid/fees`,
    `${origin}/orders/sea-export/${uuid}/fees/extra`,
    `${origin}/commission-adjustments/${uuid}/other`,
    `${origin}/admin/users`,
    `${origin}/welcome\n`,
    'https://configured.example.test\\@evil.test/welcome',
  ])('拒绝非法或非固定路径 %s', (url) => {
    expect(notificationActionUrl('查看详情', url)).toBe('');
  });

  it.each([
    [undefined, undefined],
    ['打开系统首页', undefined],
    ['', `${origin}/welcome`],
    [' ', `${origin}/welcome`],
    [undefined, `${origin}/welcome`],
  ])('动作文案与目标必须成对存在', (title, url) => {
    expect(notificationActionUrl(title, url)).toBe('');
  });
});
