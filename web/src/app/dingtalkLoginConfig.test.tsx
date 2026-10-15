import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { act, cleanup, render, screen } from '@testing-library/react';
import { DetachedBrowserFrame } from 'happy-dom';
import { StrictMode } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { DingTalkQrLogin } from '@/pages/user/login/components/dingtalk-qr-login';
import {
  dingTalkLoginConfigOptions,
  preloadDingTalkLoginConfig,
  refreshDingTalkLoginConfig,
} from './dingtalkLoginConfig';

const { getConfig } = vi.hoisted(() => ({ getConfig: vi.fn() }));
vi.mock('@/services/roncin/authService', () => ({
  authServiceGetDingTalkLoginConfig: getConfig,
}));

function config(state = 'state-1') {
  const authorize = new URL('https://login.dingtalk.com/oauth2/auth');
  authorize.search = new URLSearchParams({
    client_id: 'app',
    scope: 'openid',
    response_type: 'code',
    state,
    redirect_uri: `${window.location.origin}/user/login/dingtalk/callback`,
  }).toString();
  return {
    data: {
      enabled: true,
      authorizeUrl: authorize.href,
      expiresAt: new Date(Date.now() + 300000).toISOString(),
    },
  };
}

function mount(queryClient: ReturnType<typeof createTestQueryClient>) {
  return render(
    <StrictMode>
      <QueryClientProvider client={queryClient}>
        <DingTalkQrLogin onAuthenticated={vi.fn()} />
      </QueryClientProvider>
    </StrictMode>,
  );
}

async function settle() {
  await act(async () => vi.advanceTimersByTimeAsync(1));
}

