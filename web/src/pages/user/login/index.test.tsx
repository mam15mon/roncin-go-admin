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
  messageSuccessMock,
  setInitialStateMock,
} = vi.hoisted(() => ({
  authServiceLoginMock: vi.fn(),
  authServiceSwitchOrganizationMock: vi.fn(),
  authServiceGetWeComLoginConfigMock: vi.fn(),
  authServiceGetDingTalkLoginConfigMock: vi.fn(),
  messageSuccessMock: vi.fn(),
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
      message: { success: messageSuccessMock },
    }),
  });
  return { ...actual, App: MockedApp };
});

vi.mock('@/services/roncin/authService', () => ({
  authServiceLogin: authServiceLoginMock,
  authServiceSwitchOrganization: authServiceSwitchOrganizationMock,
  authServiceGetWeComLoginConfig: authServiceGetWeComLoginConfigMock,
  authServiceGetDingTalkLoginConfig: authServiceGetDingTalkLoginConfigMock,
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
  });

  afterEach(() => {
    vi.restoreAllMocks();
    cleanup();
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
});
