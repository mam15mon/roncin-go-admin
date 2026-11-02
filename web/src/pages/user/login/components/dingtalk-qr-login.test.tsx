import { deferred, qrConfig, settleQR } from '@root/tests/dingtalk-qr-support';
import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, cleanup, fireEvent, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { DingTalkQrLogin } from './dingtalk-qr-login';

const { getConfig, poll } = vi.hoisted(() => ({
  getConfig: vi.fn(),
  poll: vi.fn(),
}));
vi.mock('@/services/roncin/authService', () => ({
  authServiceGetDingTalkLoginConfig: getConfig,
  authServicePollDingTalkQrCode: poll,
}));

async function mount() {
  const onAuthenticated = vi.fn();
  const view = renderWithClient(
    <DingTalkQrLogin onAuthenticated={onAuthenticated} />,
  );
  await settleQR();
  return { ...view, onAuthenticated };
}

beforeEach(() => {
  vi.useFakeTimers();
  getConfig.mockReset().mockResolvedValue(qrConfig());
  poll.mockReset().mockImplementation(() => new Promise(() => {}));
});
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  vi.useRealTimers();
});

describe('本站二维码与扫码状态', () => {
  it('真实确认链接在本地绘制SVG，不创建外部iframe或灰色骨架', async () => {
    await mount();
    expect(
      screen
        .getByRole('img', { name: '钉钉扫码登录二维码' })
        .querySelector('svg'),
    ).not.toBeNull();
    expect(document.querySelector('iframe')).toBeNull();
    expect(screen.queryByText('正在加载钉钉二维码')).not.toBeInTheDocument();
    expect(poll).toHaveBeenCalledWith(
      { state: 'state-1' },
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    );
  });

  it('生成完成前只在固定区域显示加载提示，手动刷新串行生成新码', async () => {
    const first = deferred<ReturnType<typeof qrConfig>>();
    getConfig.mockReturnValueOnce(first.promise);
    const onAuthenticated = vi.fn();
    renderWithClient(<DingTalkQrLogin onAuthenticated={onAuthenticated} />);
    expect(screen.getByText('正在加载钉钉二维码')).toBeVisible();
    fireEvent.click(screen.getByRole('button', { name: /刷新二维码/ }));
    expect(getConfig).toHaveBeenCalledOnce();
    await act(async () => first.resolve(qrConfig()));
    await settleQR();
    const next = deferred<ReturnType<typeof qrConfig>>();
    getConfig.mockReturnValueOnce(next.promise);
    fireEvent.click(screen.getByRole('button', { name: /刷新二维码/ }));
    fireEvent.click(screen.getByRole('button', { name: /刷新二维码/ }));
    expect(
      screen.queryByRole('img', { name: '钉钉扫码登录二维码' }),
    ).not.toBeInTheDocument();
    expect(getConfig).toHaveBeenCalledTimes(2);
    await act(async () => next.resolve(qrConfig('state-2')));
    await settleQR();
    expect(poll).toHaveBeenLastCalledWith(
      { state: 'state-2' },
      expect.anything(),
    );
  });

  it('已扫码状态等待手机确认，实际确认后只导航本站回调一次', async () => {
    poll.mockResolvedValueOnce({ data: { status: 2 } }).mockResolvedValueOnce({
      data: { status: 3, authCode: 'confirmed-code' },
    });
    const { onAuthenticated } = await mount();
    expect(screen.getByText('已扫码，请在手机钉钉上确认')).toBeVisible();
    expect(onAuthenticated).not.toHaveBeenCalled();
    await settleQR(1001);
    expect(onAuthenticated).toHaveBeenCalledExactlyOnceWith(
      '/user/login/dingtalk/callback?authCode=confirmed-code&state=state-1',
    );
    await settleQR(3000);
    expect(onAuthenticated).toHaveBeenCalledOnce();
  });

  it('轮询失败不静默重试，刷新入口仍可用', async () => {
    poll.mockRejectedValueOnce(new Error('网络失败'));
    await mount();
    expect(screen.getByText('二维码加载失败')).toBeVisible();
    const calls = poll.mock.calls.length;
    await settleQR(5000);
    expect(poll).toHaveBeenCalledTimes(calls);
    expect(screen.getByRole('button', { name: /刷新二维码/ })).toBeEnabled();
    getConfig.mockResolvedValueOnce(qrConfig('retry-state'));
    fireEvent.click(screen.getByText('点击刷新'));
    await settleQR();
    expect(poll).toHaveBeenLastCalledWith(
      { state: 'retry-state' },
      expect.anything(),
    );
  });

  it('刷新后旧轮询的迟到确认不能认证，新尝试可以继续确认', async () => {
    const old = deferred<{ data: { status: number; authCode: string } }>();
    const next = deferred<{ data: { status: number; authCode: string } }>();
    poll.mockReturnValueOnce(old.promise).mockReturnValueOnce(next.promise);
    const { onAuthenticated } = await mount();
    getConfig.mockResolvedValue(qrConfig('state-2'));
    fireEvent.click(screen.getByRole('button', { name: /刷新二维码/ }));
    await settleQR();
    await act(async () =>
      old.resolve({ data: { status: 3, authCode: 'old-code' } }),
    );
    await settleQR();
    expect(onAuthenticated).not.toHaveBeenCalled();
    await act(async () =>
      next.resolve({ data: { status: 3, authCode: 'new-code' } }),
    );
    await settleQR();
    expect(onAuthenticated).toHaveBeenCalledExactlyOnceWith(
      '/user/login/dingtalk/callback?authCode=new-code&state=state-2',
    );
  });

  it('确认结构缺少授权码或状态未知时拒绝认证', async () => {
    poll.mockResolvedValueOnce({ data: { status: 3 } });
    const { onAuthenticated } = await mount();
    expect(onAuthenticated).not.toHaveBeenCalled();
    expect(screen.getByText('二维码加载失败')).toBeVisible();
  });
  it('点击健康二维码生成不同state的新码，不需要底部刷新按钮', async () => {
    await mount();
    const old = screen.getByRole('img', { name: '钉钉扫码登录二维码' });
    getConfig.mockResolvedValueOnce(qrConfig('clicked-state'));
    fireEvent.click(old);
    await settleQR();
    expect(getConfig).toHaveBeenCalledTimes(2);
    expect(poll).toHaveBeenLastCalledWith(
      { state: 'clicked-state' },
      expect.anything(),
    );
    expect(screen.getByRole('img', { name: '钉钉扫码登录二维码' })).not.toBe(
      old,
    );
    expect(
      screen.queryByText('使用前请联系管理员同步人员并分配组织和角色'),
    ).not.toBeInTheDocument();
    expect(screen.queryByText('刷新二维码')).not.toBeInTheDocument();
  });

  it('过期保持卡内失效状态，点击刷新后才获取新码', async () => {
    getConfig.mockResolvedValueOnce(qrConfig('expiring-state', 1000));
    await mount();
    await settleQR(1001);
    expect(screen.getByText('二维码已失效')).toBeVisible();
    expect(screen.getByText('点击刷新')).toBeVisible();
    expect(getConfig).toHaveBeenCalledOnce();
    getConfig.mockResolvedValueOnce(qrConfig('renewed-state'));
    fireEvent.click(screen.getByText('点击刷新'));
    await settleQR();
    expect(getConfig).toHaveBeenCalledTimes(2);
    expect(poll).toHaveBeenLastCalledWith(
      { state: 'renewed-state' },
      expect.anything(),
    );
  });
});
