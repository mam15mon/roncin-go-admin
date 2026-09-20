import { afterEach, describe, expect, it, vi } from 'vitest';

const sentryMock = vi.hoisted(() => ({
  init: vi.fn(),
}));

vi.mock('@sentry/react', () => ({
  init: sentryMock.init,
}));

describe('sentry 轻量接入', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    sentryMock.init.mockClear();
    vi.resetModules();
  });

  it('未配置 VITE_SENTRY_DSN 时完全不初始化（零网络行为）', async () => {
    vi.stubEnv('VITE_SENTRY_DSN', '');
    await import('./sentry');
    expect(sentryMock.init).not.toHaveBeenCalled();
  });

  it('配置 DSN 后初始化，release 使用提交哈希，不开启 tracing/Replay/PII', async () => {
    vi.stubEnv('VITE_SENTRY_DSN', 'https://public-key@example.com/1');
    vi.stubEnv('VITE_COMMIT_HASH', 'abc1234');
    vi.stubEnv('MODE', 'test');
    await import('./sentry');

    expect(sentryMock.init).toHaveBeenCalledTimes(1);
    const options = sentryMock.init.mock.calls[0][0] as Record<string, unknown>;
    expect(options.dsn).toBe('https://public-key@example.com/1');
    expect(options.release).toBe('abc1234');
    expect(options.environment).toBe('test');
    expect(options.tracesSampleRate).toBeUndefined();
    expect(options.replaysSessionSampleRate).toBeUndefined();
    expect(options.sendDefaultPii).toBeUndefined();
  });

  it('生产构建（MODE=production）时 environment 为 production', async () => {
    vi.stubEnv('VITE_SENTRY_DSN', 'https://public-key@example.com/1');
    vi.stubEnv('MODE', 'production');
    await import('./sentry');

    const options = sentryMock.init.mock.calls[0][0] as Record<string, unknown>;
    expect(options.environment).toBe('production');
  });

  it('裸 vite 开发模式（MODE=development）时 environment 为 development', async () => {
    vi.stubEnv('VITE_SENTRY_DSN', 'https://public-key@example.com/1');
    vi.stubEnv('MODE', 'development');
    await import('./sentry');

    const options = sentryMock.init.mock.calls[0][0] as Record<string, unknown>;
    expect(options.environment).toBe('development');
  });

  it('beforeSend 双保险剔除 Authorization/Cookie 请求头', async () => {
    vi.stubEnv('VITE_SENTRY_DSN', 'https://public-key@example.com/1');
    await import('./sentry');

    const options = sentryMock.init.mock.calls[0][0] as {
      beforeSend?: (event: unknown) => unknown;
    };
    const event = {
      request: {
        headers: {
          Authorization: 'Bearer secret-token',
          Cookie: 'session=secret',
          'X-Request-ID': 'keep-me',
        },
      },
    };
    const result = options.beforeSend?.(event) as typeof event;
    expect(result.request.headers.Authorization).toBeUndefined();
    expect(result.request.headers.Cookie).toBeUndefined();
    expect(result.request.headers['X-Request-ID']).toBe('keep-me');
  });
});
