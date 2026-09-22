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
import DingTalkCallback from './dingtalk-callback';

const {
  dingTalkLogin,
  registerDingTalkUser,
  switchOrganization,
  getInvitationInfo,
  messageSuccessMock,
} = vi.hoisted(() => ({
  dingTalkLogin: vi.fn(),
  registerDingTalkUser: vi.fn(),
  switchOrganization: vi.fn(),
  getInvitationInfo: vi.fn(),
  messageSuccessMock: vi.fn(),
}));

vi.mock('react-helmet-async', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-helmet-async')>();
  return {
    ...actual,
    Helmet: ({ children }: { children?: React.ReactNode }) => children,
  };
});

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({ setInitialState: vi.fn() }),
}));

vi.mock('antd', async (importOriginal) => {
  const actual = await importOriginal<typeof import('antd')>();
  const MockedApp = Object.assign(actual.App, {
    useApp: () => ({
      message: { success: messageSuccessMock },
    }),
  });
  return { ...actual, App: MockedApp };
});

vi.mock('@/services/roncin/authService', () => ({
  authServiceDingTalkLogin: dingTalkLogin,
  authServiceRegisterDingTalkUser: registerDingTalkUser,
  authServiceSwitchOrganization: switchOrganization,
  authServiceGetDingTalkInvitationInfo: getInvitationInfo,
}));

function authenticatedUser(
  organizations: Array<{ id: string; code: string; name: string }>,
  currentOrganizationId: string,
): API.DingTalkLoginResult {
  return {
    status: 1,
    currentUser: {
      id: 'user-1',
      displayName: '张三',
      currentOrganization: organizations.find(
        (organization) => organization.id === currentOrganizationId,
      ),
      organizations,
    },
  };
}

