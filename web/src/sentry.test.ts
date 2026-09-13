import { afterEach, describe, expect, it, vi } from 'vitest';

const sentryMock = vi.hoisted(() => ({
  init: vi.fn(),
}));

vi.mock('@sentry/react', () => ({
  init: sentryMock.init,
}));

describe('sentry 轻量接入', () => {
  const originalDsn = process.env.SENTRY_DSN;
  const originalUmiEnv = process.env.UMI_ENV;
  const originalCommitHash = process.env.COMMIT_HASH;

  afterEach(() => {
    process.env.SENTRY_DSN = originalDsn;
    process.env.UMI_ENV = originalUmiEnv;
    process.env.COMMIT_HASH = originalCommitHash;
    sentryMock.init.mockClear();
    vi.resetModules();
  });

  it('未配置 SENTRY_DSN 时完全不初始化（零网络行为）', async () => {
    delete process.env.SENTRY_DSN;
    vi.resetModules();
    await import('./sentry');
    expect(sentryMock.init).not.toHaveBeenCalled();
  });

  it('配置 DSN 后初始化，release 使用 COMMIT_HASH，不开启 tracing/Replay/PII', async () => {
    process.env.SENTRY_DSN = 'https://public-key@example.com/1';
    process.env.COMMIT_HASH = 'abc1234';
    delete process.env.UMI_ENV;
    vi.resetModules();
    await import('./sentry');

    expect(sentryMock.init).toHaveBeenCalledTimes(1);
    const options = sentryMock.init.mock.calls[0][0] as Record<string, unknown>;
    expect(options.dsn).toBe('https://public-key@example.com/1');
    expect(options.release).toBe('abc1234');
    expect(options.environment).toBe('development');
    expect(options.tracesSampleRate).toBeUndefined();
    expect(options.replaysSessionSampleRate).toBeUndefined();
    expect(options.sendDefaultPii).toBeUndefined();
  });

  it('UMI_ENV=prod 时 environment 为 production', async () => {
    process.env.SENTRY_DSN = 'https://public-key@example.com/1';
    process.env.UMI_ENV = 'prod';
    vi.resetModules();
    await import('./sentry');

    const options = sentryMock.init.mock.calls[0][0] as Record<string, unknown>;
    expect(options.environment).toBe('production');
  });

  it('UMI_ENV=dev（构建期由 define 注入）时 environment 为 dev', async () => {
    process.env.SENTRY_DSN = 'https://public-key@example.com/1';
    process.env.UMI_ENV = 'dev';
    vi.resetModules();
    await import('./sentry');

    const options = sentryMock.init.mock.calls[0][0] as Record<string, unknown>;
    expect(options.environment).toBe('dev');
  });

  it('beforeSend 双保险剔除 Authorization/Cookie 请求头', async () => {
    process.env.SENTRY_DSN = 'https://public-key@example.com/1';
    vi.resetModules();
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
