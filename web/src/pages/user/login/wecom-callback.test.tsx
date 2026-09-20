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
import WeComCallback from './wecom-callback';

const { weComLogin, switchOrganization, messageSuccessMock } = vi.hoisted(
  () => ({
    weComLogin: vi.fn(),
    switchOrganization: vi.fn(),
    messageSuccessMock: vi.fn(),
  }),
);

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
  authServiceWeComLogin: weComLogin,
  authServiceSwitchOrganization: switchOrganization,
}));

function currentUser(
  organizations: Array<{ id: string; code: string; name: string }>,
  currentOrganizationId: string,
): API.CurrentUser {
  return {
    id: 'user-1',
    displayName: '李四',
    currentOrganization: organizations.find(
      (organization) => organization.id === currentOrganizationId,
    ),
    organizations,
  };
}

describe('WeComCallback', () => {
  let replaceMock: ReturnType<typeof vi.fn<(url: string | URL) => void>>;

  beforeEach(() => {
    sessionStorage.clear();
    window.history.replaceState(
      {},
      '',
      '/user/login/wecom/callback?code=code&state=state',
    );
    replaceMock = vi.fn();
    vi.spyOn(window.location, 'replace').mockImplementation(replaceMock);
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
    vi.restoreAllMocks();
  });

  it('登录失败时展示错误原因并允许返回登录', async () => {
    weComLogin.mockRejectedValueOnce({
      data: { message: '企业微信凭据已过期' },
    });

    render(
      <App>
        <WeComCallback />
      </App>,
    );

    expect(await screen.findByText('企业微信登录未完成')).toBeInTheDocument();
    expect(screen.getByText('企业微信凭据已过期')).toBeInTheDocument();
  });

  it('单组织用户登录后直接进入应用', async () => {
    sessionStorage.setItem('wecom_login_redirect', '/orders');
    weComLogin.mockResolvedValueOnce({
      data: currentUser(
        [{ id: 'org-1', code: 'TJ', name: '天津公司' }],
        'org-1',
      ),
    });

    render(
      <App>
        <WeComCallback />
      </App>,
    );

    await waitFor(() => {
      expect(replaceMock).toHaveBeenCalledWith('/orders');
    });
    expect(screen.queryByText('选择进入的组织')).not.toBeInTheDocument();
    expect(switchOrganization).not.toHaveBeenCalled();
  });

  it('多组织用户登录后先选组织，选其他组织经切换进入', async () => {
    sessionStorage.setItem('wecom_login_redirect', '/welcome');
    const organizations = [
      { id: 'org-1', code: 'TJ', name: '天津公司' },
      { id: 'org-2', code: 'BJ', name: '北京公司' },
    ];
    weComLogin.mockResolvedValueOnce({
      data: currentUser(organizations, 'org-1'),
    });
    switchOrganization.mockResolvedValueOnce({
      data: {
        id: 'user-1',
        currentOrganization: { id: 'org-2', code: 'BJ', name: '北京公司' },
      },
    });

    render(
      <App>
        <WeComCallback />
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
    expect(messageSuccessMock).toHaveBeenCalledWith('企业微信登录成功');
  });
});
