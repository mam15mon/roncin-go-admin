import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
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

  it.each([
    [2, '人员尚未同步', '请联系管理员同步开通'],
    [3, '尚未分配组织', '请联系管理员分配组织和角色'],
    [4, '账号已停用', '请联系管理员重新开通'],
  ])('未开通状态 %s 仅展示管理员处理提示', async (status, title, message) => {
    dingTalkLogin.mockResolvedValueOnce({ data: { status, message } });
    renderWithClient(
      <App>
        <DingTalkCallback />
      </App>,
    );
    expect(await screen.findByText(title)).toBeInTheDocument();
    expect(screen.getByText(message)).toBeInTheDocument();
    expect(screen.queryByText('确认注册')).not.toBeInTheDocument();
    expect(screen.getByText('返回登录')).toBeInTheDocument();
    expect(registerDingTalkUser).not.toHaveBeenCalled();
    expect(replaceMock).not.toHaveBeenCalled();
  });

  it('单组织用户认证后直接进入应用', async () => {
    sessionStorage.setItem('dingtalk_login_redirect', '/orders');
    dingTalkLogin.mockResolvedValueOnce({
      data: authenticatedUser(
        [{ id: 'org-1', code: 'TJ', name: '天津公司' }],
        'org-1',
      ),
    });

    renderWithClient(
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

    renderWithClient(
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

  it('多组织用户选本次会话组织不发起切换', async () => {
    sessionStorage.setItem('dingtalk_login_redirect', '/welcome');
    const organizations = [
      { id: 'org-1', code: 'TJ', name: '天津公司' },
      { id: 'org-2', code: 'BJ', name: '北京公司' },
    ];
    dingTalkLogin.mockResolvedValueOnce({
      data: authenticatedUser(organizations, 'org-1'),
    });

    renderWithClient(
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
});
