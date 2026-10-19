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
  switchOrganization: vi.fn(),
}));
vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({ setInitialState: vi.fn() }),
}));
vi.mock('@/services/roncin/authService', () => ({
  authServiceGetDingTalkEmbeddedConfig: mocks.config,
  authServiceDingTalkEmbeddedLogin: mocks.login,
  authServiceGetDingTalkLoginConfig: mocks.qrConfig,
  authServiceSwitchOrganization: mocks.switchOrganization,
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

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: unknown) => void;
  const promise = new Promise<T>((accept, fail) => {
    resolve = accept;
    reject = fail;
  });
  return { promise, resolve, reject };
}

it('SDK 等待阶段仍可切账号，迟到授权码不换码', async () => {
  const pending = deferred<string>();
  mocks.code.mockReturnValueOnce(pending.promise);
  renderWithClient(<Login />);
  await waitFor(() => expect(mocks.code).toHaveBeenCalledOnce());
  expect(screen.getByRole('radio', { name: '账号登录' })).not.toBeDisabled();
  fireEvent.click(screen.getByRole('button', { name: '使用账号登录' }));
  expect(screen.getByPlaceholderText('钉钉手机号 / 本地账号')).toBeVisible();
  await act(async () => pending.resolve('late-code'));
  expect(mocks.login).not.toHaveBeenCalled();
});

it('换码发出前同步拦截同 tick 切换，HTTP 在途禁用账号按钮与顶部方式', async () => {
  const pending = deferred<API.DingTalkEmbeddedLoginResponse>();
  mocks.code.mockResolvedValueOnce('code');
  mocks.login.mockImplementationOnce(() => {
    // 请求调用发生时 DOM 可能尚未提交 disabled，回调同步门禁仍必须生效。
    fireEvent.click(screen.getByRole('radio', { name: '账号登录' }));
    fireEvent.click(screen.getByRole('button', { name: '使用账号登录' }));
    return pending.promise;
  });
  renderWithClient(<Login />);
  await waitFor(() => expect(mocks.login).toHaveBeenCalledOnce());
  expect(
    screen.queryByPlaceholderText('钉钉手机号 / 本地账号'),
  ).not.toBeInTheDocument();
  expect(screen.getByRole('radio', { name: '账号登录' })).toBeDisabled();
  expect(screen.getByRole('button', { name: '使用账号登录' })).toBeDisabled();
  await act(async () => pending.reject(new Error('网络失败')));
  expect(await screen.findByText('钉钉免登未完成')).toBeVisible();
  expect(screen.getByRole('radio', { name: '账号登录' })).not.toBeDisabled();
  expect(
    screen.getByRole('button', { name: '使用账号登录' }),
  ).not.toBeDisabled();
  fireEvent.click(screen.getByRole('radio', { name: '账号登录' }));
  expect(screen.getByPlaceholderText('钉钉手机号 / 本地账号')).toBeVisible();
});

it('人员待处理响应落定后恢复切账号', async () => {
  mocks.code.mockResolvedValueOnce('code');
  mocks.login.mockResolvedValueOnce({ data: { status: 3 } });
  renderWithClient(<Login />);
  expect(await screen.findByText('尚未分配组织')).toBeVisible();
  expect(screen.getByRole('radio', { name: '账号登录' })).not.toBeDisabled();
  fireEvent.click(screen.getByRole('button', { name: '使用账号登录' }));
  expect(screen.getByPlaceholderText('钉钉手机号 / 本地账号')).toBeVisible();
});

it('多组织选择全程锁住顶部方式，轮转在途不能开始另一登录，卸载迟到不导航', async () => {
  const pending = deferred<{ data: { id: string } }>();
  const replace = vi
    .spyOn(window.location, 'replace')
    .mockImplementation(() => {});
  mocks.code.mockResolvedValueOnce('code');
  mocks.login.mockResolvedValueOnce({
    data: {
      status: 1,
      currentUser: {
        id: 'user',
        currentOrganization: { id: 'org-a' },
        organizations: [
          { id: 'org-a', name: '天津公司', code: 'TJ' },
          { id: 'org-b', name: '北京公司', code: 'BJ' },
        ],
      },
    },
  });
  mocks.switchOrganization.mockReturnValueOnce(pending.promise);
  const view = renderWithClient(<Login />);
  expect(await screen.findByText('选择进入的组织')).toBeVisible();
  expect(screen.getByRole('radio', { name: '账号登录' })).toBeDisabled();
  fireEvent.click(screen.getByRole('radio', { name: '账号登录' }));
  expect(
    screen.queryByPlaceholderText('钉钉手机号 / 本地账号'),
  ).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: '进入北京公司' }));
  await waitFor(() => expect(mocks.switchOrganization).toHaveBeenCalledOnce());
  expect(screen.getByRole('radio', { name: '账号登录' })).toBeDisabled();
  expect(screen.getByRole('button', { name: '进入天津公司' })).toBeDisabled();
  view.unmount();
  await act(async () => pending.resolve({ data: { id: 'user' } }));
  expect(replace).not.toHaveBeenCalled();
});
