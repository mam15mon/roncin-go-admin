import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  _clearAllTabCloseGuards,
  registerTabCloseGuard,
} from './layout/tabCloseGuard';
import OrganizationSwitcher from './OrganizationSwitcher';

const {
  authServiceSwitchOrganizationMock,
  messageErrorMock,
  messageSuccessMock,
  messageInfoMock,
  modalConfirmMock,
  replaceMock,
  switchWorkspaceMock,
} = vi.hoisted(() => ({
  authServiceSwitchOrganizationMock: vi.fn(),
  messageErrorMock: vi.fn(),
  messageSuccessMock: vi.fn(),
  messageInfoMock: vi.fn(),
  modalConfirmMock: vi.fn(),
  replaceMock: vi.fn(),
  switchWorkspaceMock: vi.fn(),
}));

let currentUser: API.CurrentUser;

vi.mock('@/router/history', () => ({
  history: {
    replace: replaceMock,
    location: { pathname: '/admin', search: '?tab=roles' },
  },
}));

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: { currentUser },
  }),
}));

vi.mock('antd', async (importOriginal) => {
  const actual = await importOriginal<typeof import('antd')>();
  const MockedApp = Object.assign(actual.App, {
    useApp: () => ({
      message: {
        error: messageErrorMock,
        success: messageSuccessMock,
        info: messageInfoMock,
      },
    }),
  });
  return {
    ...actual,
    App: MockedApp,
    Spin: ({
      spinning,
      description,
    }: {
      spinning?: boolean;
      description?: React.ReactNode;
    }) => (spinning ? <div role="status">{description}</div> : null),
  };
});

vi.mock('@/components/HeaderDropdown', () => ({
  default: ({
    menu,
    children,
  }: {
    menu?: {
      items?: Array<{
        key: string;
        disabled?: boolean;
        label: React.ReactNode;
      }>;
      onClick?: (info: { key: string }) => void;
    };
    children?: React.ReactNode;
  }) => (
    <div>
      {children}
      <div data-testid="organization-menu">
        {menu?.items?.map((item) => (
          <button
            key={item.key}
            type="button"
            data-menu-key={item.key}
            disabled={item.disabled}
            onClick={() => menu.onClick?.({ key: item.key })}
          >
            {item.label}
          </button>
        ))}
      </div>
    </div>
  ),
}));

vi.mock('@/utils/appFeedback', () => ({
  getAppFeedback: () => ({
    modal: { confirm: modalConfirmMock },
  }),
}));

vi.mock('@/services/roncin/authService', () => ({
  authServiceSwitchOrganization: authServiceSwitchOrganizationMock,
}));

function createUser(organizationId = 'org-1'): API.CurrentUser {
  return {
    id: 'user-1',
    permissionCapabilities: ['system.platform.access', 'system.role.read'].map(
      (key) => ({ key, dataScope: 'organization' }),
    ),
    currentOrganization: {
      id: organizationId,
      code: organizationId === 'org-1' ? 'TJ' : 'BJ',
      name: organizationId === 'org-1' ? '天津公司' : '北京公司',
    },
    organizations: [
      { id: 'org-1', code: 'TJ', name: '天津公司' },
      { id: 'org-2', code: 'BJ', name: '北京公司' },
    ],
  };
}

function menuButton(organizationId: string) {
  return screen
    .getByTestId('organization-menu')
    .querySelector(`[data-menu-key="${organizationId}"]`) as HTMLButtonElement;
}

