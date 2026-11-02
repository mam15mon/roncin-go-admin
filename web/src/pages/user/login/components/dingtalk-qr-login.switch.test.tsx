import { deferred, qrConfig, settleQR } from '@root/tests/dingtalk-qr-support';
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
import { dingTalkLoginConfigOptions } from '@/app/dingtalkLoginConfig';
import { DingTalkQrLogin } from './dingtalk-qr-login';

const { getConfig, poll } = vi.hoisted(() => ({
  getConfig: vi.fn(),
  poll: vi.fn(),
}));
vi.mock('@/services/roncin/authService', () => ({
  authServiceGetDingTalkLoginConfig: getConfig,
  authServicePollDingTalkQrCode: poll,
}));
let client: ReturnType<typeof createTestQueryClient>;
beforeEach(() => {
  vi.useFakeTimers();
  client = createTestQueryClient();
  getConfig.mockReset().mockResolvedValue(qrConfig());
  poll.mockReset().mockImplementation(() => new Promise(() => {}));
});
afterEach(() => {
  cleanup();
  client.clear();
  vi.restoreAllMocks();
  vi.useRealTimers();
});
async function mount() {
  const onAuthenticated = vi.fn();
  const ui = (active: boolean) => (
    <StrictMode>
      <QueryClientProvider client={client}>
        <DingTalkQrLogin active={active} onAuthenticated={onAuthenticated} />
      </QueryClientProvider>
    </StrictMode>
  );
  const view = render(ui(true));
  await settleQR();
  return {
    ...view,
    onAuthenticated,
    switchTo: (active: boolean) => view.rerender(ui(active)),
  };
}

