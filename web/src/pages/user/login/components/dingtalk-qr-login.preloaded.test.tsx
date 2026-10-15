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
import { preloadDingTalkLoginConfig } from '@/app/dingtalkLoginConfig';
import { PRELOADED_FRAME_ID } from '@/app/dingtalkLoginProtocol';
import { DingTalkQrLogin } from './dingtalk-qr-login';

const { getConfig } = vi.hoisted(() => ({ getConfig: vi.fn() }));
vi.mock('@/services/roncin/authService', () => ({
  authServiceGetDingTalkLoginConfig: getConfig,
}));
const origin = 'https://login.dingtalk.com';
const callback = () => `${window.location.origin}/user/login/dingtalk/callback`;
function config(state = 'html-state') {
  const url = new URL(`${origin}/oauth2/auth`);
  url.search = new URLSearchParams({
    state,
    client_id: 'app',
    scope: 'openid',
    response_type: 'code',
    redirect_uri: callback(),
  }).toString();
  return {
    success: true,
    data: {
      enabled: true,
      authorizeUrl: url.href,
      expiresAt: new Date(Date.now() + 300000).toISOString(),
    },
  };
}
function inline(payload: unknown) {
  const script = document.createElement('script');
  script.id = 'roncin-dingtalk-login-config';
  script.type = 'application/json';
  script.textContent = JSON.stringify(payload);
  document.head.appendChild(script);
}
function preloaded(payload = config()) {
  inline(payload);
  const frame = document.createElement('iframe');
  const url = new URL(payload.data.authorizeUrl);
  url.searchParams.set('iframe', 'true');
  // Go 输出按键排序；浏览器端必须按完整 URL 语义认领，不改 src。
  url.searchParams.sort();
  frame.src = url.href;
  frame.id = PRELOADED_FRAME_ID;
  frame.title = '钉钉扫码登录二维码';
  frame.dataset.expiresAt = String(Date.parse(payload.data.expiresAt));
  frame.style.visibility = 'hidden';
  document.body.appendChild(frame);
  return frame;
}
function send(
  source: Window | null,
  state = 'html-state',
  messageOrigin = origin,
) {
  fireEvent(
    window,
    new MessageEvent('message', {
      origin: messageOrigin,
      source,
      data: {
        success: true,
        redirectUrl: `${callback()}?authCode=code&state=${state}`,
      },
    }),
  );
}
async function settle(milliseconds = 1) {
  await act(async () => vi.advanceTimersByTimeAsync(milliseconds));
}

