import {
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
  modalConfirmMock,
  replaceMock,
  setInitialStateMock,
} = vi.hoisted(() => ({
  authServiceSwitchOrganizationMock: vi.fn(),
  messageErrorMock: vi.fn(),
  messageSuccessMock: vi.fn(),
  modalConfirmMock: vi.fn(),
  replaceMock: vi.fn(),
  setInitialStateMock: vi.fn(),
}));

let currentUser: API.CurrentUser;

vi.mock('@umijs/max', () => ({
  history: { replace: replaceMock },
  useModel: () => ({
    initialState: { currentUser },
    setInitialState: setInitialStateMock,
  }),
}));

vi.mock('antd', () => ({
  App: {
    useApp: () => ({
      message: {
        error: messageErrorMock,
        success: messageSuccessMock,
      },
    }),
  },
  Modal: { confirm: modalConfirmMock },
  Select: ({
    'aria-label': ariaLabel,
    disabled,
    loading: _loading,
    onChange,
    options,
    value,
  }: {
    'aria-label'?: string;
    disabled?: boolean;
    loading?: boolean;
    onChange?: (value: string) => void;
    options?: Array<{ label: string; value: string }>;
    value?: string;
  }) => (
    <select
      aria-label={ariaLabel}
      disabled={disabled}
      onChange={(event) => onChange?.(event.target.value)}
      value={value}
    >
      {options?.map((option) => (
        <option key={option.value} value={option.value}>
          {option.label}
        </option>
      ))}
    </select>
  ),
}));

vi.mock('@/services/roncin/authService', () => ({
  authServiceSwitchOrganization: authServiceSwitchOrganizationMock,
}));

function createUser(organizationId = 'org-1'): API.CurrentUser {
  return {
    id: 'user-1',
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

describe('OrganizationSwitcher', () => {
  beforeEach(() => {
    currentUser = createUser();
    vi.clearAllMocks();
    _clearAllTabCloseGuards();
  });

  afterEach(() => {
    _clearAllTabCloseGuards();
    cleanup();
  });

  it('选择当前组织时不发起切换请求', () => {
    render(<OrganizationSwitcher />);

    fireEvent.change(screen.getByLabelText('切换当前组织'), {
      target: { value: 'org-1' },
    });

    expect(authServiceSwitchOrganizationMock).not.toHaveBeenCalled();
  });

  it('存在 dirty 表单时，取消确认不会请求；确认后才切换', async () => {
    registerTabCloseGuard('/orders/sea-export', {
      isDirty: () => true,
    });
    authServiceSwitchOrganizationMock.mockResolvedValue({
      data: createUser('org-2'),
    });
    render(<OrganizationSwitcher />);

    fireEvent.change(screen.getByLabelText('切换当前组织'), {
      target: { value: 'org-2' },
    });

    expect(modalConfirmMock).toHaveBeenCalledTimes(1);
    const modalProps = modalConfirmMock.mock.calls[0][0];
    modalProps.onCancel();
    expect(authServiceSwitchOrganizationMock).not.toHaveBeenCalled();

    fireEvent.change(screen.getByLabelText('切换当前组织'), {
      target: { value: 'org-2' },
    });
    modalConfirmMock.mock.calls[1][0].onOk();

    await waitFor(() => {
      expect(authServiceSwitchOrganizationMock).toHaveBeenCalledWith({
        organizationId: 'org-2',
      });
    });
    expect(replaceMock).toHaveBeenCalledWith('/welcome');
  });

  it('切换成功后更新用户状态并跳转到工作台', async () => {
    const switchedUser = createUser('org-2');
    const callOrder: string[] = [];
    setInitialStateMock.mockImplementation(() => {
      callOrder.push('setInitialState');
    });
    replaceMock.mockImplementation(() => {
      callOrder.push('replace');
    });
    authServiceSwitchOrganizationMock.mockResolvedValue({ data: switchedUser });
    render(<OrganizationSwitcher />);

    fireEvent.change(screen.getByLabelText('切换当前组织'), {
      target: { value: 'org-2' },
    });

    await waitFor(() => {
      expect(setInitialStateMock).toHaveBeenCalledTimes(1);
    });
    const updateInitialState = setInitialStateMock.mock.calls[0][0];
    expect(updateInitialState({ currentUser })).toEqual({
      currentUser: switchedUser,
    });
    expect(replaceMock).toHaveBeenCalledWith('/welcome');
    expect(callOrder).toEqual(['setInitialState', 'replace']);
    expect(messageSuccessMock).toHaveBeenCalledWith('已切换当前组织');
  });

  it('请求失败时保留当前状态与路由', async () => {
    authServiceSwitchOrganizationMock.mockRejectedValue(new Error('network'));
    render(<OrganizationSwitcher />);

    fireEvent.change(screen.getByLabelText('切换当前组织'), {
      target: { value: 'org-2' },
    });

    await waitFor(() => {
      expect(authServiceSwitchOrganizationMock).toHaveBeenCalledTimes(1);
    });
    expect(setInitialStateMock).not.toHaveBeenCalled();
    expect(replaceMock).not.toHaveBeenCalled();
  });

  it('服务端未返回新会话时保留当前状态与路由并提示失败', async () => {
    authServiceSwitchOrganizationMock.mockResolvedValue({ data: undefined });
    render(<OrganizationSwitcher />);

    fireEvent.change(screen.getByLabelText('切换当前组织'), {
      target: { value: 'org-2' },
    });

    await waitFor(() => {
      expect(messageErrorMock).toHaveBeenCalledWith('组织切换失败，请稍后重试');
    });
    expect(setInitialStateMock).not.toHaveBeenCalled();
    expect(replaceMock).not.toHaveBeenCalled();
  });

  it('请求进行中忽略重复切换', async () => {
    let resolveSwitch: ((value: { data: API.CurrentUser }) => void) | undefined;
    authServiceSwitchOrganizationMock.mockReturnValue(
      new Promise<{ data: API.CurrentUser }>((resolve) => {
        resolveSwitch = resolve;
      }),
    );
    render(<OrganizationSwitcher />);

    const select = screen.getByLabelText('切换当前组织');
    fireEvent.change(select, { target: { value: 'org-2' } });
    fireEvent.change(select, { target: { value: 'org-2' } });
    expect(authServiceSwitchOrganizationMock).toHaveBeenCalledTimes(1);
    expect(select).toBeDisabled();

    if (!resolveSwitch) {
      throw new Error('切换请求未初始化');
    }
    resolveSwitch({ data: createUser('org-2') });
    await waitFor(() => {
      expect(replaceMock).toHaveBeenCalledWith('/welcome');
    });
  });
});
