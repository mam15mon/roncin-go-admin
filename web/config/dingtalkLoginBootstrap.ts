import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import type { Plugin } from 'vite';
import { getProxyConfig } from './proxy';

const failure = {
  success: false,
  code: 500,
  message: '钉钉登录配置加载失败，请稍后重试',
};

// 仅开发服务器使用；构建产物由 Go 在每次登录页导航时生成独立配置与 Cookie。
export function dingTalkLoginBootstrap(): Plugin {
  return {
    name: 'roncin-dingtalk-login-bootstrap',
    apply: 'serve',
    configureServer(server) {
      const proxy = getProxyConfig(server.config.mode)['/api/'];
      const configURL = new URL(proxy.target);
      // 与开发代理默认 prependPath 一致，保留测试/预发目标可能自带的路径。
      configURL.pathname = `${configURL.pathname.replace(/\/$/, '')}/api/v1/auth/dingtalk/login-config`;
      server.middlewares.use(async (request, response, next) => {
        const pathname = request.url?.split('?')[0];
        if (
          request.method !== 'GET' ||
          pathname !== '/user/login' ||
          (request.headers.accept &&
            !request.headers.accept.includes('text/html'))
        ) {
          next();
          return;
        }

        let payload: unknown = failure;
        let cookies: string[] = [];
        try {
          const configResponse = await fetch(configURL, {
            headers:
              !proxy.changeOrigin && request.headers.host
                ? { host: request.headers.host }
                : undefined,
            signal: AbortSignal.timeout(5000),
          });
          if (!configResponse.ok) throw new Error('配置接口请求失败');
          payload = await configResponse.json();
          cookies = configResponse.headers.getSetCookie();
        } catch {
          // 错误也随 HTML 显式传递，浏览器不再发第二次请求覆盖本次校验状态。
          payload = failure;
        }

        try {
          const serialized = JSON.stringify(payload)
            .replace(/</g, '\\u003c')
            .replace(/>/g, '\\u003e')
            .replace(/&/g, '\\u0026')
            .replace(/\u2028/g, '\\u2028')
            .replace(/\u2029/g, '\\u2029');
          const template = await readFile(
            resolve(server.config.root, 'index.html'),
            'utf8',
          );
          const html = await server.transformIndexHtml(
            request.url ?? '/user/login',
            template.replace(
              '</head>',
              () =>
                `<script type="application/json" id="roncin-dingtalk-login-config">${serialized}</script></head>`,
            ),
            request.originalUrl,
          );
          response.statusCode = 200;
          response.setHeader('Content-Type', 'text/html; charset=utf-8');
          response.setHeader('Cache-Control', 'no-store');
          response.setHeader('Pragma', 'no-cache');
          if (cookies.length) response.setHeader('Set-Cookie', cookies);
          response.end(html);
        } catch (error) {
          next(error);
        }
      });
    },
  };
}
