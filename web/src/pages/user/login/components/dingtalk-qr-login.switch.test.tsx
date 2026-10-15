import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
} from '@testing-library/react';
import { DetachedBrowserFrame } from 'happy-dom';
import { StrictMode } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  dingTalkLoginConfigOptions,
  preloadDingTalkLoginConfig,
} from '@/app/dingtalkLoginConfig';
import { PRELOADED_FRAME_ID } from '@/app/dingtalkLoginProtocol';
import { DingTalkQrLogin } from './dingtalk-qr-login';

const { getConfig } = vi.hoisted(() => ({ getConfig: vi.fn() }));
vi.mock('@/services/roncin/authService', () => ({
  authServiceGetDingTalkLoginConfig: getConfig,
}));
const origin = 'https://login.dingtalk.com';
const callback = () => `${window.location.origin}/user/login/dingtalk/callback`;
function config(state = 'state-1', duration = 300000) {
  return {
    data: {
      enabled: true,
      authorizeUrl: `${origin}/oauth2/auth?client_id=app&scope=openid&response_type=code&state=${state}&redirect_uri=${encodeURIComponent(callback())}`,
      expiresAt: new Date(Date.now() + duration).toISOString(),
    },
  };
}
function send(source: Window | null, state = 'state-1', success = true) {
  fireEvent(
    window,
    new MessageEvent('message', {
      origin,
      source,
      data: {
        success,
        redirectUrl: `${callback()}?authCode=code&state=${state}`,
      },
    }),
  );
}
async function settle(milliseconds = 1) {
  await act(async () => vi.advanceTimersByTimeAsync(milliseconds));
}

