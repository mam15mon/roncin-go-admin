import { renderWithApp } from '@root/tests/renderWithApp';
import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useInitialState } from '@/app/AppProvider';
import Login from './index';

const { sendCode, confirmPassword, login, clearOrderCache } = vi.hoisted(
  () => ({
    sendCode: vi.fn(),
    confirmPassword: vi.fn(),
    login: vi.fn(),
    clearOrderCache: vi.fn(),
  }),
);
vi.mock('@/services/roncin/authService', () => ({
  authServiceGetDingTalkLoginConfig: () =>
    Promise.resolve({ data: { enabled: false } }),
  authServiceRequestPasswordResetCode: sendCode,
  authServiceConfirmPasswordReset: confirmPassword,
  authServiceLogin: login,
}));
vi.mock('@/features/orders/options', () => ({
  clearOrderMasterDataCache: clearOrderCache,
}));

function StateProbe() {
  const { initialState } = useInitialState();
  return (
    <output aria-label="当前登录身份">
      {initialState?.currentUser?.id ?? '未登录'}
    </output>
  );
}

beforeEach(() => {
  vi.resetAllMocks();
  window.history.replaceState({}, '', '/user/login');
});
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  sessionStorage.clear();
});

describe('登录页账号与自助设置衔接', () => {
  it('重置成功清除旧身份及缓存、保留手机号并清空密码，返回账号模式且不自动登录', async () => {
    sendCode.mockResolvedValue({
      success: true,
      data: {
        challengeToken: 'reset-token',
        serverTime: '2026-10-19T00:00:00Z',
        resendAvailableAt: '2026-10-19T00:01:00Z',
        expiresAt: '2026-10-19T00:05:00Z',
      },
    });
    confirmPassword.mockResolvedValue({ success: true });
    const { queryClient } = renderWithApp(
      <>
        <Login />
        <StateProbe />
      </>,
      { user: { id: 'stale-user' } },
    );
    queryClient.setQueryData(['old-user-business'], {
      privateData: 'old-data',
    });
    sessionStorage.setItem('dingtalk_login_redirect', '/orders');
    expect(screen.getByRole('radio', { name: '钉钉扫码' })).toBeChecked();
    fireEvent.click(screen.getByRole('radio', { name: '账号登录' }));
    fireEvent.change(screen.getByPlaceholderText('钉钉手机号 / 本地账号'), {
      target: { value: '13800138000' },
    });
    fireEvent.change(screen.getByPlaceholderText('请输入密码'), {
      target: { value: 'old-password' },
    });
    fireEvent.click(screen.getByRole('button', { name: '设置 / 忘记密码' }));
    expect(screen.getByLabelText('钉钉手机号')).toHaveValue('13800138000');
    fireEvent.click(screen.getByRole('button', { name: '发送验证码' }));
    await screen.findByText(/请前往钉钉查看/);
    fireEvent.change(screen.getByRole('textbox', { name: '钉钉验证码' }), {
      target: { value: '123456' },
    });
    fireEvent.change(screen.getByLabelText('新密码'), {
      target: { value: 'new-password-1234' },
    });
    fireEvent.change(screen.getByLabelText('确认新密码'), {
      target: { value: 'new-password-1234' },
    });
    fireEvent.click(screen.getByRole('button', { name: '设置密码并返回登录' }));
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
    );
    expect(screen.getByLabelText('当前登录身份')).toHaveTextContent('未登录');
    expect(screen.getByRole('radio', { name: '账号登录' })).toBeChecked();
    expect(screen.getByPlaceholderText('钉钉手机号 / 本地账号')).toHaveValue(
      '13800138000',
    );
    expect(screen.getByPlaceholderText('请输入密码')).toHaveValue('');
    expect(clearOrderCache).toHaveBeenCalledOnce();
    expect(queryClient.getQueryData(['old-user-business'])).toBeUndefined();
    expect(sessionStorage.getItem('dingtalk_login_redirect')).toBeNull();
    expect(login).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: '设置 / 忘记密码' }));
    expect(screen.getByLabelText('新密码')).toHaveValue('');
    expect(screen.getByRole('textbox', { name: '钉钉验证码' })).toHaveValue('');
    expect(
      screen.getByRole('button', { name: '设置密码并返回登录' }),
    ).toBeDisabled();
  });

  it('手机号与本地账号都从同一个表单传入现有登录RPC', async () => {
    login.mockResolvedValue({ data: undefined });
    renderWithApp(<Login />);
    fireEvent.click(screen.getByRole('radio', { name: '账号登录' }));
    for (const username of ['13800138000', 'admin']) {
      fireEvent.change(screen.getByPlaceholderText('钉钉手机号 / 本地账号'), {
        target: { value: username },
      });
      fireEvent.change(screen.getByPlaceholderText('请输入密码'), {
        target: { value: 'password-1234' },
      });
      fireEvent.click(screen.getByRole('button', { name: '登录' }));
      await waitFor(() =>
        expect(login).toHaveBeenLastCalledWith(
          { username, password: 'password-1234' },
          { skipErrorHandler: true },
        ),
      );
      await waitFor(() =>
        expect(screen.getByRole('button', { name: '登录' })).toBeEnabled(),
      );
    }
  });
});
