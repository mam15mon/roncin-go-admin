// @vitest-environment node
import type { IncomingMessage, ServerResponse } from 'node:http';
import { fileURLToPath } from 'node:url';
import type { Connect, ViteDevServer } from 'vite';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { dingTalkLoginBootstrap } from './dingtalkLoginBootstrap';

function setup(mode = 'dev') {
  let middleware: Connect.NextHandleFunction | undefined;
  const transformIndexHtml = vi.fn(async (_url: string, html: string) =>
    html.replace('</head>', '<script src="/@vite/client"></script></head>'),
  );
  const server = {
    config: {
      root: fileURLToPath(new URL('..', import.meta.url)),
      mode,
    },
    middlewares: {
      use: (handler: Connect.NextHandleFunction) => {
        middleware = handler;
      },
    },
    transformIndexHtml,
  } as unknown as ViteDevServer;
  const plugin = dingTalkLoginBootstrap();
  if (typeof plugin.configureServer !== 'function') {
    throw new Error('缺少开发服务器配置');
  }
  plugin.configureServer.call({} as never, server);

  async function request(
    url = '/user/login',
    method = 'GET',
    headers: IncomingMessage['headers'] = { accept: 'text/html' },
  ) {
    const response = {
      setHeader: vi.fn(),
      end: vi.fn(),
      statusCode: 0,
    };
    const next = vi.fn();
    await middleware?.(
      { url, method, headers, originalUrl: url } as unknown as IncomingMessage,
      response as unknown as ServerResponse,
      next,
    );
    return { response, next };
  }
  return { request, transformIndexHtml, plugin };
}

afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});

describe('开发登录页 HTML 引导', () => {
  it.each(['DingTalk/7.0', 'Mozilla/5.0 dInGtAlK PC'])(
    '钉钉 UA %s 不获取扫码配置且登录 HTML 仍禁止缓存',
    async (userAgent) => {
      const fetchConfig = vi.fn();
      vi.stubGlobal('fetch', fetchConfig);
      const { response, next } = await setup().request(
        '/user/login?redirect=%2Forders',
        'GET',
        {
          host: 'localhost:8001',
          'user-agent': userAgent,
          accept: 'text/html',
        },
      );
      const html = response.end.mock.calls[0][0] as string;
      expect(fetchConfig).not.toHaveBeenCalled();
      expect(html).not.toContain('roncin-dingtalk-login-config');
      expect(html).not.toContain('<iframe');
      expect(html).toContain('/@vite/client');
      expect(response.setHeader).toHaveBeenCalledWith(
        'Cache-Control',
        'no-store',
      );
      expect(response.setHeader).not.toHaveBeenCalledWith(
        'Set-Cookie',
        expect.anything(),
      );
      expect(next).not.toHaveBeenCalled();
    },
  );
  it('同一响应注入配置并原样透传多条Cookie，仍经过Vite转换且禁止缓存', async () => {
    const payload = { success: true, data: { enabled: false } };
    const cookies = [
      'state=one; Path=/; HttpOnly; SameSite=Lax',
      'other=two; Expires=Thu, 01 Jan 2037 00:00:00 GMT; Secure; Path=/',
    ];
    const fetchConfig = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => payload,
      headers: { getSetCookie: () => cookies },
    });
    vi.stubGlobal('fetch', fetchConfig);
    const { request, transformIndexHtml, plugin } = setup();
    const { response, next } = await request('/user/login?redirect=%2Fwelcome');
    const html = response.end.mock.calls[0][0] as string;
    expect(html).toContain(
      `<script type="application/json" id="roncin-dingtalk-login-config">${JSON.stringify(payload)}</script>`,
    );
    expect(html).toContain('/@vite/client');
    expect(transformIndexHtml).toHaveBeenCalledWith(
      '/user/login?redirect=%2Fwelcome',
      expect.any(String),
      '/user/login?redirect=%2Fwelcome',
    );
    expect(response.setHeader).toHaveBeenCalledWith('Set-Cookie', cookies);
    expect(response.setHeader).toHaveBeenCalledWith(
      'Cache-Control',
      'no-store',
    );
    expect(response.setHeader).toHaveBeenCalledWith('Pragma', 'no-cache');
    expect(response.statusCode).toBe(200);
    expect(next).not.toHaveBeenCalled();
    expect(plugin.apply).toBe('serve');
    expect(fetchConfig.mock.calls[0][0].href).toBe(
      'http://127.0.0.1:8000/api/v1/auth/dingtalk/login-config',
    );
  });

  it('不处理HEAD、回调、业务页、资源或非HTML请求', async () => {
    const fetchConfig = vi.fn();
    vi.stubGlobal('fetch', fetchConfig);
    const { request } = setup();
    for (const [url, method, accept] of [
      ['/user/login', 'HEAD', 'text/html'],
      ['/user/login', 'POST', 'text/html'],
      ['/user/login/dingtalk/callback', 'GET', 'text/html'],
      ['/user/login/', 'GET', 'text/html'],
      ['/welcome', 'GET', 'text/html'],
      ['/favicon.ico', 'GET', '*/*'],
      ['/api/v1/auth/me', 'GET', 'application/json'],
      ['/user/login', 'GET', 'application/json'],
    ]) {
      const { response, next } = await request(url, method, { accept });
      expect(next).toHaveBeenCalledOnce();
      expect(response.end).not.toHaveBeenCalled();
    }
    expect(fetchConfig).not.toHaveBeenCalled();
  });

  it('使用当前远端代理目标且不把浏览器缓存条件传给配置接口', async () => {
    vi.stubEnv(
      'RONCIN_API_PROXY_TARGET',
      'https://pre.example.com/backend?routing=preview',
    );
    const fetchConfig = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ success: true, data: {} }),
      headers: { getSetCookie: () => [] },
    });
    vi.stubGlobal('fetch', fetchConfig);
    const { request } = setup('pre');
    const { response } = await request('/user/login', 'GET', {
      host: 'localhost:8001',
      'if-none-match': 'old-etag',
      'if-modified-since': 'Thu, 01 Jan 2037 00:00:00 GMT',
    });
    expect(fetchConfig.mock.calls[0][0].href).toBe(
      'https://pre.example.com/backend/api/v1/auth/dingtalk/login-config?routing=preview',
    );
    expect(fetchConfig.mock.calls[0][1].headers).toBeUndefined();
    expect(response.statusCode).toBe(200);
    expect(response.setHeader).not.toHaveBeenCalledWith(
      'Set-Cookie',
      expect.anything(),
    );
  });

  it('将HTML闭合和分隔字符转义为不可执行JSON', async () => {
    const message = '</script><script>alert(1)</script>&>$&\u2028\u2029';
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({ success: false, message }),
        headers: { getSetCookie: () => [] },
      }),
    );
    const { response } = await setup().request();
    const html = response.end.mock.calls[0][0] as string;
    const match = html.match(
      /id="roncin-dingtalk-login-config">(.*?)<\/script>/,
    );
    expect(match).not.toBeNull();
    expect(match?.[1]).not.toContain('<');
    expect(match?.[1]).not.toContain('&');
    expect(JSON.parse(match?.[1] ?? '').message).toBe(message);
    expect(html).not.toContain('<script>alert(1)</script>');
  });

  it.each(['network', 'status', 'json'])(
    '%s失败随HTML返回固定错误且不泄露内部异常',
    async (kind) => {
      vi.stubGlobal(
        'fetch',
        vi.fn(async () => {
          if (kind === 'network') throw new Error('internal-secret');
          return {
            ok: kind !== 'status',
            json: async () => {
              throw new Error('internal-secret');
            },
          };
        }),
      );
      const { response } = await setup().request();
      const html = response.end.mock.calls[0][0] as string;
      expect(html).toContain('钉钉登录配置加载失败，请稍后重试');
      expect(html).toContain('"success":false');
      expect(html).not.toContain('internal-secret');
      expect(html).toContain('id="root"');
      expect(response.setHeader).not.toHaveBeenCalledWith(
        'Set-Cookie',
        expect.anything(),
      );
    },
  );
});

