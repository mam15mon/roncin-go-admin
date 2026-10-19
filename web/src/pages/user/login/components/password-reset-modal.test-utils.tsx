import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { ConfigProvider } from 'antd';
import { expect, vi } from 'vitest';
import { PasswordResetModal } from './password-reset-modal';

const { requestCode, confirmPassword } = vi.hoisted(() => ({
  requestCode: vi.fn(),
  confirmPassword: vi.fn(),
}));

vi.mock('@/services/roncin/authService', () => ({
  authServiceRequestPasswordResetCode: requestCode,
  authServiceConfirmPasswordReset: confirmPassword,
}));

export const mobile = '13800138000';
export const challenge = (token = 'challenge-one') => ({
  success: true,
  data: {
    challengeToken: token,
    serverTime: '2026-10-19T00:00:00Z',
    resendAvailableAt: '2026-10-19T00:01:00Z',
    expiresAt: '2026-10-19T00:05:00Z',
  },
});

export function renderModal(initialMobile = mobile) {
  const onClose = vi.fn();
  const onSuccess = vi.fn();
  const view = renderWithApp(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <PasswordResetModal
        initialMobile={initialMobile}
        onClose={onClose}
        onSuccess={onSuccess}
      />
    </ConfigProvider>,
  );
  return { ...view, onClose, onSuccess };
}

export async function sendCode() {
  fireEvent.click(screen.getByRole('button', { name: '发送验证码' }));
  await waitFor(() => expect(requestCode).toHaveBeenCalled());
}

export async function readyChallenge() {
  requestCode.mockResolvedValue(challenge());
  await sendCode();
  await screen.findByText(
    '若该手机号对应已开通的钉钉账号，验证码将发送到本人钉钉，请前往钉钉查看。',
  );
}

export function fillPassword(
  code = '123456',
  password = 'new-password-1234',
  confirm = password,
) {
  fireEvent.change(screen.getByRole('textbox', { name: '钉钉验证码' }), {
    target: { value: code },
  });
  fireEvent.change(screen.getByLabelText('新密码'), {
    target: { value: password },
  });
  fireEvent.change(screen.getByLabelText('确认新密码'), {
    target: { value: confirm },
  });
}

export function submitPassword() {
  fireEvent.click(screen.getByRole('button', { name: '设置密码并返回登录' }));
}

export { confirmPassword, requestCode };
