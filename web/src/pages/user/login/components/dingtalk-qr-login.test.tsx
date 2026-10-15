import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, cleanup, fireEvent, screen } from '@testing-library/react';
import { DetachedBrowserFrame } from 'happy-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { DingTalkQrLogin } from './dingtalk-qr-login';

const { getConfig } = vi.hoisted(() => ({ getConfig: vi.fn() }));

vi.mock('@/services/roncin/authService', () => ({
  authServiceGetDingTalkLoginConfig: getConfig,
}));

const origin = 'https://login.dingtalk.com';
const callback = () => `${window.location.origin}/user/login/dingtalk/callback`;
const recovery =
  '若二维码未显示或加载失败，请检查网络后刷新二维码，或使用管理员账号登录。';

function config(state = 'state-1') {
  return {
    data: {
      enabled: true,
      authorizeUrl: `${origin}/oauth2/auth?client_id=app&scope=openid&response_type=code&state=${state}&redirect_uri=${encodeURIComponent(callback())}`,
      expiresAt: new Date(Date.now() + 300000).toISOString(),
    },
  };
}

async function mount() {
  const onAuthenticated = vi.fn();
  const view = renderWithClient(
    <DingTalkQrLogin onAuthenticated={onAuthenticated} />,
  );
  await act(async () => vi.advanceTimersByTimeAsync(1));
  const iframe = screen.getByTitle('钉钉扫码登录二维码') as HTMLIFrameElement;
  return { ...view, iframe, onAuthenticated };
}

function send(
  source: Window | null,
  data: unknown = {
    success: true,
    redirectUrl: `${callback()}?authCode=code&state=state-1`,
  },
  messageOrigin = origin,
) {
  fireEvent(
    window,
    new MessageEvent('message', { origin: messageOrigin, source, data }),
  );
}