describe('OrganizationSwitcher', () => {
  beforeEach(() => {
    currentUser = createUser();
    vi.clearAllMocks();
    switchWorkspaceMock.mockResolvedValue(undefined);
    _clearAllTabCloseGuards();
  });

  afterEach(() => {
    _clearAllTabCloseGuards();
    cleanup();
  });

  it('单组织或缺少当前组织时不渲染切换入口', () => {
    currentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-1', code: 'TJ', name: '天津公司' },
      organizations: [{ id: 'org-1', code: 'TJ', name: '天津公司' }],
    };
    const { container, unmount } = render(
      <OrganizationSwitcher onSwitchWorkspace={switchWorkspaceMock} />,
    );
    expect(container.firstChild).toBeNull();
    unmount();

    currentUser = {
      id: 'user-1',
      organizations: [
        { id: 'org-1', code: 'TJ', name: '天津公司' },
        { id: 'org-2', code: 'BJ', name: '北京公司' },
      ],
    };
    const { container: containerWithoutCurrent } = render(
      <OrganizationSwitcher onSwitchWorkspace={switchWorkspaceMock} />,
    );
    expect(containerWithoutCurrent.firstChild).toBeNull();
  });

  it('多组织时展示当前组织入口：当前项打勾置灰，其他组织可切换', () => {
    render(<OrganizationSwitcher onSwitchWorkspace={switchWorkspaceMock} />);

    expect(
      screen.getByRole('button', { name: '切换当前组织' }),
    ).toBeInTheDocument();
    expect(screen.getAllByText('天津公司').length).toBeGreaterThan(0);

    const current = menuButton('org-1');
    expect(current).toBeDisabled();
    // 当前项带对勾标识
    expect(current.querySelector('.anticon-check')).not.toBeNull();
    expect(menuButton('org-2')).toBeEnabled();
  });

  it('点击当前组织不发起切换请求', () => {
    render(<OrganizationSwitcher onSwitchWorkspace={switchWorkspaceMock} />);

    fireEvent.click(menuButton('org-1'));

    expect(authServiceSwitchOrganizationMock).not.toHaveBeenCalled();
  });

  it('存在 dirty 表单时，取消确认不会请求；确认后才切换', async () => {
    registerTabCloseGuard('/orders/sea-export', {
      isDirty: () => true,
    });
    authServiceSwitchOrganizationMock.mockResolvedValue({
      data: createUser('org-2'),
    });
    render(<OrganizationSwitcher onSwitchWorkspace={switchWorkspaceMock} />);

    fireEvent.click(menuButton('org-2'));

    expect(modalConfirmMock).toHaveBeenCalledTimes(1);
    modalConfirmMock.mock.calls[0][0].onCancel();
    expect(authServiceSwitchOrganizationMock).not.toHaveBeenCalled();

    fireEvent.click(menuButton('org-2'));
    // onOk 同步触发切换 loading 状态更新，必须在 act 内调用
    act(() => {
      modalConfirmMock.mock.calls[1][0].onOk();
    });

    await waitFor(() => {
      expect(authServiceSwitchOrganizationMock).toHaveBeenCalledWith({
        organizationId: 'org-2',
      });
      // 等待切换链路完全收敛：跳转已发生且 loading 复位（按钮恢复可用），
      // 避免用例结束后仍有状态更新落地触发 act 警告
      expect(switchWorkspaceMock).toHaveBeenCalledWith(
        createUser('org-2'),
        '/admin?tab=roles',
      );
      expect(
        screen.getByRole('button', { name: '切换当前组织' }),
      ).toBeEnabled();
    });
    expect(switchWorkspaceMock).toHaveBeenCalledWith(
      createUser('org-2'),
      '/admin?tab=roles',
    );
  });

  it('切换成功时保持 loading，直到业务工作区和目标路由完成转换', async () => {
    let resolveWorkspace: (() => void) | undefined;
    switchWorkspaceMock.mockReturnValue(
      new Promise<void>((resolve) => {
        resolveWorkspace = resolve;
      }),
    );
    authServiceSwitchOrganizationMock.mockResolvedValue({
      data: createUser('org-2'),
    });
    render(<OrganizationSwitcher onSwitchWorkspace={switchWorkspaceMock} />);
    fireEvent.click(menuButton('org-2'));
    expect(await screen.findByText('正在切换至 北京公司…')).toBeInTheDocument();
    await waitFor(() =>
      expect(switchWorkspaceMock).toHaveBeenCalledWith(
        createUser('org-2'),
        '/admin?tab=roles',
      ),
    );
    expect(screen.getByRole('button', { name: '切换当前组织' })).toBeDisabled();
    expect(messageSuccessMock).not.toHaveBeenCalled();
    await act(async () => resolveWorkspace?.());
    expect(messageSuccessMock).toHaveBeenCalledWith('已切换至 北京公司');
    expect(screen.getByRole('button', { name: '切换当前组织' })).toBeEnabled();
  });

  it('目标公司缺少当前子页签权限时回工作台并说明原因', async () => {
    const nextUser = createUser('org-2');
    nextUser.permissionCapabilities = [
      { key: 'system.platform.access', dataScope: 'organization' },
    ];
    authServiceSwitchOrganizationMock.mockResolvedValue({ data: nextUser });
    render(<OrganizationSwitcher onSwitchWorkspace={switchWorkspaceMock} />);
    fireEvent.click(menuButton('org-2'));
    await waitFor(() =>
      expect(switchWorkspaceMock).toHaveBeenCalledWith(nextUser, '/welcome'),
    );
    expect(messageInfoMock).toHaveBeenCalledWith(
      '当前组织无权访问该功能，已返回工作台',
    );
  });

  it('请求失败时保留当前状态与路由', async () => {
    authServiceSwitchOrganizationMock.mockRejectedValue(new Error('network'));
    render(<OrganizationSwitcher onSwitchWorkspace={switchWorkspaceMock} />);

    fireEvent.click(menuButton('org-2'));

    await waitFor(() => {
      expect(authServiceSwitchOrganizationMock).toHaveBeenCalledTimes(1);
    });
    expect(switchWorkspaceMock).not.toHaveBeenCalled();
    expect(replaceMock).not.toHaveBeenCalled();
  });

  it('服务端未返回新会话时保留当前状态与路由并提示失败', async () => {
    authServiceSwitchOrganizationMock.mockResolvedValue({ data: undefined });
    render(<OrganizationSwitcher onSwitchWorkspace={switchWorkspaceMock} />);

    fireEvent.click(menuButton('org-2'));

    await waitFor(() => {
      expect(messageErrorMock).toHaveBeenCalledWith('组织切换失败，请稍后重试');
    });
    expect(switchWorkspaceMock).not.toHaveBeenCalled();
    expect(replaceMock).not.toHaveBeenCalled();
  });

  it('请求进行中忽略重复切换', async () => {
    let resolveSwitch: ((value: { data: API.CurrentUser }) => void) | undefined;
    authServiceSwitchOrganizationMock.mockReturnValue(
      new Promise<{ data: API.CurrentUser }>((resolve) => {
        resolveSwitch = resolve;
      }),
    );
    render(<OrganizationSwitcher onSwitchWorkspace={switchWorkspaceMock} />);

    fireEvent.click(menuButton('org-2'));
    fireEvent.click(menuButton('org-2'));
    expect(authServiceSwitchOrganizationMock).toHaveBeenCalledTimes(1);

    if (!resolveSwitch) {
      throw new Error('切换请求未初始化');
    }
    resolveSwitch({ data: createUser('org-2') });
    await waitFor(() => {
      expect(switchWorkspaceMock).toHaveBeenCalledWith(
        createUser('org-2'),
        '/admin?tab=roles',
      );
    });
  });
});
