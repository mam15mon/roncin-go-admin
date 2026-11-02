import { renderWithClient } from '@root/tests/queryClientTestUtils';
import {
  browserSupportsWebAuthn,
  startAuthentication,
  WebAuthnAbortService,
} from '@simplewebauthn/browser';
import {
  act,
  cleanup,
  fireEvent,
  screen,
  waitFor,
} from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  passkeyServiceBeginPasskeyLogin,
  passkeyServiceFinishPasskeyLogin,
  passkeyServiceGetPasskeyConfig,
} from '@/services/roncin/passkeyService';
import { usePasskeyLogin } from './usePasskeyLogin';

vi.mock('@/services/roncin/passkeyService');
vi.mock('@simplewebauthn/browser', () => ({
  browserSupportsWebAuthn: vi.fn(),
  startAuthentication: vi.fn(),
  WebAuthnAbortService: { cancelCeremony: vi.fn() },
}));

const getConfig = vi.mocked(passkeyServiceGetPasskeyConfig);
const begin = vi.mocked(passkeyServiceBeginPasskeyLogin);
const finish = vi.mocked(passkeyServiceFinishPasskeyLogin);
const nativeAuthentication = vi.mocked(startAuthentication);
const secureDescriptor = Object.getOwnPropertyDescriptor(
  window,
  'isSecureContext',
);
const accepted = vi.fn();

function LoginProbe({ rememberMe = false }: { rememberMe?: boolean }) {
  const passkey = usePasskeyLogin(rememberMe);
  return (
    <>
      <span>{passkey.reason}</span>
      <button
        type="button"
        disabled={!passkey.canUse || passkey.isPreparing || passkey.isPending}
        onClick={() => {
          void passkey
            .login()
            .then(accepted)
            .catch(() => undefined);
        }}
      >
        免密登录
      </button>
      <button
        type="button"
        onClick={() => {
          void passkey.prepareAgain();
        }}
      >
        重新准备
      </button>
      <button type="button" onClick={() => passkey.setEnrollAfterLogin(true)}>
        登录后添加
      </button>
      {passkey.error && <div role="alert">{passkey.error}</div>}
    </>
  );
}

beforeEach(() => {
  vi.resetAllMocks();
  sessionStorage.clear();
  Object.defineProperty(window, 'isSecureContext', {
    configurable: true,
    value: true,
  });
  vi.mocked(browserSupportsWebAuthn).mockReturnValue(true);
  getConfig.mockResolvedValue({
    success: true,
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
        challenge: 'dGVzdA',
        rpId: 'localhost',
        userVerification: 'required',
      }),
      expiresAt: new Date(Date.now() + 300_000).toISOString(),
    },
  });
  nativeAuthentication.mockResolvedValue({
    id: 'credential',
    rawId: 'credential',
    response: { authenticatorData: '', clientDataJSON: '', signature: '' },
    clientExtensionResults: {},
    type: 'public-key',
  });
  finish.mockResolvedValue({
    success: true,
    data: { id: 'user-1' },
    organizationChoices: [],
  });
});
afterEach(() => {
  cleanup();
  if (secureDescriptor)
    Object.defineProperty(window, 'isSecureContext', secureDescriptor);
  else Reflect.deleteProperty(window, 'isSecureContext');
});