describe('钉钉二维码生命周期', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    // 保留真实 iframe/window，导航事件由用例驱动，禁止单测访问钉钉网络。
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

  it('网络失败也触发 load 时保留可见恢复说明和刷新入口', async () => {
    const { iframe, onAuthenticated } = await mount();
    expect(iframe).toBeVisible();
    expect(screen.getByText('正在加载钉钉二维码')).toBeVisible();
    fireEvent.load(iframe);
    await act(async () => vi.advanceTimersByTimeAsync(16000));

    expect(screen.getByText(recovery)).toBeVisible();
    expect(screen.getByRole('button', { name: /刷新二维码/ })).toBeEnabled();
    expect(screen.queryByText('正在加载钉钉二维码')).not.toBeInTheDocument();
    expect(onAuthenticated).not.toHaveBeenCalled();
  });

  it('导航超过15秒显示慢加载提示，并允许尚未过期的扫码成功', async () => {
    const { iframe, onAuthenticated } = await mount();
    await act(async () => vi.advanceTimersByTimeAsync(15000));

    expect(
      screen.getByText('钉钉二维码加载较慢，请检查网络或刷新二维码'),
    ).toBeVisible();
    expect(iframe).toBeVisible();
    send(iframe.contentWindow);
    expect(onAuthenticated).toHaveBeenCalledWith(
      '/user/login/dingtalk/callback?authCode=code&state=state-1',
    );
  });

  it('明确失败立即退役，迟到的成功消息不可继续认证', async () => {
    const { iframe, onAuthenticated } = await mount();
    const source = iframe.contentWindow;
    send(source, { success: false });
    expect(
      screen.getByText('钉钉扫码未完成，请刷新二维码后重试'),
    ).toBeVisible();
    send(source);
    expect(onAuthenticated).not.toHaveBeenCalled();
  });

  it('过期后自动获取新配置并装载新二维码，旧实例消息被忽略', async () => {
    const { iframe, onAuthenticated } = await mount();
    const oldSource = iframe.contentWindow;
    getConfig.mockImplementation(async () => config('state-2'));
    await act(async () => vi.advanceTimersByTimeAsync(300000));
    await act(async () => vi.advanceTimersByTimeAsync(1));

    expect(getConfig).toHaveBeenCalledTimes(2);
    send(oldSource);
    expect(onAuthenticated).not.toHaveBeenCalled();
    const next = screen.getByTitle('钉钉扫码登录二维码') as HTMLIFrameElement;
    expect(next.src).toContain('state=state-2');
    send(next.contentWindow);
    expect(onAuthenticated).not.toHaveBeenCalled();
    send(next.contentWindow, {
      success: true,
      redirectUrl: `${callback()}?authCode=new-code&state=state-2`,
    });
    expect(onAuthenticated).toHaveBeenCalledWith(
      '/user/login/dingtalk/callback?authCode=new-code&state=state-2',
    );
  });

  it('新配置一返回就过期时限流自动刷新，退回手动刷新提示', async () => {
    const { onAuthenticated } = await mount();
    const expiredConfig = {
      data: {
        enabled: true,
        authorizeUrl: `${origin}/oauth2/auth?client_id=app&scope=openid&response_type=code&state=state-2&redirect_uri=${encodeURIComponent(callback())}`,
        expiresAt: new Date(Date.now() - 60000).toISOString(),
      },
    };
    getConfig.mockImplementation(async () => expiredConfig);
    await act(async () => vi.advanceTimersByTimeAsync(300000));
    await act(async () => vi.advanceTimersByTimeAsync(1));

    expect(getConfig).toHaveBeenCalledTimes(2);
    expect(screen.getByText('二维码已过期，请刷新')).toBeVisible();
    expect(screen.queryByTitle('钉钉扫码登录二维码')).not.toBeInTheDocument();
    expect(onAuthenticated).not.toHaveBeenCalled();
  });

  it('刷新等待新配置时卸载旧实例，新实例只接收自己的来源和state', async () => {
    const { iframe, onAuthenticated } = await mount();
    const oldSource = iframe.contentWindow;
    let resolveConfig: (value: ReturnType<typeof config>) => void = () => {};
    getConfig.mockReturnValueOnce(
      new Promise<ReturnType<typeof config>>((resolve) => {
        resolveConfig = resolve;
      }),
    );
    fireEvent.click(screen.getByRole('button', { name: /刷新二维码/ }));
    expect(screen.queryByTitle('钉钉扫码登录二维码')).not.toBeInTheDocument();
    send(oldSource);
    expect(onAuthenticated).not.toHaveBeenCalled();
    await act(async () => {
      resolveConfig(config('state-2'));
      await vi.advanceTimersByTimeAsync(1);
    });
    const next = screen.getByTitle('钉钉扫码登录二维码') as HTMLIFrameElement;
    send(oldSource);
    send(next.contentWindow);
    expect(onAuthenticated).not.toHaveBeenCalled();
    send(next.contentWindow, {
      success: true,
      redirectUrl: `${callback()}?authCode=new-code&state=state-2`,
    });
    expect(onAuthenticated).toHaveBeenCalledExactlyOnceWith(
      '/user/login/dingtalk/callback?authCode=new-code&state=state-2',
    );
    expect(getConfig).toHaveBeenCalledTimes(2);
  });

  it('来源、state、回调或重复字段不合法时忽略，仅处理一次有效成功', async () => {
    const { iframe, onAuthenticated } = await mount();
    const source = iframe.contentWindow;
    send(source, undefined, 'https://example.com');
    send(window);
    for (const redirectUrl of [
      `${callback()}?authCode=code&state=wrong`,
      `${callback()}?authCode=code&state=state-1&state=state-1`,
      `${callback()}?authCode=code&authCode=other&state=state-1`,
      'https://example.com/user/login/dingtalk/callback?authCode=code&state=state-1',
      `${window.location.origin}/other?authCode=code&state=state-1`,
    ]) {
      send(source, { success: true, redirectUrl });
    }
    expect(onAuthenticated).not.toHaveBeenCalled();
    send(source);
    send(source);
    expect(onAuthenticated).toHaveBeenCalledTimes(1);
  });

  it('组件卸载后旧消息不能触发认证', async () => {
    const { iframe, onAuthenticated, unmount } = await mount();
    const source = iframe.contentWindow;
    unmount();
    send(source);
    expect(onAuthenticated).not.toHaveBeenCalled();
  });
});
