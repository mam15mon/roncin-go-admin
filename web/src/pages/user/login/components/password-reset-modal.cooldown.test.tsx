import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  challenge,
  confirmPassword,
  fillPassword,
  readyChallenge,
  renderModal,
  requestCode,
  sendCode,
  submitPassword,
} from './password-reset-modal.test-utils';

beforeEach(() => vi.resetAllMocks());
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

describe('密码找回服务端冷却与消息结果', () => {
  it('按服务器时间差显示60秒，边界前不可重发，重发后的确认仅用新挑战', async () => {
    let monotonic = 1000;
    vi.spyOn(performance, 'now').mockImplementation(() => monotonic);
    renderModal();
    await readyChallenge();
    expect(screen.getByRole('button', { name: '60 秒后重发' })).toBeDisabled();
    monotonic += 59999;
    await waitFor(() =>
      expect(screen.getByRole('button', { name: '1 秒后重发' })).toBeDisabled(),
    );
    expect(requestCode).toHaveBeenCalledOnce();
    monotonic += 1;
    await waitFor(() =>
      expect(screen.getByRole('button', { name: '发送验证码' })).toBeEnabled(),
    );
    requestCode.mockResolvedValueOnce(challenge('new-token'));
    await sendCode();
    confirmPassword.mockResolvedValue({ success: true });
    fillPassword();
    submitPassword();
    await waitFor(() =>
      expect(confirmPassword).toHaveBeenCalledWith(
        expect.objectContaining({ challengeToken: 'new-token' }),
        expect.anything(),
      ),
    );
  });

  it('刷新或重开后收到429，用服务端剩余时间恢复倒计时，不凭空创建挑战', async () => {
    requestCode.mockRejectedValueOnce({
      response: {
        status: 429,
        data: {
          reason: 'AUTH_PASSWORD_RESET_COOLDOWN',
          message: '验证码发送过于频繁，请等待倒计时结束',
          metadata: {
            serverTime: '2026-10-19T00:00:25Z',
            resendAvailableAt: '2026-10-19T00:01:00Z',
          },
        },
      },
    });
    renderModal();
    await sendCode();
    expect(
      await screen.findByRole('button', { name: '35 秒后重发' }),
    ).toBeDisabled();
    expect(
      screen.getByText('验证码发送过于频繁，请等待倒计时结束'),
    ).toBeVisible();
    expect(
      screen.getByRole('button', { name: '设置密码并返回登录' }),
    ).toBeDisabled();
  });

  it('消息失败与网络结果未知不显示已发送，失败后旧挑战不可再提交', async () => {
    let monotonic = 1000;
    vi.spyOn(performance, 'now').mockImplementation(() => monotonic);
    const { onSuccess } = renderModal();
    await readyChallenge();
    monotonic += 60000;
    await waitFor(() =>
      expect(screen.getByRole('button', { name: '发送验证码' })).toBeEnabled(),
    );
    requestCode.mockRejectedValueOnce(new Error('Network Error'));
    await sendCode();
    expect(
      await screen.findByText(
        '验证码发送结果暂时无法确认，请稍后重试或使用钉钉扫码',
      ),
    ).toBeVisible();
    expect(screen.queryByText(/请前往钉钉查看/)).not.toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: '设置密码并返回登录' }),
    ).toBeDisabled();
    expect(onSuccess).not.toHaveBeenCalled();
    requestCode.mockRejectedValueOnce({
      data: { message: '暂时无法发送验证码，请稍后重试或使用钉钉扫码' },
    });
    fireEvent.click(screen.getByRole('button', { name: '发送验证码' }));
    expect(
      await screen.findByText('暂时无法发送验证码，请稍后重试或使用钉钉扫码'),
    ).toBeVisible();
  });
});
