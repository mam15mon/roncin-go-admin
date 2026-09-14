import {
  cleanup,
  fireEvent,
  render,
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
  authServiceGetWeComLoginConfigMock,
  authServiceGetDingTalkLoginConfigMock,
  authServiceGetDingTalkInvitationInfoMock,
  messageSuccessMock,
  messageErrorMock,
  setInitialStateMock,
} = vi.hoisted(() => ({
  authServiceLoginMock: vi.fn(),
  authServiceSwitchOrganizationMock: vi.fn(),
  authServiceGetWeComLoginConfigMock: vi.fn(),
  authServiceGetDingTalkLoginConfigMock: vi.fn(),
  authServiceGetDingTalkInvitationInfoMock: vi.fn(),
  messageSuccessMock: vi.fn(),
  messageErrorMock: vi.fn(),
  setInitialStateMock: vi.fn(),
}));

vi.mock('@umijs/max', () => ({
  Helmet: ({ children }: { children?: React.ReactNode }) => children,
  Link: ({ children }: { children?: React.ReactNode }) => <a>{children}</a>,
  useModel: () => ({ setInitialState: setInitialStateMock }),
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
  authServiceGetWeComLoginConfig: authServiceGetWeComLoginConfigMock,
  authServiceGetDingTalkLoginConfig: authServiceGetDingTalkLoginConfigMock,
  authServiceGetDingTalkInvitationInfo:
    authServiceGetDingTalkInvitationInfoMock,
}));

const singleOrgChoices: API.OrganizationChoice[] = [
  {
    organizationId: 'org-1',
    organizationName: '天津公司',
    organizationCode: 'TJ',
    isDefault: true,
  },
];

const multiOrgChoices: API.OrganizationChoice[] = [
  {
    organizationId: 'org-1',
    organizationName: '天津公司',
    organizationCode: 'TJ',
    isDefault: true,
  },
  {
    organizationId: 'org-2',
    organizationName: '北京公司',
    organizationCode: 'BJ',
    isDefault: false,
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
  fireEvent.change(screen.getByPlaceholderText('用户名 / 邮箱'), {
    target: { value: 'operator' },
  });
  fireEvent.change(screen.getByPlaceholderText('请输入密码'), {
    target: { value: 'secret' },
  });
  fireEvent.click(screen.getByRole('button', { name: /登录/ }));
  await waitFor(() => {
    expect(authServiceLoginMock).toHaveBeenCalledTimes(1);
  });
}

describe('Login', () => {
  let assignMock: ReturnType<typeof vi.fn<(url: string | URL) => void>>;

  beforeEach(() => {
    vi.clearAllMocks();
    assignMock = vi.fn<(url: string | URL) => void>();
    vi.spyOn(window.location, 'assign').mockImplementation(assignMock);
    authServiceGetWeComLoginConfigMock.mockResolvedValue({
      data: { enabled: false },
    });
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

  it('多组织用户选择默认组织：直接进入，不发起切换', async () => {
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
    // 会话组织（默认）高亮预选，点击直接进入
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

  it('携带邀请 Token 的落地页展示专属欢迎与防呆警示', async () => {
    authServiceGetDingTalkLoginConfigMock.mockResolvedValue({
      data: { enabled: true, authorizeUrl: 'https://dingtalk.login/auth' },
    });
    authServiceGetDingTalkInvitationInfoMock.mockResolvedValue({
      data: {
        organizationName: '成都分公司',
        inviterName: '李经理',
        expiresAt: '2030-01-01T00:00:00Z',
      },
    });
    window.history.replaceState({}, '', '/user/login?invite=valid-token');

    render(
      <App>
        <Login />
      </App>,
    );

    expect(
      await screen.findByText('【成都分公司】专属邀请'),
    ).toBeInTheDocument();
    // PRD 3.2.3：防呆警示必须醒目呈现（warning 级 Alert）
    expect(
      screen.getByText(
        '若您属于其他分公司（如成都、深圳），请勿加入，请向所属分公司主管索取专属码',
      ),
    ).toBeInTheDocument();
    expect(screen.getByText(/邀请人：李经理/)).toBeInTheDocument();
    expect(
      await screen.findByRole('button', { name: /使用钉钉加入【成都分公司】/ }),
    ).toBeInTheDocument();
    expect(screen.queryByText('使用钉钉扫码注册')).not.toBeInTheDocument();
  });

  it('钉钉启用时，普通登录页展示快捷入职按钮与提示文案，不展示注册链接', async () => {
    authServiceGetDingTalkLoginConfigMock.mockResolvedValue({
      data: { enabled: true, authorizeUrl: 'https://dingtalk.login/auth' },
    });
    window.history.replaceState({}, '', '/user/login');

    render(
      <App>
        <Login />
      </App>,
    );

    expect(
      await screen.findByRole('button', { name: /钉钉登录/ }),
    ).toBeInTheDocument();
    expect(screen.getByText(/快捷入职/)).toBeInTheDocument();
    expect(
      screen.getByText('首次使用钉钉扫码将自动提交入职审批申请'),
    ).toBeInTheDocument();
    expect(screen.queryByText('使用钉钉扫码注册')).not.toBeInTheDocument();
  });

  it('邀请 Token 失效时仅呈现失效提示，不展示欢迎卡片', async () => {
    authServiceGetDingTalkInvitationInfoMock.mockRejectedValue(
      new Error('邀请链接已失效或已过期，请联系主管重新获取'),
    );
    window.history.replaceState({}, '', '/user/login?invite=bad-token');

    render(
      <App>
        <Login />
      </App>,
    );

    expect(await screen.findByText('邀请链接已失效')).toBeInTheDocument();
    expect(
      screen.queryByText('【成都分公司】专属邀请'),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(
        '若您属于其他分公司（如成都、深圳），请勿加入，请向所属分公司主管索取专属码',
      ),
    ).not.toBeInTheDocument();
  });

  it('直接访问普通登录页（无 ?invite=）时清空 sessionStorage 残留邀请 Token', () => {
    sessionStorage.setItem('dingtalk_invitation_token', 'stale-token-xyz');
    window.history.replaceState({}, '', '/user/login');

    render(
      <App>
        <Login />
      </App>,
    );

    expect(sessionStorage.getItem('dingtalk_invitation_token')).toBeNull();
  });
});
