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