describe('HTML 二维码实例接管', () => {
  let client: ReturnType<typeof createTestQueryClient>;
  let geometry: DOMRect;
  beforeEach(() => {
    vi.useFakeTimers();
    vi.spyOn(DetachedBrowserFrame.prototype, 'goto').mockImplementation(
      () => new Promise(() => {}),
    );
    geometry = { left: 120, top: 160, width: 280, height: 280 } as DOMRect;
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(
      () => geometry,
    );
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
  async function mount() {
    await preloadDingTalkLoginConfig(client, '/user/login');
    const onAuthenticated = vi.fn();
    const view = render(
      <StrictMode>
        <QueryClientProvider client={client}>
          <DingTalkQrLogin onAuthenticated={onAuthenticated} />
        </QueryClientProvider>
      </StrictMode>,
    );
    await settle();
    return { ...view, onAuthenticated };
  }

  it('StrictMode 认领同一个浏览上下文且不移动节点、重写src或GET配置', async () => {
    const frame = preloaded();
    const source = frame.contentWindow;
    const parent = frame.parentElement;
    const src = frame.src;
    const attributes = vi.spyOn(frame, 'setAttribute');
    await mount();
    expect(screen.getByTitle('钉钉扫码登录二维码')).toBe(frame);
    expect(frame.contentWindow).toBe(source);
    expect(frame.parentElement).toBe(parent);
    expect(frame.src).toBe(src);
    expect(attributes.mock.calls.some(([name]) => name === 'src')).toBe(false);
    expect(frame).toBeVisible();
    expect(document.querySelectorAll('iframe')).toHaveLength(1);
    expect(getConfig).not.toHaveBeenCalled();
  });

  it('早期已完成load不再等待导航，也不会15秒后显示错误慢提示', async () => {
    const frame = preloaded();
    frame.dataset.navigation = 'loaded';
    await mount();
    await settle(16000);
    expect(frame).toBeVisible();
    expect(screen.queryByText('正在加载钉钉二维码')).not.toBeInTheDocument();
    expect(
      screen.queryByText('钉钉二维码加载较慢，请检查网络或刷新二维码'),
    ).not.toBeInTheDocument();
    expect(screen.getByText(/若二维码未显示或加载失败/)).toBeVisible();
  });

  it('早期明确失败立即移除预载实例并允许手动恢复', async () => {
    const frame = preloaded();
    frame.dataset.navigation = 'failed';
    await mount();
    expect(frame).not.toBeInTheDocument();
    expect(screen.getByText('钉钉二维码加载失败，请刷新重试')).toBeVisible();
    expect(screen.getByRole('button', { name: /刷新二维码/ })).toBeEnabled();
    expect(getConfig).not.toHaveBeenCalled();
  });

  it('刷新请求发出前已销毁预载实例，旧source不再认证', async () => {
    const frame = preloaded();
    const source = frame.contentWindow;
    const { onAuthenticated } = await mount();
    getConfig.mockImplementation(async () => {
      expect(frame).not.toBeInTheDocument();
      return config('new-state');
    });
    fireEvent.click(screen.getByRole('button', { name: /刷新二维码/ }));
    expect(frame).not.toBeInTheDocument();
    send(source);
    await settle();
    expect(onAuthenticated).not.toHaveBeenCalled();
    const next = screen.getByTitle('钉钉扫码登录二维码') as HTMLIFrameElement;
    send(next.contentWindow, 'new-state');
    expect(onAuthenticated).toHaveBeenCalledOnce();
    expect(getConfig).toHaveBeenCalledOnce();
  });

  it.each(['disabled', 'invalid', 'mismatch'])(
    '%s 配置销毁不对应的HTML实例',
    async (kind) => {
      const frame = preloaded();
      document.getElementById('roncin-dingtalk-login-config')?.remove();
      if (kind === 'disabled') inline({ success: true, data: {} });
      if (kind === 'invalid') inline({ success: false });
      if (kind === 'mismatch') inline(config('different-state'));
      await mount();
      expect(frame).not.toBeInTheDocument();
      expect(getConfig).not.toHaveBeenCalled();
      if (kind === 'mismatch') {
        expect(screen.getByTitle('钉钉扫码登录二维码')).toHaveAttribute(
          'src',
          expect.stringContaining('different-state'),
        );
      } else
        expect(
          screen.queryByTitle('钉钉扫码登录二维码'),
        ).not.toBeInTheDocument();
    },
  );

  it('位置随布局变化同步，卸载立即隐藏、清监听并销毁', async () => {
    const frame = preloaded();
    const source = frame.contentWindow;
    const { unmount, onAuthenticated } = await mount();
    expect(frame.style.left).toBe('120px');
    geometry = { left: 12, top: 240, width: 250, height: 280 } as DOMRect;
    await settle(20);
    expect(frame.style.left).toBe('12px');
    expect(frame.style.top).toBe('240px');
    expect(frame.style.width).toBe('250px');
    unmount();
    expect(frame.style.visibility).toBe('hidden');
    send(source);
    await settle();
    expect(frame).not.toBeInTheDocument();
    expect(onAuthenticated).not.toHaveBeenCalled();
    expect(vi.getTimerCount()).toBeLessThanOrEqual(1);
  });

  it('原生实例同样严格校验origin、source、state，只处理一次成功', async () => {
    const frame = preloaded();
    const source = frame.contentWindow;
    const { onAuthenticated } = await mount();
    send(source, 'html-state', 'https://evil.example');
    send(window);
    send(source, 'wrong-state');
    expect(onAuthenticated).not.toHaveBeenCalled();
    send(source);
    send(source);
    expect(onAuthenticated).toHaveBeenCalledExactlyOnceWith(
      '/user/login/dingtalk/callback?authCode=code&state=html-state',
    );
    expect(frame).not.toBeInTheDocument();
  });
});
