import '@root/tests/passkey-login-disabled';
import { deferred, qrConfig } from '@root/tests/dingtalk-qr-support';
import { renderWithClient as render } from '@root/tests/queryClientTestUtils';
import {
  act,
  cleanup,
  fireEvent,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import Login from './index';

const {
  authServiceLoginMock,
  authServiceSwitchOrganizationMock,
  authServiceGetDingTalkLoginConfigMock,
  authServicePollDingTalkQRCodeMock,
  messageSuccessMock,
  messageErrorMock,
  setInitialStateMock,
  historyReplaceMock,
} = vi.hoisted(() => ({
  authServiceLoginMock: vi.fn(),
  authServiceSwitchOrganizationMock: vi.fn(),
  authServiceGetDingTalkLoginConfigMock: vi.fn(),
  authServicePollDingTalkQRCodeMock: vi.fn(),
  messageSuccessMock: vi.fn(),
  messageErrorMock: vi.fn(),
  setInitialStateMock: vi.fn(),
  historyReplaceMock: vi.fn(),
}));

vi.mock('react-helmet-async', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-helmet-async')>();
  return {
    ...actual,
    Helmet: ({ children }: { children?: React.ReactNode }) => children,
  };
});

vi.mock('@/router/history', () => ({
  history: { replace: historyReplaceMock },
}));

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({ setInitialState: setInitialStateMock }),
}));

vi.mock('antd', async (importOriginal) => {
  const actual = await importOriginal<typeof import('antd')>();
  const MockedApp = Object.assign(actual.App, {
    useApp: () => ({
      message: { success: messageSuccessMock, error: messageErrorMock },
    }),
  });
  return { ...actual, App: MockedApp };
});

vi.mock('@/services/roncin/authService', () => ({
  authServiceLogin: authServiceLoginMock,
  authServiceSwitchOrganization: authServiceSwitchOrganizationMock,
  authServiceGetDingTalkLoginConfig: authServiceGetDingTalkLoginConfigMock,
  authServicePollDingTalkQrCode: authServicePollDingTalkQRCodeMock,
}));

const singleOrgChoices: API.OrganizationChoice[] = [
  {
    organizationId: 'org-1',
    organizationName: '天津公司',
    organizationCode: 'TJ',
  },
];

const multiOrgChoices: API.OrganizationChoice[] = [
  {
    organizationId: 'org-1',
    organizationName: '天津公司',
    organizationCode: 'TJ',
  },
  {
    organizationId: 'org-2',
    organizationName: '北京公司',
    organizationCode: 'BJ',
  },
];

function currentUser(): API.CurrentUser {
  return {
    id: 'user-1',
    username: 'operator',
    displayName: '操作员',
    currentOrganization: { id: 'org-1', code: 'TJ', name: '天津公司' },
    organizations: [
      { id: 'org-1', code: 'TJ', name: '天津公司' },
      { id: 'org-2', code: 'BJ', name: '北京公司' },
    ],
  };
}

function mockLoginResponse(
  organizationChoices: API.OrganizationChoice[],
): void {
  authServiceLoginMock.mockResolvedValue({
    data: currentUser(),
    organizationChoices,
  });
}

async function submitLogin(rememberMe = true): Promise<void> {
  fireEvent.click(screen.getByRole('radio', { name: '账号登录' }));
  fireEvent.change(screen.getByPlaceholderText('钉钉手机号 / 本地账号'), {
    target: { value: 'operator' },
  });
  await waitFor(() =>
    expect(screen.getByRole('button', { name: '下一步' })).toBeEnabled(),
  );
  fireEvent.click(screen.getByRole('button', { name: '下一步' }));
  await waitFor(() =>
    expect(screen.getByPlaceholderText('请输入密码')).toBeVisible(),
  );
  fireEvent.change(screen.getByPlaceholderText('请输入密码'), {
    target: { value: 'secret' },
  });
  await waitFor(() =>
    expect(screen.getByRole('button', { name: /^登录$/ })).toBeEnabled(),
  );
  fireEvent.click(screen.getByRole('button', { name: /^登录$/ }));
  await waitFor(() => {
    expect(authServiceLoginMock).toHaveBeenCalledTimes(1);
  });
  expect(authServiceLoginMock).toHaveBeenCalledWith(
    { username: 'operator', password: 'secret', rememberMe },
    { skipErrorHandler: true },
  );
}