describe('本站通行密钥登录', () => {
  it.each(['HTTP地址', '未启用', '未授权来源', '不支持的浏览器'])(
    '%s 不创建验证仪式',
    async (kind) => {
      if (kind === '不支持的浏览器')
        vi.mocked(browserSupportsWebAuthn).mockReturnValue(false);
      if (kind === 'HTTP地址')
        Object.defineProperty(window, 'isSecureContext', {
          configurable: true,
          value: false,
        });
      if (kind === '未启用')
        getConfig.mockResolvedValue({ data: { enabled: false } });
      if (kind === '未授权来源')
        getConfig.mockResolvedValue({
          data: {
            enabled: true,
            rpId: 'localhost',
            rpName: 'Roncin',
            origins: ['https://other.example'],
          },
        });
      renderWithClient(<LoginProbe />);
      await waitFor(() => expect(getConfig).toHaveBeenCalledOnce());
      await act(async () => {});
      expect(screen.getByRole('button', { name: '免密登录' })).toBeDisabled();
      expect(begin).not.toHaveBeenCalled();
      expect(nativeAuthentication).not.toHaveBeenCalled();
    },
  );

  it('先准备选项，点击时同步启动原生验证并提交真实记忆偏好', async () => {
    renderWithClient(<LoginProbe />);
    const button = screen.getByRole('button', { name: '免密登录' });
    await waitFor(() => expect(button).toBeEnabled());
    expect(nativeAuthentication).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: '登录后添加' }));
    fireEvent.click(button);
    expect(nativeAuthentication).toHaveBeenCalledOnce();
    expect(nativeAuthentication).toHaveBeenCalledWith({
      optionsJSON: expect.objectContaining({
        rpId: 'localhost',
        challenge: 'dGVzdA',
      }),
    });
    await waitFor(() => expect(accepted).toHaveBeenCalledOnce());
    expect(finish).toHaveBeenCalledWith(
      { credentialJson: expect.any(String), rememberMe: false },
      { skipErrorHandler: true },
    );
    expect(
      sessionStorage.getItem('roncin.passkey.enroll-after-login'),
    ).toBeNull();
  });

  it('过期选项不得调用浏览器验证，重新准备后由用户重试', async () => {
    const clock = vi.spyOn(Date, 'now');
    const initial = Date.now();
    clock.mockReturnValue(initial);
    renderWithClient(<LoginProbe />);
    const button = screen.getByRole('button', { name: '免密登录' });
    await waitFor(() => expect(button).toBeEnabled());
    clock.mockReturnValue(initial + 400_000);
    fireEvent.click(button);
    await waitFor(() =>
      expect(screen.getByRole('alert')).toHaveTextContent('已过期'),
    );
    expect(nativeAuthentication).not.toHaveBeenCalled();
    expect(finish).not.toHaveBeenCalled();
    await waitFor(() => expect(begin).toHaveBeenCalledTimes(2));
    clock.mockRestore();
  });

  it('原生验证取消后显示明确错误，不自动登录或降级', async () => {
    nativeAuthentication.mockRejectedValueOnce(
      new DOMException('Cancelled', 'NotAllowedError'),
    );
    renderWithClient(<LoginProbe />);
    const button = screen.getByRole('button', { name: '免密登录' });
    await waitFor(() => expect(button).toBeEnabled());
    fireEvent.click(button);
    await waitFor(() =>
      expect(screen.getByRole('alert')).toHaveTextContent('未完成通行密钥验证'),
    );
    expect(finish).not.toHaveBeenCalled();
    expect(accepted).not.toHaveBeenCalled();
    expect(nativeAuthentication).toHaveBeenCalledOnce();
  });
  it('离开登录页时取消原生验证，迟到结果不得发送认证请求', async () => {
    let resolveCredential:
      | ((value: Awaited<ReturnType<typeof startAuthentication>>) => void)
      | undefined;
    nativeAuthentication.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          resolveCredential = resolve;
        }),
    );
    const { unmount } = renderWithClient(<LoginProbe />);
    const button = screen.getByRole('button', { name: '免密登录' });
    await waitFor(() => expect(button).toBeEnabled());
    fireEvent.click(button);
    expect(nativeAuthentication).toHaveBeenCalledOnce();
    unmount();
    expect(WebAuthnAbortService.cancelCeremony).toHaveBeenCalledOnce();
    await act(async () => {
      resolveCredential?.({
        id: 'late',
        rawId: 'late',
        response: { authenticatorData: '', clientDataJSON: '', signature: '' },
        clientExtensionResults: {},
        type: 'public-key',
      });
    });
    expect(finish).not.toHaveBeenCalled();
    expect(accepted).not.toHaveBeenCalled();
  });
  it('认证提交后离开页面，迟到成功不得继续旧页面登录流程', async () => {
    let resolveLogin:
      | ((value: API.FinishPasskeyLoginResponse) => void)
      | undefined;
    finish.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          resolveLogin = resolve;
        }),
    );
    const { unmount } = renderWithClient(<LoginProbe />);
    const button = screen.getByRole('button', { name: '免密登录' });
    await waitFor(() => expect(button).toBeEnabled());
    fireEvent.click(button);
    await waitFor(() => expect(finish).toHaveBeenCalledOnce());
    unmount();
    await act(async () => {
      resolveLogin?.({ success: true, data: { id: 'user-1' } });
    });
    expect(accepted).not.toHaveBeenCalled();
  });
  it('配置首次失败后可显式重取，不自动启动浏览器认证', async () => {
    getConfig.mockRejectedValueOnce(new Error('network failed'));
    renderWithClient(<LoginProbe />);
    await screen.findByText('无法获取通行密钥配置，请稍后重试');
    expect(begin).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: '重新准备' }));
    await waitFor(() =>
      expect(screen.getByRole('button', { name: '免密登录' })).toBeEnabled(),
    );
    expect(nativeAuthentication).not.toHaveBeenCalled();
  });
});