describe('本站二维码登录方式切换', () => {
  it('有效二维码连续切换仍复用同一DOM与配置，不出现初始加载', async () => {
    const { switchTo } = await mount();
    const qr = screen.getByRole('img', { name: '钉钉扫码登录二维码' });
    for (let i = 0; i < 3; i++) {
      switchTo(false);
      expect(qr).not.toBeVisible();
      switchTo(true);
      expect(screen.getByRole('img', { name: '钉钉扫码登录二维码' })).toBe(qr);
      expect(screen.queryByText('正在加载钉钉二维码')).not.toBeInTheDocument();
    }
    expect(getConfig).toHaveBeenCalledOnce();
  });

  it('隐藏期间取消轮询且迟到确认不在切回时补做认证', async () => {
    const result = deferred<{ data: { status: number; authCode: string } }>();
    // StrictMode 首次轮询可能被模拟卸载取消；所有旧请求共用相同受控结果。
    poll.mockImplementation(() => result.promise);
    const { switchTo, onAuthenticated } = await mount();
    switchTo(false);
    expect(poll.mock.calls.at(-1)?.[1].signal.aborted).toBe(true);
    await act(async () =>
      result.resolve({ data: { status: 3, authCode: 'hidden-code' } }),
    );
    await settleQR();
    expect(onAuthenticated).not.toHaveBeenCalled();
    getConfig.mockResolvedValue(qrConfig('new-state'));
    poll.mockImplementation(() => new Promise(() => {}));
    switchTo(true);
    await settleQR();
    expect(getConfig).toHaveBeenCalledTimes(2);
    expect(poll).toHaveBeenLastCalledWith(
      { state: 'new-state' },
      expect.anything(),
    );
    expect(onAuthenticated).not.toHaveBeenCalled();
  });

  it('隐藏到期不在后台生成新码，返回只获取一次新配置', async () => {
    getConfig.mockResolvedValueOnce(qrConfig('state-1', 10000));
    const { switchTo } = await mount();
    switchTo(false);
    await settleQR(10001);
    expect(getConfig).toHaveBeenCalledOnce();
    getConfig.mockResolvedValue(qrConfig('new-state'));
    switchTo(true);
    await settleQR();
    expect(getConfig).toHaveBeenCalledTimes(2);
    expect(
      client.getQueryData(dingTalkLoginConfigOptions.queryKey),
    ).toMatchObject({ enabled: true, qr: { state: 'new-state' } });
  });

  it('后台定时器未执行也同步拒绝显示已过期旧码', async () => {
    const { switchTo } = await mount();
    switchTo(false);
    vi.setSystemTime(Date.now() + 300001);
    getConfig.mockResolvedValue(qrConfig('new-state'));
    switchTo(true);
    expect(
      screen.queryByRole('img', { name: '钉钉扫码登录二维码' }),
    ).not.toBeInTheDocument();
    await settleQR();
    expect(getConfig).toHaveBeenCalledTimes(2);
  });

  it('轮询失败后返回生成新尝试，新码下一次切换仍可复用', async () => {
    poll.mockRejectedValue(new Error('状态请求失败'));
    const { switchTo } = await mount();
    expect(screen.getByText('二维码加载失败')).toBeVisible();
    switchTo(false);
    getConfig.mockResolvedValue(qrConfig('new-state'));
    poll.mockImplementation(() => new Promise(() => {}));
    switchTo(true);
    await settleQR();
    expect(getConfig).toHaveBeenCalledTimes(2);
    const next = screen.getByRole('img', { name: '钉钉扫码登录二维码' });
    switchTo(false);
    switchTo(true);
    await settleQR();
    expect(getConfig).toHaveBeenCalledTimes(2);
    expect(screen.getByRole('img', { name: '钉钉扫码登录二维码' })).toBe(next);
  });

  it('浏览器隐藏暂停轮询，恢复前台继续有效尝试', async () => {
    poll.mockResolvedValue({ data: { status: 1 } });
    await mount();
    const visibility = vi
      .spyOn(document, 'visibilityState', 'get')
      .mockReturnValue('hidden');
    act(() => document.dispatchEvent(new Event('visibilitychange')));
    const count = poll.mock.calls.length;
    await settleQR(3001);
    expect(poll).toHaveBeenCalledTimes(count);
    visibility.mockReturnValue('visible');
    act(() => document.dispatchEvent(new Event('visibilitychange')));
    await settleQR(1001);
    expect(poll.mock.calls.length).toBeGreaterThan(count);
    expect(getConfig).toHaveBeenCalledOnce();
  });
  it('浏览器隐藏时迟到确认只退役，回到前台获取新码且不补认证', async () => {
    const pending = deferred<{ data: { status: number; authCode: string } }>();
    poll.mockImplementation(() => pending.promise);
    const { onAuthenticated } = await mount();
    const visibility = vi
      .spyOn(document, 'visibilityState', 'get')
      .mockReturnValue('hidden');
    act(() => document.dispatchEvent(new Event('visibilitychange')));
    await act(async () =>
      pending.resolve({ data: { status: 3, authCode: 'hidden-code' } }),
    );
    await settleQR();
    expect(getConfig).toHaveBeenCalledOnce();
    expect(onAuthenticated).not.toHaveBeenCalled();
    getConfig.mockResolvedValue(qrConfig('foreground-state'));
    poll.mockImplementation(() => new Promise(() => {}));
    visibility.mockReturnValue('visible');
    act(() => document.dispatchEvent(new Event('visibilitychange')));
    await settleQR();
    expect(getConfig).toHaveBeenCalledTimes(2);
    expect(poll).toHaveBeenLastCalledWith(
      { state: 'foreground-state' },
      expect.anything(),
    );
    expect(onAuthenticated).not.toHaveBeenCalled();
  });

  it('浏览器隐藏期间到期不自动生成，前台恢复再刷新一次', async () => {
    getConfig.mockResolvedValueOnce(qrConfig('state-1', 10000));
    await mount();
    const visibility = vi
      .spyOn(document, 'visibilityState', 'get')
      .mockReturnValue('hidden');
    act(() => document.dispatchEvent(new Event('visibilitychange')));
    await settleQR(10001);
    expect(getConfig).toHaveBeenCalledOnce();
    getConfig.mockResolvedValue(qrConfig('foreground-state'));
    visibility.mockReturnValue('visible');
    act(() => document.dispatchEvent(new Event('visibilitychange')));
    await settleQR();
    expect(getConfig).toHaveBeenCalledTimes(2);
  });
  it('先恢复前台再收到旧取消轮询的确认时，退役旧码并保留点击刷新', async () => {
    const requests: Array<{
      signal: AbortSignal;
      result: ReturnType<
        typeof deferred<{ data: { status: number; authCode: string } }>
      >;
    }> = [];
    poll.mockImplementation((_body, options) => {
      const result = deferred<{ data: { status: number; authCode: string } }>();
      requests.push({ signal: options.signal, result });
      return result.promise;
    });
    const { onAuthenticated } = await mount();
    const cancelled = requests.at(-1);
    expect(cancelled).toBeDefined();
    const visibility = vi
      .spyOn(document, 'visibilityState', 'get')
      .mockReturnValue('hidden');
    act(() => document.dispatchEvent(new Event('visibilitychange')));
    expect(cancelled?.signal.aborted).toBe(true);
    visibility.mockReturnValue('visible');
    act(() => document.dispatchEvent(new Event('visibilitychange')));
    await settleQR();
    expect(getConfig).toHaveBeenCalledOnce();
    await act(async () =>
      cancelled?.result.resolve({
        data: { status: 3, authCode: 'late-cancelled-code' },
      }),
    );
    await settleQR();
    expect(onAuthenticated).not.toHaveBeenCalled();
    expect(screen.getByText('二维码已失效')).toBeVisible();
    getConfig.mockResolvedValueOnce(qrConfig('retried-state'));
    fireEvent.click(screen.getByText('点击刷新'));
    await settleQR();
    expect(getConfig).toHaveBeenCalledTimes(2);
    expect(poll).toHaveBeenLastCalledWith(
      { state: 'retried-state' },
      expect.anything(),
    );
    expect(onAuthenticated).not.toHaveBeenCalled();
  });
});
