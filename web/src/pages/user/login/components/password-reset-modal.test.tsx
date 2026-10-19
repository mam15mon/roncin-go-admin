import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  confirmPassword,
  fillPassword,
  mobile,
  readyChallenge,
  renderModal,
  requestCode,
  submitPassword,
} from './password-reset-modal.test-utils';

beforeEach(() => vi.resetAllMocks());
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

describe('钉钉验证码设置与找回密码', () => {
  it('首次设置与找回复用同一表单，正确提交挑战与新密码，不传确认密码', async () => {
    confirmPassword.mockResolvedValue({ success: true });
    const { onSuccess } = renderModal();
    await waitFor(() =>
      expect(screen.getByText('首次设置与找回使用同一流程')).toBeVisible(),
    );
    await readyChallenge();
    expect(requestCode).toHaveBeenCalledWith(
      { mobile },
      { skipErrorHandler: true },
    );
    fillPassword('001234');
    submitPassword();
    await waitFor(() => expect(onSuccess).toHaveBeenCalledWith(mobile));
    expect(confirmPassword).toHaveBeenCalledWith(
      {
        challengeToken: 'challenge-one',
        code: '001234',
        newPassword: 'new-password-1234',
      },
      { skipErrorHandler: true },
    );
  });

  it('两次密码不一致或长度不足时不提交，修正后可继续', async () => {
    const { onSuccess } = renderModal();
    await readyChallenge();
    fillPassword('123456', 'short', 'different');
    submitPassword();
    expect(await screen.findByText('密码至少 12 位')).toBeVisible();
    expect(await screen.findByText('两次输入的密码不一致')).toBeVisible();
    expect(confirmPassword).not.toHaveBeenCalled();
    confirmPassword.mockResolvedValue({ success: true });
    fillPassword();
    submitPassword();
    await waitFor(() => expect(onSuccess).toHaveBeenCalledOnce());
  });

  it('格式错误的手机号和非六位验证码不发请求，合法国际前缀保留原载荷', async () => {
    renderModal('123');
    fireEvent.click(screen.getByRole('button', { name: '发送验证码' }));
    expect(await screen.findByText('请输入有效的国内手机号')).toBeVisible();
    expect(requestCode).not.toHaveBeenCalled();
    fireEvent.change(screen.getByLabelText('钉钉手机号'), {
      target: { value: '+86 138-0013-8000' },
    });
    await readyChallenge();
    expect(requestCode).toHaveBeenCalledWith(
      { mobile: '+86 138-0013-8000' },
      { skipErrorHandler: true },
    );
    fillPassword('12345');
    submitPassword();
    expect(await screen.findByText('请输入 6 位数字验证码')).toBeVisible();
    expect(confirmPassword).not.toHaveBeenCalled();
  });

  it('验证码错误保留输入并展示服务端作废提示，错误和空响应均不误报成功', async () => {
    confirmPassword.mockRejectedValueOnce({
      response: { data: { message: '验证码无效或已过期，请重新获取' } },
    });
    const { onSuccess } = renderModal();
    await readyChallenge();
    fillPassword();
    submitPassword();
    expect(
      await screen.findByText('验证码无效或已过期，请重新获取'),
    ).toBeVisible();
    expect(screen.getByLabelText('新密码')).toHaveValue('new-password-1234');
    expect(onSuccess).not.toHaveBeenCalled();
    confirmPassword.mockResolvedValueOnce(undefined);
    submitPassword();
    await waitFor(() => expect(confirmPassword).toHaveBeenCalledTimes(2));
    expect(onSuccess).not.toHaveBeenCalled();
    expect(screen.getByRole('dialog')).toBeVisible();
  });
});