describe('开发 HTML 提前加载扫码实例', () => {
  function payload() {
    return {
      success: true,
      data: {
        enabled: true,
        authorizeUrl: `https://login.dingtalk.com/oauth2/auth?state=sample%3C%26%22state&client_id=app&scope=openid&response_type=code&redirect_uri=${encodeURIComponent('http://localhost:8001/user/login/dingtalk/callback')}`,
        expiresAt: '2037-01-01T00:00:00Z',
      },
    };
  }

  async function html(value: unknown) {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => value,
        headers: {
          getSetCookie: () => ['state=same-response; HttpOnly; Path=/'],
        },
      }),
    );
    const { response } = await setup().request('/user/login', 'GET', {
      accept: 'text/html',
      host: 'localhost:8001',
    });
    return { body: response.end.mock.calls[0][0] as string, response };
  }

  it('iframe 位于 root 外并隐藏，URL 属性安全转义且保留导航事件记录', async () => {
    const { body, response } = await html(payload());
    const frame = body.slice(body.indexOf('<iframe'));
    expect(body.indexOf('<iframe')).toBeGreaterThan(
      body.indexOf('<div id="root"></div>'),
    );
    expect(frame).toContain('id="roncin-dingtalk-login-frame"');
    expect(frame).toContain('visibility:hidden');
    expect(frame).toContain('iframe=true');
    expect(frame).toContain('&amp;');
    expect(frame).toContain('data-expires-at="2114380800000"');
    expect(frame).toContain('onload="this.dataset.navigation=\'loaded\'"');
    expect(frame).not.toContain('sample<&"state');
    expect(response.setHeader).toHaveBeenCalledWith('Set-Cookie', [
      'state=same-response; HttpOnly; Path=/',
    ]);
  });

  it.each([
    'disabled',
    'failure',
    'expired',
    'origin',
    'callback',
    'missing-client',
    'invalid-date',
  ])('%s 配置不提前发授权请求', async (kind) => {
    const value = payload();
    const authorize = new URL(value.data.authorizeUrl);
    if (kind === 'disabled') value.data.enabled = false;
    if (kind === 'failure') value.success = false;
    if (kind === 'expired') value.data.expiresAt = '2000-01-01T00:00:00Z';
    if (kind === 'invalid-date') value.data.expiresAt = 'invalid';
    if (kind === 'origin') authorize.hostname = 'example.net';
    if (kind === 'callback')
      authorize.searchParams.set(
        'redirect_uri',
        'http://evil.example/user/login/dingtalk/callback',
      );
    if (kind === 'missing-client') authorize.searchParams.delete('client_id');
    value.data.authorizeUrl = authorize.href;
    expect((await html(value)).body).not.toContain('<iframe');
  });
});
