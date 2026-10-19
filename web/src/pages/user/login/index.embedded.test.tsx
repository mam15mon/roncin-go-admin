import { renderWithClient } from '@root/tests/queryClientTestUtils';
import {
  act,
  cleanup,
  fireEvent,
  screen,
  waitFor,
} from '@testing-library/react';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import Login from './index';

const mocks = vi.hoisted(() => ({
  config: vi.fn(),
  code: vi.fn(),
  login: vi.fn(),
  qrConfig: vi.fn(),
}));
vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({ setInitialState: vi.fn() }),
}));
vi.mock('@/services/roncin/authService', () => ({
  authServiceGetDingTalkEmbeddedConfig: mocks.config,
  authServiceDingTalkEmbeddedLogin: mocks.login,
  authServiceGetDingTalkLoginConfig: mocks.qrConfig,
}));
vi.mock('./dingtalkEmbeddedSdk', () => ({
  requestEmbeddedAuthCode: mocks.code,
}));
vi.mock('./components/animated-characters', () => ({
  AnimatedCharacters: () => null,
}));
vi.mock('react-helmet-async', () => ({ Helmet: () => null }));
beforeEach(() => {
  vi.spyOn(navigator, 'userAgent', 'get').mockReturnValue(
    'Mozilla dInGtAlK/7.0',
  );
  mocks.config.mockResolvedValue({
    data: {
      enabled: true,
      corpId: 'corp',
      state: 'state',
      expiresAt: new Date(Date.now() + 300000).toISOString(),
    },
  });
  mocks.code.mockRejectedValue(new Error('钉钉客户端认证失败，请重试'));
});
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  vi.clearAllMocks();
});
it('钉钉内默认免登，切账号后不挂扫码、不请求扫码配置', async () => {
  renderWithClient(<Login />);
  expect(await screen.findByText('钉钉客户端认证失败，请重试')).toBeVisible();
  expect(mocks.config).toHaveBeenCalledOnce();
  expect(screen.queryByTitle('钉钉扫码登录二维码')).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole('radio', { name: '账号登录' }));
  expect(screen.getByPlaceholderText('钉钉手机号 / 本地账号')).toBeVisible();
  expect(screen.queryByLabelText('钉钉客户端免登')).not.toBeInTheDocument();
  await act(async () => {});
  expect(mocks.qrConfig).not.toHaveBeenCalled();
  expect(mocks.login).not.toHaveBeenCalled();
});
it('明确切回钉钉免登建立新尝试，保留账号入口', async () => {
  renderWithClient(<Login />);
  await screen.findByText('钉钉客户端认证失败，请重试');
  fireEvent.click(screen.getByRole('button', { name: '使用账号登录' }));
  fireEvent.click(screen.getByRole('radio', { name: '钉钉免登' }));
  await waitFor(() => expect(mocks.code).toHaveBeenCalledTimes(2));
  expect(await screen.findByText('钉钉客户端认证失败，请重试')).toBeVisible();
  expect(mocks.qrConfig).not.toHaveBeenCalled();
});