describe('DingTalkCallback', () => {
  let replaceMock: ReturnType<typeof vi.fn<(url: string | URL) => void>>;

  beforeEach(() => {
    sessionStorage.clear();
    window.history.replaceState(
      {},
      '',
      '/user/login/dingtalk/callback?authCode=code&state=state',
    );
    replaceMock = vi.fn();
    vi.spyOn(window.location, 'replace').mockImplementation(replaceMock);
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
    vi.restoreAllMocks();
  });

  it('一次身份验证后由人员确认注册，不再重复扫码', async () => {
    dingTalkLogin.mockResolvedValueOnce({
      data: {
        status: 2,
        displayName: '张三',
      },
    });
    registerDingTalkUser.mockResolvedValueOnce({
      data: { displayName: '张三', status: 'PENDING' },
    });

    render(
      <App>
        <DingTalkCallback />
      </App>,
    );

    expect(await screen.findByText('确认注册')).toBeInTheDocument();
    expect(screen.getByText(/已确认 张三 属于本企业/)).toBeInTheDocument();
    // 无可选目标公司时不渲染组织选择。
    expect(screen.queryByText('要加入的公司')).not.toBeInTheDocument();

    fireEvent.click(screen.getByText('确认注册'));

    expect(await screen.findByText('入职或返聘申请已提交')).toBeInTheDocument();
    expect(registerDingTalkUser).toHaveBeenCalledWith(
      {},
      { skipErrorHandler: true },
    );
  });

  it('多个候选公司时可自选目标公司，注册请求携带组织 ID', async () => {
    dingTalkLogin.mockResolvedValueOnce({
      data: {
        status: 2,
        displayName: '李四',
        registrationOrganizations: [
          {
            organizationId: 'org-cd',
            organizationName: '成都分公司',
            organizationCode: 'CD',
          },
          {
            organizationId: 'org-tj',
            organizationName: '天津分公司',
            organizationCode: 'TJ',
          },
        ],
      },
    });
    registerDingTalkUser.mockResolvedValueOnce({
      data: { displayName: '李四', status: 'PENDING' },
    });

    render(
      <App>
        <DingTalkCallback />
      </App>,
    );

    expect(await screen.findByText('要加入的公司')).toBeInTheDocument();
    expect(screen.getByText('默认（系统管理审批）')).toBeInTheDocument();

    // 选择成都分公司后确认注册。
    fireEvent.mouseDown(screen.getByRole('combobox'));
    const option = await screen.findByText('成都分公司 (CD)');
    fireEvent.click(option);

    fireEvent.click(screen.getByText('确认注册'));

    expect(await screen.findByText('入职或返聘申请已提交')).toBeInTheDocument();
    expect(registerDingTalkUser).toHaveBeenCalledWith(
      { organizationId: 'org-cd' },
      { skipErrorHandler: true },
    );
  });

  it('多个候选公司时不选择则走系统管理收口注册', async () => {
    dingTalkLogin.mockResolvedValueOnce({
      data: {
        status: 2,
        displayName: '王五',
        registrationOrganizations: [
          {
            organizationId: 'org-cd',
            organizationName: '成都分公司',
            organizationCode: 'CD',
          },
          {
            organizationId: 'org-tj',
            organizationName: '天津分公司',
            organizationCode: 'TJ',
          },
        ],
      },
    });
    registerDingTalkUser.mockResolvedValueOnce({
      data: { displayName: '王五', status: 'PENDING' },
    });

    render(
      <App>
        <DingTalkCallback />
      </App>,
    );

    expect(await screen.findByText('要加入的公司')).toBeInTheDocument();

    fireEvent.click(screen.getByText('确认注册'));

    expect(await screen.findByText('入职或返聘申请已提交')).toBeInTheDocument();
    expect(registerDingTalkUser).toHaveBeenCalledWith(
      {},
      { skipErrorHandler: true },
    );
  });

  it('单一候选公司维持现状，不渲染组织选择', async () => {
    dingTalkLogin.mockResolvedValueOnce({
      data: {
        status: 2,
        displayName: '赵六',
        registrationOrganizations: [
          {
            organizationId: 'org-cd',
            organizationName: '成都分公司',
            organizationCode: 'CD',
          },
        ],
      },
    });
    registerDingTalkUser.mockResolvedValueOnce({
      data: { displayName: '赵六', status: 'PENDING' },
    });

    render(
      <App>
        <DingTalkCallback />
      </App>,
    );

    expect(await screen.findByText('确认注册')).toBeInTheDocument();
    expect(screen.queryByText('要加入的公司')).not.toBeInTheDocument();

    fireEvent.click(screen.getByText('确认注册'));

    expect(await screen.findByText('入职或返聘申请已提交')).toBeInTheDocument();
    expect(registerDingTalkUser).toHaveBeenCalledWith(
      {},
      { skipErrorHandler: true },
    );
  });

  it('单组织用户认证后直接进入应用', async () => {
    sessionStorage.setItem('dingtalk_login_redirect', '/orders');
    dingTalkLogin.mockResolvedValueOnce({
      data: authenticatedUser(
        [{ id: 'org-1', code: 'TJ', name: '天津公司' }],
        'org-1',
      ),
    });

    render(
      <App>
        <DingTalkCallback />
      </App>,
    );

    await waitFor(() => {
      expect(replaceMock).toHaveBeenCalledWith('/orders');
    });
    expect(screen.queryByText('选择进入的组织')).not.toBeInTheDocument();
    expect(switchOrganization).not.toHaveBeenCalled();
  });

  it('多组织用户认证后先选组织，选其他组织经切换进入', async () => {
    sessionStorage.setItem('dingtalk_login_redirect', '/welcome');
    const organizations = [
      { id: 'org-1', code: 'TJ', name: '天津公司' },
      { id: 'org-2', code: 'BJ', name: '北京公司' },
    ];
    dingTalkLogin.mockResolvedValueOnce({
      data: authenticatedUser(organizations, 'org-1'),
    });
    switchOrganization.mockResolvedValueOnce({
      data: {
        id: 'user-1',
        currentOrganization: { id: 'org-2', code: 'BJ', name: '北京公司' },
      },
    });

    render(
      <App>
        <DingTalkCallback />
      </App>,
    );

    expect(await screen.findByText('选择进入的组织')).toBeInTheDocument();
    expect(replaceMock).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole('button', { name: '进入北京公司' }));

    await waitFor(() => {
      expect(replaceMock).toHaveBeenCalledWith('/welcome');
    });
    expect(switchOrganization).toHaveBeenCalledWith(
      { organizationId: 'org-2' },
      { skipErrorHandler: true },
    );
  });

  it('多组织用户选默认组织不发起切换', async () => {
    sessionStorage.setItem('dingtalk_login_redirect', '/welcome');
    const organizations = [
      { id: 'org-1', code: 'TJ', name: '天津公司' },
      { id: 'org-2', code: 'BJ', name: '北京公司' },
    ];
    dingTalkLogin.mockResolvedValueOnce({
      data: authenticatedUser(organizations, 'org-1'),
    });

    render(
      <App>
        <DingTalkCallback />
      </App>,
    );

    fireEvent.click(
      await screen.findByRole('button', { name: '进入天津公司' }),
    );

    await waitFor(() => {
      expect(replaceMock).toHaveBeenCalledWith('/welcome');
    });
    expect(switchOrganization).not.toHaveBeenCalled();
    expect(messageSuccessMock).toHaveBeenCalledWith('钉钉登录成功');
  });

  it('携带 invitationToken 时不渲染公司选择，直接绑定专属通道并携带 token 注册', async () => {
    sessionStorage.setItem('dingtalk_invitation_token', 'invite-token-abc');
    getInvitationInfo.mockResolvedValueOnce({
      data: {
        organizationName: '天津分公司',
        inviterName: '张经理',
      },
    });
    dingTalkLogin.mockResolvedValueOnce({
      data: {
        status: 2,
        displayName: '孙七',
        registrationOrganizations: [
          {
            organizationId: 'org-cd',
            organizationName: '成都分公司',
            organizationCode: 'CD',
          },
          {
            organizationId: 'org-tj',
            organizationName: '天津分公司',
            organizationCode: 'TJ',
          },
        ],
      },
    });
    registerDingTalkUser.mockResolvedValueOnce({
      data: { displayName: '孙七', status: 'PENDING' },
    });

    render(
      <App>
        <DingTalkCallback />
      </App>,
    );

    expect(await screen.findByText('确认注册')).toBeInTheDocument();
    expect(screen.getByText(/已确认 孙七 属于本企业/)).toBeInTheDocument();
    expect(screen.getByText(/天津分公司/)).toBeInTheDocument();
    // 专属邀请模式下不渲染公司自选下拉框
    expect(screen.queryByText('要加入的公司')).not.toBeInTheDocument();

    fireEvent.click(screen.getByText('确认注册'));

    expect(await screen.findByText('入职或返聘申请已提交')).toBeInTheDocument();
    expect(registerDingTalkUser).toHaveBeenCalledWith(
      { invitationToken: 'invite-token-abc' },
      { skipErrorHandler: true },
    );
    expect(sessionStorage.getItem('dingtalk_invitation_token')).toBeNull();
  });
});