describe('Login', () => {
  let assignMock: ReturnType<typeof vi.fn<(url: string | URL) => void>>;

  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    authServicePollDingTalkQRCodeMock
      .mockReset()
      .mockImplementation(() => new Promise(() => {}));
    assignMock = vi.fn<(url: string | URL) => void>();
    vi.spyOn(window.location, 'assign').mockImplementation(assignMock);
    authServiceGetDingTalkLoginConfigMock.mockResolvedValue({
      data: { enabled: false },
    });
    window.history.replaceState({}, '', '/user/login');
    sessionStorage.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    cleanup();
  });

  it('登录失败时展示服务端业务提示，而非 Axios 默认状态码文案', async () => {
    authServiceLoginMock.mockRejectedValue({
      response: {
        status: 401,
        data: { success: false, code: 401, message: '用户名或密码错误' },
      },
    });

    render(
      <App>
        <Login />
      </App>,
    );
    await submitLogin();

    await waitFor(() => {
      expect(messageErrorMock).toHaveBeenCalledWith('用户名或密码错误');
    });
  });

  it('登录请求无服务端报文时回退通用失败提示', async () => {
    authServiceLoginMock.mockRejectedValue(new Error('Network Error'));

    render(
      <App>
        <Login />
      </App>,
    );
    await submitLogin();

    await waitFor(() => {
      expect(messageErrorMock).toHaveBeenCalledWith('登录失败，请稍后重试');
    });
  });

  it('密码认证期间与多组织选择视图均忽略隐藏二维码确认', async () => {
    const pending = deferred<{ data: { status: number; authCode: string } }>();
    authServicePollDingTalkQRCodeMock.mockReturnValue(pending.promise);
    authServiceGetDingTalkLoginConfigMock.mockResolvedValue(
      qrConfig('sample-state'),
    );
    let completeLogin: (response: {
      data: API.CurrentUser;
      organizationChoices: API.OrganizationChoice[];
    }) => void = () => {};
    authServiceLoginMock.mockReturnValue(
      new Promise((resolve) => {
        completeLogin = resolve;
      }),
    );

    render(
      <App>
        <Login />
      </App>,
    );
    const frame = await screen.findByRole('img', {
      name: '钉钉扫码登录二维码',
    });
    await submitLogin();
    expect(frame).not.toBeVisible();
    expect(screen.getByRole('radio', { name: '扫码登录' })).toBeDisabled();
    expect(screen.getByRole('radio', { name: '账号登录' })).toBeDisabled();
    expect(screen.getByRole('radio', { name: '账号登录' })).toBeChecked();
    await act(async () =>
      pending.resolve({ data: { status: 3, authCode: 'hidden-code' } }),
    );
    expect(historyReplaceMock).not.toHaveBeenCalled();
    await act(async () =>
      completeLogin({
        data: currentUser(),
        organizationChoices: multiOrgChoices,
      }),
    );
    expect(await screen.findByText('选择进入的组织')).toBeInTheDocument();

    expect(historyReplaceMock).not.toHaveBeenCalled();
    expect(frame).not.toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: '进入天津公司' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: '进入北京公司' }),
    ).toBeInTheDocument();
    // 未选择组织前不进入应用
    expect(assignMock).not.toHaveBeenCalled();
  });

  it('多组织用户选择本次会话组织：直接进入，不发起切换', async () => {
    mockLoginResponse(multiOrgChoices);

    render(
      <App>
        <Login />
      </App>,
    );
    await submitLogin();

    fireEvent.click(
      await screen.findByRole('button', { name: '进入天津公司' }),
    );

    await waitFor(() => {
      expect(assignMock).toHaveBeenCalledWith('/');
    });
    expect(authServiceSwitchOrganizationMock).not.toHaveBeenCalled();
    expect(messageSuccessMock).toHaveBeenCalledWith('登录成功');
  });

  it('多组织用户选择其他组织：经切换接口换发会话后进入', async () => {
    mockLoginResponse(multiOrgChoices);
    authServiceSwitchOrganizationMock.mockResolvedValue({
      data: { ...currentUser(), currentOrganization: { id: 'org-2' } },
    });

    render(
      <App>
        <Login />
      </App>,
    );
    await submitLogin();

    fireEvent.click(
      await screen.findByRole('button', { name: '进入北京公司' }),
    );

    await waitFor(() => {
      expect(assignMock).toHaveBeenCalledWith('/');
    });
    expect(authServiceSwitchOrganizationMock).toHaveBeenCalledWith(
      { organizationId: 'org-2' },
      { skipErrorHandler: true },
    );
  });

  it('登录响应未携带候选列表时回退 principal organizations 判定组织数', async () => {
    authServiceLoginMock.mockResolvedValue({ data: currentUser() });

    render(
      <App>
        <Login />
      </App>,
    );
    await submitLogin();

    expect(await screen.findByText('选择进入的组织')).toBeInTheDocument();
    // 会话组织点击后直接进入，卡片不显示默认标记
    fireEvent.click(screen.getByRole('button', { name: '进入天津公司' }));
    await waitFor(() => {
      expect(assignMock).toHaveBeenCalledWith('/');
    });
    expect(authServiceSwitchOrganizationMock).not.toHaveBeenCalled();
  });

  it('单组织用户登录后直接进入，不出现组织选择视图', async () => {
    mockLoginResponse(singleOrgChoices);

    render(
      <App>
        <Login />
      </App>,
    );
    await submitLogin();

    await waitFor(() => {
      expect(assignMock).toHaveBeenCalledWith('/');
    });
    expect(screen.queryByText('选择进入的组织')).not.toBeInTheDocument();
    expect(authServiceSwitchOrganizationMock).not.toHaveBeenCalled();
    expect(messageSuccessMock).toHaveBeenCalledWith('登录成功');
  });

  it('默认扫码并保留账号方式切换，不展示额外底部说明', async () => {
    authServiceGetDingTalkLoginConfigMock.mockResolvedValue(
      qrConfig('sample-state'),
    );
    window.history.replaceState({}, '', '/user/login');

    render(
      <App>
        <Login />
      </App>,
    );

    expect(
      await screen.findByRole('img', { name: '钉钉扫码登录二维码' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('radio', { name: '账号登录' })).not.toBeChecked();
    expect(screen.getByRole('radio', { name: '扫码登录' })).toBeChecked();
    expect(screen.queryByPlaceholderText('请输入密码')).not.toBeInTheDocument();
    expect(
      screen.queryByText('使用前请联系管理员同步人员并分配组织和角色'),
    ).not.toBeInTheDocument();
    expect(screen.queryByText('使用钉钉扫码注册')).not.toBeInTheDocument();
    const iframe = screen.getByRole('img', { name: '钉钉扫码登录二维码' });
    for (let count = 0; count < 3; count++) {
      fireEvent.click(screen.getByRole('radio', { name: '账号登录' }));
      expect(screen.getByRole('radio', { name: '账号登录' })).toBeChecked();
      expect(
        screen.getByPlaceholderText('钉钉手机号 / 本地账号'),
      ).toBeVisible();
      expect(iframe).not.toBeVisible();
      expect(iframe.closest('button')).not.toBeVisible();
      fireEvent.click(screen.getByRole('radio', { name: '扫码登录' }));
      expect(screen.getByRole('radio', { name: '扫码登录' })).toBeChecked();
      expect(screen.getByRole('img', { name: '钉钉扫码登录二维码' })).toBe(
        iframe,
      );
      expect(iframe).toBeVisible();
      expect(screen.queryByText('正在加载钉钉二维码')).not.toBeInTheDocument();
      expect(
        screen.queryByText('正在获取钉钉登录配置'),
      ).not.toBeInTheDocument();
    }
    fireEvent.keyDown(screen.getByRole('radio', { name: '扫码登录' }), {
      key: 'ArrowRight',
    });
    expect(screen.getByRole('radio', { name: '账号登录' })).toBeChecked();
    expect(screen.getByPlaceholderText('钉钉手机号 / 本地账号')).toBeVisible();
    fireEvent.keyDown(screen.getByRole('radio', { name: '账号登录' }), {
      key: 'ArrowLeft',
    });
    expect(screen.getByRole('radio', { name: '扫码登录' })).toBeChecked();
    expect(screen.getByRole('img', { name: '钉钉扫码登录二维码' })).toBe(
      iframe,
    );
    expect(authServiceGetDingTalkLoginConfigMock).toHaveBeenCalledOnce();
  });

  it('钉钉扫码保留安全回跳地址并进入本站认证回调', async () => {
    authServicePollDingTalkQRCodeMock.mockResolvedValueOnce({
      data: { status: 3, authCode: 'sample-code' },
    });
    authServiceGetDingTalkLoginConfigMock.mockResolvedValue(
      qrConfig('sample-state'),
    );
    window.history.replaceState({}, '', '/user/login?redirect=%2Fwelcome');

    render(
      <App>
        <Login />
      </App>,
    );

    await waitFor(() => {
      expect(historyReplaceMock).toHaveBeenCalledWith(
        '/user/login/dingtalk/callback?authCode=sample-code&state=sample-state',
      );
    });
    expect(sessionStorage.getItem('dingtalk_login_redirect')).toBe('/welcome');
    expect(authServiceGetDingTalkLoginConfigMock).toHaveBeenCalledTimes(1);
  });

  it('扫码配置失败仍可进入管理员密码，并可返回扫码重试', async () => {
    authServiceGetDingTalkLoginConfigMock.mockRejectedValueOnce(
      new Error('获取钉钉登录配置失败，请重试'),
    );
    render(
      <App>
        <Login />
      </App>,
    );
    expect(await screen.findByText('二维码加载失败')).toBeVisible();
    fireEvent.click(screen.getByRole('radio', { name: '账号登录' }));
    expect(screen.getByRole('button', { name: '下一步' })).toHaveAttribute(
      'type',
      'submit',
    );
    expect(screen.getByPlaceholderText('钉钉手机号 / 本地账号')).toBeVisible();
    fireEvent.click(screen.getByRole('radio', { name: '扫码登录' }));
    expect(
      await screen.findByText('钉钉登录暂未启用，请联系管理员'),
    ).toBeVisible();
    expect(screen.queryByPlaceholderText('请输入密码')).not.toBeInTheDocument();
    expect(authServiceGetDingTalkLoginConfigMock).toHaveBeenCalledTimes(2);
  });
  it('账号首步只切换密码视图，不提前请求账号或执行登录', async () => {
    render(
      <App>
        <Login />
      </App>,
    );
    fireEvent.click(screen.getByRole('radio', { name: '账号登录' }));
    fireEvent.change(screen.getByPlaceholderText('钉钉手机号 / 本地账号'), {
      target: { value: 'operator' },
    });
    await waitFor(() =>
      expect(screen.getByRole('button', { name: '下一步' })).toBeEnabled(),
    );
    fireEvent.click(screen.getByRole('button', { name: '下一步' }));
    await waitFor(() =>
      expect(screen.getByPlaceholderText('请输入密码')).toBeVisible(),
    );
    expect(authServiceLoginMock).not.toHaveBeenCalled();
    expect(screen.getByText('operator')).toBeVisible();
    fireEvent.click(screen.getByRole('button', { name: '修改账号' }));
    expect(screen.getByPlaceholderText('钉钉手机号 / 本地账号')).toHaveValue(
      'operator',
    );
  });

  it('取消自动登录后密码请求携带rememberMe false', async () => {
    mockLoginResponse(singleOrgChoices);
    render(
      <App>
        <Login />
      </App>,
    );
    fireEvent.click(screen.getByRole('checkbox', { name: '自动登录' }));
    await submitLogin(false);
  });
});
