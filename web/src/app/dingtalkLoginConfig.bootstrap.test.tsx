import { qrConfig as config } from '@root/tests/dingtalk-qr-support';
import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
} from '@testing-library/react';
import { StrictMode } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { DingTalkQrLogin } from '@/pages/user/login/components/dingtalk-qr-login';
import {
  dingTalkLoginConfigOptions,
  preloadDingTalkLoginConfig,
  refreshDingTalkLoginConfig,
} from './dingtalkLoginConfig';

const { getConfig, poll } = vi.hoisted(() => ({
  getConfig: vi.fn(),
  poll: vi.fn(),
}));
vi.mock('@/services/roncin/authService', () => ({
  authServiceGetDingTalkLoginConfig: getConfig,
  authServicePollDingTalkQrCode: poll,
}));

function inline(payload: unknown, raw = false) {
  const script = document.createElement('script');
  script.id = 'roncin-dingtalk-login-config';
  script.type = 'application/json';
  script.textContent = raw ? String(payload) : JSON.stringify(payload);
  document.head.appendChild(script);
  return script;
}

function mount(client: ReturnType<typeof createTestQueryClient>) {
  return render(
    <StrictMode>
      <QueryClientProvider client={client}>
        <DingTalkQrLogin onAuthenticated={vi.fn()} />
      </QueryClientProvider>
    </StrictMode>,
  );
}

async function settle() {
  await act(async () => vi.advanceTimersByTimeAsync(1));
}

describe('钉钉登录 HTML 内嵌配置', () => {
  let client: ReturnType<typeof createTestQueryClient>;
  beforeEach(() => {
    vi.useFakeTimers();
    poll.mockReset().mockImplementation(() => new Promise(() => {}));
    client = createTestQueryClient();
    getConfig.mockReset().mockResolvedValue(config('new-state'));
  });

  afterEach(() => {
    cleanup();
    client.clear();
    document.getElementById('roncin-dingtalk-login-config')?.remove();
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  it('预热一次性消费节点，StrictMode与重复预热复用配置，首屏零GET', async () => {
    const script = inline(config());
    await preloadDingTalkLoginConfig(client, '/user/login');
    expect(script).not.toBeInTheDocument();
    mount(client);
    await preloadDingTalkLoginConfig(client, '/user/login');
    await settle();
    expect(
      client.getQueryData(dingTalkLoginConfigOptions.queryKey),
    ).toMatchObject({ enabled: true, qr: { state: 'state-1' } });
    expect(getConfig).not.toHaveBeenCalled();
  });

  it.each([{ enabled: false }, {}])(
    '禁用配置%s直接展示禁用且不GET',
    async (data) => {
      inline({ success: true, data });
      await preloadDingTalkLoginConfig(client, '/user/login');
      mount(client);
      await settle();
      expect(screen.getByText('钉钉登录暂未启用，请联系管理员')).toBeVisible();
      expect(getConfig).not.toHaveBeenCalled();
    },
  );

  it.each([
    [
      '服务端失败',
      {
        success: false,
        code: 500,
        message: '钉钉登录配置加载失败，请稍后重试',
      },
    ],
    ['坏JSON', '{'],
    ['空信封', null],
    ['错误数据类型', { success: true, data: [] }],
    ['错误启用类型', { success: true, data: { enabled: 'false' } }],
    ['缺授权参数', { success: true, data: { enabled: true } }],
  ])('%s直接进入错误状态，不自动GET', async (_name, payload) => {
    const script = inline(payload, typeof payload === 'string');
    await preloadDingTalkLoginConfig(client, '/user/login');
    const state = client.getQueryState(dingTalkLoginConfigOptions.queryKey);
    expect(state?.status).toBe('error');
    expect(state?.error).toBeInstanceOf(Error);
    expect(script).not.toBeInTheDocument();
    expect(getConfig).not.toHaveBeenCalled();
  });

  it.each(['origin', 'path', 'invalid-url', 'duplicate-code'])(
    '内嵌二维码地址%s不合法不能绕过同一校验',
    async (kind) => {
      const payload = config();
      const qr = new URL(payload.data.qrCodeUrl);
      if (kind === 'origin') qr.hostname = 'evil.example.com';
      if (kind === 'path') qr.pathname = '/oauth2/auth';
      if (kind === 'duplicate-code') qr.searchParams.append('code', 'second');
      payload.data.qrCodeUrl = kind === 'invalid-url' ? 'invalid' : qr.href;
      inline(payload);
      await preloadDingTalkLoginConfig(client, '/user/login');
      expect(
        client.getQueryState(dingTalkLoginConfigOptions.queryKey)?.status,
      ).toBe('error');
      expect(getConfig).not.toHaveBeenCalled();
    },
  );

  it('内嵌失败在StrictMode挂载、焦点恢复均不重试，手动刷新才GET', async () => {
    inline({ success: false, code: 500 });
    await preloadDingTalkLoginConfig(client, '/user/login');
    mount(client);
    await settle();
    expect(screen.getByText('二维码加载失败')).toBeVisible();
    await act(async () => window.dispatchEvent(new Event('focus')));
    expect(getConfig).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: /刷新二维码/ }));
    await settle();
    expect(getConfig).toHaveBeenCalledOnce();
    expect(
      client.getQueryData(dingTalkLoginConfigOptions.queryKey),
    ).toMatchObject({ enabled: true, qr: { state: 'new-state' } });
  });

  it('回调和业务页不读取节点，返回扫码显式刷新只获取一次新配置', async () => {
    const script = inline(config());
    await preloadDingTalkLoginConfig(client, '/user/login/dingtalk/callback');
    await preloadDingTalkLoginConfig(client, '/welcome');
    expect(script).toBeInTheDocument();
    expect(getConfig).not.toHaveBeenCalled();
    await preloadDingTalkLoginConfig(client, '/user/login');
    refreshDingTalkLoginConfig(client);
    mount(client);
    await settle();
    expect(getConfig).toHaveBeenCalledOnce();
    expect(
      client.getQueryData(dingTalkLoginConfigOptions.queryKey),
    ).toMatchObject({ enabled: true, qr: { state: 'new-state' } });
  });

  it('内嵌配置到期显示失效，点击二维码通过GET获取新state', async () => {
    const payload = config();
    payload.data.expiresAt = new Date(Date.now() + 1000).toISOString();
    inline(payload);
    await preloadDingTalkLoginConfig(client, '/user/login');
    mount(client);
    await settle();
    await act(async () => vi.advanceTimersByTimeAsync(1001));
    await settle();
    expect(getConfig).not.toHaveBeenCalled();
    expect(screen.getByText('二维码已失效')).toBeVisible();
    fireEvent.click(screen.getByRole('button', { name: '刷新二维码' }));
    await settle();
    expect(getConfig).toHaveBeenCalledOnce();
    expect(
      client.getQueryData(dingTalkLoginConfigOptions.queryKey),
    ).toMatchObject({ enabled: true, qr: { state: 'new-state' } });
  });
});
