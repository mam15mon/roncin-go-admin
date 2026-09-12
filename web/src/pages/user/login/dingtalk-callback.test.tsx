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
  messageSuccessMock,
} = vi.hoisted(() => ({
  dingTalkLogin: vi.fn(),
  registerDingTalkUser: vi.fn(),
  switchOrganization: vi.fn(),
  messageSuccessMock: vi.fn(),
}));

vi.mock('@umijs/max', () => ({
  Helmet: ({ children }: { children?: React.ReactNode }) => children,
  useModel: () => ({ setInitialState: vi.fn() }),
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
});