describe('钉钉登录配置预热', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.spyOn(DetachedBrowserFrame.prototype, 'goto').mockImplementation(
      () => new Promise(() => {}),
    );
    getConfig.mockReset().mockResolvedValue(config());
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  it('只预热登录页，业务页与钉钉回调页不覆盖state Cookie', async () => {
    const client = createTestQueryClient();
    for (const pathname of ['/', '/welcome', '/user/login/dingtalk/callback']) {
      await preloadDingTalkLoginConfig(client, pathname);
    }
    expect(getConfig).not.toHaveBeenCalled();
    await preloadDingTalkLoginConfig(client, '/user/login');
    expect(getConfig).toHaveBeenCalledTimes(1);
    client.clear();
  });

  it('提前完成的预热经过无订阅间隔仍被StrictMode挂载复用', async () => {
    const client = createTestQueryClient();
    await preloadDingTalkLoginConfig(client, '/user/login');
    await act(async () => vi.advanceTimersByTimeAsync(30000));
    mount(client);
    await settle();
    expect(screen.getByTitle('钉钉扫码登录二维码')).toBeVisible();
    expect(getConfig).toHaveBeenCalledTimes(1);
    client.clear();
  });

  it('在途预热与StrictMode挂载共用同一个请求', async () => {
    const client = createTestQueryClient();
    let finish: (value: ReturnType<typeof config>) => void = () => {};
    getConfig.mockReturnValueOnce(
      new Promise<ReturnType<typeof config>>((resolve) => {
        finish = resolve;
      }),
    );
    const preload = preloadDingTalkLoginConfig(client, '/user/login');
    mount(client);
    await settle();
    expect(getConfig).toHaveBeenCalledTimes(1);
    await act(async () => {
      finish(config());
      await preload;
    });
    await settle();
    expect(screen.getByTitle('钉钉扫码登录二维码')).toBeVisible();
    expect(getConfig).toHaveBeenCalledTimes(1);
    client.clear();
  });

  it('预热失败在首次挂载与焦点恢复时直接显示错误，不暗中重试', async () => {
    const client = createTestQueryClient();
    getConfig.mockRejectedValueOnce(new Error('配置网络失败'));
    await preloadDingTalkLoginConfig(client, '/user/login');
    mount(client);
    await settle();
    expect(screen.getByText('获取钉钉登录配置失败，请重试')).toBeVisible();
    await act(async () => {
      window.dispatchEvent(new Event('focus'));
      window.dispatchEvent(new Event('online'));
      await vi.advanceTimersByTimeAsync(30000);
    });
    expect(getConfig).toHaveBeenCalledTimes(1);
    client.clear();
  });

  it('返回扫码显式刷新已完成配置，挂载不再发第二次请求', async () => {
    const client = createTestQueryClient();
    await preloadDingTalkLoginConfig(client, '/user/login');
    const view = mount(client);
    await settle();
    view.unmount();
    getConfig.mockResolvedValue(config('state-2'));
    refreshDingTalkLoginConfig(client);
    // 模拟点击与挂载之间配置已经完成。
    await settle();
    mount(client);
    await settle();
    expect(screen.getByTitle('钉钉扫码登录二维码')).toHaveAttribute(
      'src',
      expect.stringContaining('state=state-2'),
    );
    expect(getConfig).toHaveBeenCalledTimes(2);
    client.clear();
  });

  it('快速返回扫码复用在途请求，连续点击也不会并发写Cookie', async () => {
    const client = createTestQueryClient();
    let finish: (value: ReturnType<typeof config>) => void = () => {};
    getConfig.mockReturnValueOnce(
      new Promise<ReturnType<typeof config>>((resolve) => {
        finish = resolve;
      }),
    );
    const preload = preloadDingTalkLoginConfig(client, '/user/login');
    const view = mount(client);
    view.unmount();
    refreshDingTalkLoginConfig(client);
    refreshDingTalkLoginConfig(client);
    mount(client);
    expect(getConfig).toHaveBeenCalledTimes(1);
    await act(async () => {
      finish(config());
      await preload;
    });
    await settle();
    expect(screen.getByTitle('钉钉扫码登录二维码')).toBeVisible();
    expect(getConfig).toHaveBeenCalledTimes(1);
    client.clear();
  });

  it('返回扫码显式重试失败配置；挂载不增加第三次请求', async () => {
    const client = createTestQueryClient();
    getConfig.mockRejectedValueOnce(new Error('配置网络失败'));
    await preloadDingTalkLoginConfig(client, '/user/login');
    refreshDingTalkLoginConfig(client);
    mount(client);
    await settle();
    expect(screen.getByTitle('钉钉扫码登录二维码')).toBeVisible();
    expect(getConfig).toHaveBeenCalledTimes(2);
    client.clear();
  });

  it('预热配置已过期但尚未回收时挂载直接取新配置，不显示旧iframe', async () => {
    const client = createTestQueryClient();
    const response = config();
    response.data.expiresAt = new Date(Date.now() + 1000).toISOString();
    getConfig.mockResolvedValueOnce(response);
    await preloadDingTalkLoginConfig(client, '/user/login');
    await act(async () => vi.advanceTimersByTimeAsync(1001));
    getConfig.mockResolvedValue(config('state-2'));
    mount(client);
    expect(screen.queryByTitle('钉钉扫码登录二维码')).not.toBeInTheDocument();
    await settle();
    expect(screen.getByTitle('钉钉扫码登录二维码')).toHaveAttribute(
      'src',
      expect.stringContaining('state=state-2'),
    );
    expect(getConfig).toHaveBeenCalledTimes(2);
    client.clear();
  });

  it('无订阅的预热结果五分钟后回收，下次挂载重新获取', async () => {
    const client = createTestQueryClient();
    await preloadDingTalkLoginConfig(client, '/user/login');
    await act(async () => vi.advanceTimersByTimeAsync(300001));
    expect(
      client.getQueryData(dingTalkLoginConfigOptions.queryKey),
    ).toBeUndefined();
    getConfig.mockResolvedValue(config('state-2'));
    mount(client);
    await settle();
    expect(getConfig).toHaveBeenCalledTimes(2);
    expect(screen.getByTitle('钉钉扫码登录二维码')).toHaveAttribute(
      'src',
      expect.stringContaining('state=state-2'),
    );
    client.clear();
  });
});
