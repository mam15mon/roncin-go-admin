import { renderWithClient as render } from '@root/tests/queryClientTestUtils';
import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import { DetachedBrowserFrame } from 'happy-dom';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import Login from './index';

const {
  authServiceLoginMock,
  authServiceSwitchOrganizationMock,
  authServiceGetDingTalkLoginConfigMock,
  messageSuccessMock,
  messageErrorMock,
  setInitialStateMock,
  historyReplaceMock,
} = vi.hoisted(() => ({
  authServiceLoginMock: vi.fn(),
  authServiceSwitchOrganizationMock: vi.fn(),
  authServiceGetDingTalkLoginConfigMock: vi.fn(),
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

async function submitLogin(): Promise<void> {
  fireEvent.click(screen.getByRole('button', { name: '管理员账号登录' }));
  fireEvent.change(screen.getByPlaceholderText('用户名 / 邮箱'), {
    target: { value: 'operator' },
  });
  fireEvent.change(screen.getByPlaceholderText('请输入密码'), {
    target: { value: 'secret' },
  });
  fireEvent.click(screen.getByRole('button', { name: /^登录$/ }));
  await waitFor(() => {
    expect(authServiceLoginMock).toHaveBeenCalledTimes(1);
  });
}

describe('Login', () => {
  let assignMock: ReturnType<typeof vi.fn<(url: string | URL) => void>>;

  beforeEach(() => {
    vi.clearAllMocks();
    // 扫码用例使用假配置与消息，保留 iframe/window 并阻止真实网络导航。
    vi.spyOn(DetachedBrowserFrame.prototype, 'goto').mockImplementation(
      () => new Promise(() => {}),
    );
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

  it('多组织用户登录后先出现组织选择视图', async () => {
    mockLoginResponse(multiOrgChoices);

    render(
      <App>
        <Login />
      </App>,
    );
    await submitLogin();

    expect(await screen.findByText('选择进入的组织')).toBeInTheDocument();
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

  it('钉钉启用时提示管理员开通，不展示注册链接', async () => {
    authServiceGetDingTalkLoginConfigMock.mockResolvedValue({
      data: {
        enabled: true,
        authorizeUrl: `https://login.dingtalk.com/oauth2/auth?client_id=app&scope=openid&response_type=code&state=sample-state&redirect_uri=${encodeURIComponent(`${window.location.origin}/user/login/dingtalk/callback`)}`,
        expiresAt: new Date(Date.now() + 300000).toISOString(),
      },
    });
    window.history.replaceState({}, '', '/user/login');

    render(
      <App>
        <Login />
      </App>,
    );

    expect(await screen.findByTitle('钉钉扫码登录二维码')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: '管理员账号登录' }),
    ).toBeInTheDocument();
    expect(screen.queryByPlaceholderText('请输入密码')).not.toBeInTheDocument();
    expect(
      screen.getByText('使用前请联系管理员同步人员并分配组织和角色'),
    ).toBeInTheDocument();
    expect(screen.queryByText('使用钉钉扫码注册')).not.toBeInTheDocument();
  });

  it('钉钉扫码保留安全回跳地址并进入本站认证回调', async () => {
    authServiceGetDingTalkLoginConfigMock.mockResolvedValue({
      data: {
        enabled: true,
        authorizeUrl: `https://login.dingtalk.com/oauth2/auth?client_id=app&scope=openid&response_type=code&state=sample-state&redirect_uri=${encodeURIComponent(`${window.location.origin}/user/login/dingtalk/callback`)}`,
        expiresAt: new Date(Date.now() + 300000).toISOString(),
      },
    });
    window.history.replaceState({}, '', '/user/login?redirect=%2Fwelcome');

    render(
      <App>
        <Login />
      </App>,
    );

    const iframe = (await screen.findByTitle(
      '钉钉扫码登录二维码',
    )) as HTMLIFrameElement;
    fireEvent(
      window,
      new MessageEvent('message', {
        origin: 'https://login.dingtalk.com',
        source: iframe.contentWindow,
        data: {
          success: true,
          redirectUrl: `${window.location.origin}/user/login/dingtalk/callback?authCode=sample-code&state=sample-state`,
        },
      }),
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
    expect(
      await screen.findByText('获取钉钉登录配置失败，请重试'),
    ).toBeVisible();
    fireEvent.click(screen.getByRole('button', { name: '管理员账号登录' }));
    expect(screen.getByRole('button', { name: '登录' })).toHaveAttribute(
      'type',
      'submit',
    );
    expect(screen.getByPlaceholderText('请输入密码')).toBeVisible();
    fireEvent.click(screen.getByRole('button', { name: '返回钉钉扫码登录' }));
    expect(
      await screen.findByText('钉钉登录暂未启用，请联系管理员'),
    ).toBeVisible();
    expect(screen.queryByPlaceholderText('请输入密码')).not.toBeInTheDocument();
    expect(authServiceGetDingTalkLoginConfigMock).toHaveBeenCalledTimes(2);
  });
});