describe.each(['HTML', 'dynamic'] as const)('%s 二维码切换保留', (kind) => {
  let client: ReturnType<typeof createTestQueryClient>;
  beforeEach(() => {
    vi.useFakeTimers();
    vi.spyOn(DetachedBrowserFrame.prototype, 'goto').mockImplementation(
      () => new Promise(() => {}),
    );
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({
      left: 10,
      top: 20,
      width: 280,
      height: 280,
    } as DOMRect);
    getConfig.mockReset().mockImplementation(async () => config('new-state'));
    client = createTestQueryClient();
  });
  afterEach(() => {
    cleanup();
    client.clear();
    document.getElementById(PRELOADED_FRAME_ID)?.remove();
    document.getElementById('roncin-dingtalk-login-config')?.remove();
    vi.restoreAllMocks();
    vi.useRealTimers();
  });
  async function mount(payload = config()) {
    if (kind === 'HTML') {
      const script = document.createElement('script');
      script.id = 'roncin-dingtalk-login-config';
      script.type = 'application/json';
      script.textContent = JSON.stringify(payload);
      document.head.appendChild(script);
      const frame = document.createElement('iframe');
      frame.id = PRELOADED_FRAME_ID;
      frame.src = `${payload.data.authorizeUrl}&iframe=true`;
      frame.title = '钉钉扫码登录二维码';
      frame.dataset.expiresAt = String(Date.parse(payload.data.expiresAt));
      document.body.appendChild(frame);
    } else getConfig.mockResolvedValueOnce(payload);
    await preloadDingTalkLoginConfig(client, '/user/login');
    const onAuthenticated = vi.fn();
    const ui = (active: boolean) => (
      <StrictMode>
        <QueryClientProvider client={client}>
          <DingTalkQrLogin active={active} onAuthenticated={onAuthenticated} />
        </QueryClientProvider>
      </StrictMode>
    );
    const view = render(ui(true));
    await settle();
    const frame = screen.getByTitle('钉钉扫码登录二维码') as HTMLIFrameElement;
    return {
      ...view,
      frame,
      onAuthenticated,
      switchTo: (active: boolean) => view.rerender(ui(active)),
      initialRequests: kind === 'HTML' ? 0 : 1,
    };
  }

  it('连续切换三次保留节点、父节点、src和浏览上下文，不再显示初始占位', async () => {
    const { frame, switchTo, initialRequests } = await mount();
    fireEvent.load(frame);
    const source = frame.contentWindow;
    const parent = frame.parentElement;
    const src = frame.src;
    const attributes = vi.spyOn(frame, 'setAttribute');
    for (let turn = 0; turn < 3; turn++) {
      switchTo(false);
      expect(frame).not.toBeVisible();
      expect(frame).toHaveAttribute('tabindex', '-1');
      expect(frame).toHaveAttribute('aria-hidden', 'true');
      switchTo(true);
      expect(frame).toBeVisible();
      expect(screen.getByTitle('钉钉扫码登录二维码')).toBe(frame);
      expect(frame.contentWindow).toBe(source);
      expect(frame.parentElement).toBe(parent);
      expect(frame.src).toBe(src);
      expect(screen.queryByText('正在加载钉钉二维码')).not.toBeInTheDocument();
      expect(
        screen.queryByText('正在获取钉钉登录配置'),
      ).not.toBeInTheDocument();
    }
    expect(attributes.mock.calls.some(([name]) => name === 'src')).toBe(false);
    expect(getConfig).toHaveBeenCalledTimes(initialRequests);
    expect(document.querySelectorAll('iframe')).toHaveLength(1);
  });

  it('隐藏期间忽略成功和失败消息，切回后恢复同一实例且只认证一次', async () => {
    const { frame, switchTo, onAuthenticated, initialRequests } = await mount();
    const source = frame.contentWindow;
    switchTo(false);
    send(source, 'state-1', false);
    send(source);
    expect(frame).toBeInTheDocument();
    expect(onAuthenticated).not.toHaveBeenCalled();
    switchTo(true);
    expect(getConfig).toHaveBeenCalledTimes(initialRequests);
    send(source);
    send(source);
    expect(onAuthenticated).toHaveBeenCalledExactlyOnceWith(
      '/user/login/dingtalk/callback?authCode=code&state=state-1',
    );
    expect(frame).not.toBeInTheDocument();
  });

  it('隐藏时到期先销毁，不在后台请求；返回只获取一次新配置并拒绝旧来源', async () => {
    const { frame, switchTo, onAuthenticated, initialRequests } = await mount(
      config('state-1', 10000),
    );
    const source = frame.contentWindow;
    switchTo(false);
    await settle(10001);
    expect(frame).not.toBeInTheDocument();
    expect(getConfig).toHaveBeenCalledTimes(initialRequests);
    switchTo(true);
    await settle();
    expect(getConfig).toHaveBeenCalledTimes(initialRequests + 1);
    const next = screen.getByTitle('钉钉扫码登录二维码') as HTMLIFrameElement;
    expect(next).not.toBe(frame);
    send(source);
    expect(onAuthenticated).not.toHaveBeenCalled();
    send(next.contentWindow, 'new-state');
    expect(onAuthenticated).toHaveBeenCalledOnce();
  });

  it('后台计时器尚未执行时切回也同步检查过期，不闪出旧二维码', async () => {
    const { frame, switchTo, initialRequests } = await mount();
    switchTo(false);
    vi.setSystemTime(Date.now() + 300001);
    switchTo(true);
    expect(frame).not.toBeInTheDocument();
    await settle();
    expect(getConfig).toHaveBeenCalledTimes(initialRequests + 1);
    expect(screen.getByTitle('钉钉扫码登录二维码')).not.toBe(frame);
  });

  it('明确失败后切换返回可获取新配置，失效实例不能复活', async () => {
    const { frame, switchTo, initialRequests } = await mount();
    fireEvent.error(frame);
    expect(frame).not.toBeInTheDocument();
    switchTo(false);
    expect(getConfig).toHaveBeenCalledTimes(initialRequests);
    switchTo(true);
    await settle();
    expect(getConfig).toHaveBeenCalledTimes(initialRequests + 1);
    expect(screen.getByTitle('钉钉扫码登录二维码')).not.toBe(frame);
  });

  it('失败实例隐藏失效缓存后返回刷新，新配置下一次切换仍保留', async () => {
    const { frame, switchTo, initialRequests } = await mount();
    fireEvent.error(frame);
    switchTo(false);
    await act(async () =>
      client.invalidateQueries({
        queryKey: dingTalkLoginConfigOptions.queryKey,
        refetchType: 'none',
      }),
    );
    switchTo(true);
    await settle();
    expect(getConfig).toHaveBeenCalledTimes(initialRequests + 1);
    const next = screen.getByTitle('钉钉扫码登录二维码');
    switchTo(false);
    switchTo(true);
    await settle();
    expect(getConfig).toHaveBeenCalledTimes(initialRequests + 1);
    expect(screen.getByTitle('钉钉扫码登录二维码')).toBe(next);
  });

  it('已消费实例切换后不能恢复，返回新配置，旧消息无效且卸载清理', async () => {
    const { frame, switchTo, onAuthenticated, initialRequests, unmount } =
      await mount();
    const source = frame.contentWindow;
    send(source);
    switchTo(false);
    switchTo(true);
    await settle();
    expect(getConfig).toHaveBeenCalledTimes(initialRequests + 1);
    send(source);
    expect(onAuthenticated).toHaveBeenCalledOnce();
    const next = screen.getByTitle('钉钉扫码登录二维码') as HTMLIFrameElement;
    const nextSource = next.contentWindow;
    unmount();
    send(nextSource, 'new-state');
    await settle();
    expect(next).not.toBeInTheDocument();
    expect(onAuthenticated).toHaveBeenCalledOnce();
  });

  it('隐藏时停止外部定位循环，恢复后再对齐', async () => {
    const { frame, switchTo } = await mount();
    const geometry = vi.mocked(HTMLElement.prototype.getBoundingClientRect);
    switchTo(false);
    const calls = geometry.mock.calls.length;
    await settle(100);
    expect(geometry).toHaveBeenCalledTimes(calls);
    switchTo(true);
    expect(frame).toBeVisible();
    if (kind === 'HTML')
      expect(geometry.mock.calls.length).toBeGreaterThan(calls);
  });
});
