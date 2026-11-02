import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { startRegistration } from '@simplewebauthn/browser';
import {
  act,
  cleanup,
  fireEvent,
  screen,
  waitFor,
} from '@testing-library/react';
import { App, ConfigProvider } from 'antd';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  passkeyServiceBeginPasskeyRegistration,
  passkeyServiceFinishPasskeyRegistration,
  passkeyServiceGetPasskeyConfig,
} from '@/services/roncin/passkeyService';
import { usePasskeyEnrollment } from './usePasskeyEnrollment';

vi.mock('@/services/roncin/passkeyService');
vi.mock('@simplewebauthn/browser', () => ({
  browserSupportsWebAuthn: () => true,
  startRegistration: vi.fn(),
  WebAuthnAbortService: { cancelCeremony: vi.fn() },
}));
const begin = vi.mocked(passkeyServiceBeginPasskeyRegistration);
const finish = vi.mocked(passkeyServiceFinishPasskeyRegistration);
const nativeRegistration = vi.mocked(startRegistration);
const continued = vi.fn();
const secureDescriptor = Object.getOwnPropertyDescriptor(
  window,
  'isSecureContext',
);

function EnrollmentProbe() {
  const enrollment = usePasskeyEnrollment();
  return (
    <>
      <button
        type="button"
        onClick={() => enrollment.completeLogin(continued, 'user-1')}
      >
        认证成功
      </button>
      {enrollment.enrollmentPrompt}
    </>
  );
}
function mount() {
  return renderWithClient(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <App>
        <EnrollmentProbe />
      </App>
    </ConfigProvider>,
  );
}

beforeEach(() => {
  vi.resetAllMocks();
  sessionStorage.clear();
  Object.defineProperty(window, 'isSecureContext', {
    configurable: true,
    value: true,
  });
  vi.mocked(passkeyServiceGetPasskeyConfig).mockResolvedValue({
    data: {
      enabled: true,
      rpId: 'localhost',
      rpName: 'Roncin',
      origins: [window.location.origin],
    },
  });
  begin.mockResolvedValue({
    success: true,
    data: {
      optionsJson: JSON.stringify({
        challenge: 'cmVnaXN0ZXI',
        rp: { id: 'localhost', name: 'Roncin' },
        user: { id: 'dXNlcg', name: 'account', displayName: '张三' },
        pubKeyCredParams: [{ type: 'public-key', alg: -7 }],
      }),
      expiresAt: new Date(Date.now() + 300_000).toISOString(),
    },
  });
  nativeRegistration.mockResolvedValue({
    id: 'new-credential',
    rawId: 'new-credential',
    response: { attestationObject: '', clientDataJSON: '' },
    clientExtensionResults: {},
    type: 'public-key',
  });
  finish.mockResolvedValue({
    success: true,
    data: { id: 'passkey-1', name: '我的通行密钥' },
  });
});
afterEach(() => {
  cleanup();
  if (secureDescriptor)
    Object.defineProperty(window, 'isSecureContext', secureDescriptor);
  else Reflect.deleteProperty(window, 'isSecureContext');
});

describe('登录后可选通行密钥注册', () => {
  it('未选择开启时继续原登录，不创建注册仪式', () => {
    mount();
    fireEvent.click(screen.getByRole('button', { name: '认证成功' }));
    expect(continued).toHaveBeenCalledOnce();
    expect(begin).not.toHaveBeenCalled();
  });

  it('必须明确点击添加，原生密钥保存成功后才继续', async () => {
    sessionStorage.setItem('roncin.passkey.enroll-after-login', 'true');
    mount();
    fireEvent.click(screen.getByRole('button', { name: '认证成功' }));
    const add = await screen.findByRole('button', { name: '添加通行密钥' });
    await waitFor(() => expect(add).toBeEnabled());
    expect(nativeRegistration).not.toHaveBeenCalled();
    expect(continued).not.toHaveBeenCalled();
    await act(async () => {
      fireEvent.click(add);
    });
    expect(nativeRegistration).toHaveBeenCalledOnce();
    await waitFor(() => expect(continued).toHaveBeenCalledOnce());
    expect(finish).toHaveBeenCalledWith(
      { credentialJson: expect.any(String), name: '我的通行密钥' },
      { skipErrorHandler: true },
    );
    expect(
      sessionStorage.getItem('roncin.passkey.enroll-after-login'),
    ).toBeNull();
  });

  it('取消原生注册后保留显式跳过入口，不丢失已认证会话', async () => {
    sessionStorage.setItem('roncin.passkey.enroll-after-login', 'true');
    nativeRegistration.mockRejectedValueOnce(
      new DOMException('Cancelled', 'NotAllowedError'),
    );
    mount();
    fireEvent.click(screen.getByRole('button', { name: '认证成功' }));
    const add = await screen.findByRole('button', { name: '添加通行密钥' });
    await waitFor(() => expect(add).toBeEnabled());
    await act(async () => {
      fireEvent.click(add);
    });
    expect(
      await screen.findByText('未完成通行密钥验证，可以重试或选择其他登录方式'),
    ).toBeVisible();
    expect(continued).not.toHaveBeenCalled();
    expect(finish).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: '暂不添加，继续登录' }));
    await waitFor(() => expect(continued).toHaveBeenCalledOnce());
  });
  it('注册提交后退出页面，迟到结果不得执行原登录跳转', async () => {
    sessionStorage.setItem('roncin.passkey.enroll-after-login', 'true');
    let resolveRegistration:
      | ((value: API.FinishPasskeyRegistrationResponse) => void)
      | undefined;
    finish.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          resolveRegistration = resolve;
        }),
    );
    const { unmount } = mount();
    fireEvent.click(screen.getByRole('button', { name: '认证成功' }));
    const add = await screen.findByRole('button', { name: '添加通行密钥' });
    await waitFor(() => expect(add).toBeEnabled());
    await act(async () => {
      fireEvent.click(add);
    });
    await waitFor(() => expect(finish).toHaveBeenCalledOnce());
    unmount();
    await act(async () => {
      resolveRegistration?.({ success: true, data: { id: 'key-1' } });
    });
    expect(continued).not.toHaveBeenCalled();
  });
  it('首次注册准备失败后可显式重新准备，再由用户添加', async () => {
    sessionStorage.setItem('roncin.passkey.enroll-after-login', 'true');
    begin.mockRejectedValueOnce(new Error('network unavailable'));
    mount();
    fireEvent.click(screen.getByRole('button', { name: '认证成功' }));
    const prepare = await screen.findByRole('button', {
      name: '重新准备通行密钥',
    });
    expect(nativeRegistration).not.toHaveBeenCalled();
    fireEvent.click(prepare);
    const add = screen.getByRole('button', { name: '添加通行密钥' });
    await waitFor(() => expect(add).toBeEnabled());
    expect(begin).toHaveBeenCalledTimes(2);
    expect(nativeRegistration).not.toHaveBeenCalled();
    fireEvent.click(add);
    await waitFor(() => expect(continued).toHaveBeenCalledOnce());
  });
});
