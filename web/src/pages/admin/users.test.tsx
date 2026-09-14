import { act, fireEvent, render, screen } from '@testing-library/react';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const accessState = vi.hoisted(() => ({
  value: {
    canReadRoles: true,
    canReadOrganizations: false,
  },
}));

const initialStateState = vi.hoisted(() => ({
  model: {
    initialState: {
      currentUser: {
        id: 'user-1',
        currentOrganization: { id: 'org-1', name: '天津分公司', code: 'TJ' },
      },
    },
  },
}));

const serviceMocks = vi.hoisted(() => ({
  listRoles: vi.fn(),
  listOrganizations: vi.fn(),
  listUsers: vi.fn(),
  terminateUser: vi.fn(),
}));

const buildUserColumnsState = vi.hoisted(() => ({ calls: [] as any[] }));

const proTableState = vi.hoisted(() => ({
  props: undefined as any,
  // 模拟 ProTable 内部分页状态：页码在页签切换后应回到第一页。
  page: 1,
}));

const userFormModalState = vi.hoisted(() => ({ props: undefined as any }));

vi.mock('@umijs/max', () => ({
  useAccess: () => accessState.value,
  useModel: (namespace: string) =>
    namespace === '@@initialState' ? initialStateState.model : {},
}));

vi.mock('@ant-design/pro-components', () => ({
  ProTable: (props: any) => {
    proTableState.props = props;
    // 与真实 ProTable 一致：request 沿用当前分页状态，页码重置只能来自
    // 组件经 actionRef.setPageInfo 的显式调用。
    if (props.actionRef) {
      props.actionRef.current = {
        reload: vi.fn(),
        setPageInfo: (info: { current?: number }) => {
          if (info.current !== undefined) {
            proTableState.page = info.current;
          }
        },
      };
    }
    void props.request?.({
      current: proTableState.page,
      pageSize: 20,
      enabled: props.params?.enabled,
    });
    return <div />;
  },
}));

vi.mock('@/components/ui', () => ({
  SearchFilterTemplate: () => <div />,
}));

vi.mock('antd', () => ({
  App: { useApp: () => ({ message: { success: vi.fn() } }) },
  Button: ({ children }: { children: React.ReactNode }) => (
    <button type="button">{children}</button>
  ),
  Card: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Space: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Tabs: ({
    items,
    activeKey,
    onChange,
  }: {
    items: { key: string; label: string }[];
    activeKey?: string;
    onChange?: (key: string) => void;
  }) => (
    <div>
      {items.map((item) => (
        <button
          key={item.key}
          type="button"
          data-active={item.key === activeKey}
          onClick={() => onChange?.(item.key)}
        >
          {item.label}
        </button>
      ))}
    </div>
  ),
}));

vi.mock('@/services/roncin/adminService', () => ({
  adminServiceListRoles: serviceMocks.listRoles,
  adminServiceListOrganizations: serviceMocks.listOrganizations,
  adminServiceListUsers: serviceMocks.listUsers,
  adminServiceTerminateUser: serviceMocks.terminateUser,
}));

vi.mock('./components/users/ResetPasswordModal', () => ({
  default: () => null,
}));

vi.mock('./components/users/UserFormModal', () => ({
  default: (props: any) => {
    userFormModalState.props = props;
    return null;
  },
}));

vi.mock('./components/users/userColumns', () => ({
  buildUserColumns: (deps: any) => {
    buildUserColumnsState.calls.push(deps);
    return [];
  },
}));

import UsersPanel from './users';

describe('UsersPanel 角色与组织数据源按权限分流', () => {
  beforeEach(() => {
    serviceMocks.listRoles.mockReset();
    serviceMocks.listOrganizations.mockReset();
    serviceMocks.listUsers.mockReset();
    serviceMocks.listUsers.mockResolvedValue({ data: [], total: 0 });
    buildUserColumnsState.calls = [];
    proTableState.page = 1;
    serviceMocks.listRoles.mockResolvedValue({ data: [] });
    serviceMocks.listOrganizations.mockResolvedValue({ data: [] });
  });

  it('普通组织管理员只请求当前组织角色，不发全组织请求', async () => {
    accessState.value = { canReadRoles: true, canReadOrganizations: false };
    await act(async () => {
      render(<UsersPanel />);
    });

    expect(serviceMocks.listRoles).toHaveBeenCalledTimes(1);
    expect(serviceMocks.listOrganizations).not.toHaveBeenCalled();
  });

  it('全局管理员同时请求当前组织角色与全组织列表', async () => {
    accessState.value = { canReadRoles: true, canReadOrganizations: true };
    await act(async () => {
      render(<UsersPanel />);
    });

    expect(serviceMocks.listRoles).toHaveBeenCalledTimes(1);
    expect(serviceMocks.listOrganizations).toHaveBeenCalledTimes(1);
  });

  it('缺少角色读取权限时不请求角色列表', async () => {
    accessState.value = { canReadRoles: false, canReadOrganizations: false };
    await act(async () => {
      render(<UsersPanel />);
    });

    expect(serviceMocks.listRoles).not.toHaveBeenCalled();
    expect(serviceMocks.listOrganizations).not.toHaveBeenCalled();
  });
});

describe('UsersPanel 在职/离职页签数据视图', () => {
  beforeEach(() => {
    serviceMocks.listRoles.mockReset();
    serviceMocks.listOrganizations.mockReset();
    serviceMocks.listUsers.mockReset();
    serviceMocks.listRoles.mockResolvedValue({ data: [] });
    serviceMocks.listOrganizations.mockResolvedValue({ data: [] });
    serviceMocks.listUsers.mockResolvedValue({ data: [], total: 0 });
    buildUserColumnsState.calls = [];
    proTableState.page = 1;
    userFormModalState.props = undefined;
    accessState.value = { canReadRoles: true, canReadOrganizations: false };
  });

  it('默认展示在职页签，以 enabled=true 请求并保留操作列', async () => {
    await act(async () => {
      render(<UsersPanel />);
    });

    expect(screen.getByRole('button', { name: '在职用户' })).toHaveAttribute(
      'data-active',
      'true',
    );
    expect(serviceMocks.listUsers).toHaveBeenCalledWith({
      page: 1,
      pageSize: 20,
      keyword: undefined,
      enabled: true,
    });
    expect(buildUserColumnsState.calls.at(-1)?.showActions).toBe(true);
  });

  it('切换离职页签后以 enabled=false 重新请求并隐藏操作列', async () => {
    await act(async () => {
      render(<UsersPanel />);
    });

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: '离职用户' }));
    });

    expect(screen.getByRole('button', { name: '离职用户' })).toHaveAttribute(
      'data-active',
      'true',
    );
    expect(serviceMocks.listUsers).toHaveBeenLastCalledWith({
      page: 1,
      pageSize: 20,
      keyword: undefined,
      enabled: false,
    });
    expect(buildUserColumnsState.calls.at(-1)?.showActions).toBe(false);
  });

  it('切回在职页签后恢复 enabled=true 请求与操作列', async () => {
    await act(async () => {
      render(<UsersPanel />);
    });

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: '离职用户' }));
    });
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: '在职用户' }));
    });

    expect(serviceMocks.listUsers).toHaveBeenLastCalledWith(
      expect.objectContaining({ enabled: true }),
    );
    expect(buildUserColumnsState.calls.at(-1)?.showActions).toBe(true);
  });

  it('页签切换后回到第一页并以新页签参数请求', async () => {
    await act(async () => {
      render(<UsersPanel />);
    });

    // 模拟用户已翻到第 3 页后切换页签
    await act(async () => {
      proTableState.props.actionRef.current.setPageInfo({ current: 3 });
    });
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: '离职用户' }));
    });

    expect(serviceMocks.listUsers).toHaveBeenLastCalledWith({
      page: 1,
      pageSize: 20,
      keyword: undefined,
      enabled: false,
    });
  });

  it('离职页签下新建用户成功后落回在职页签', async () => {
    await act(async () => {
      render(<UsersPanel />);
    });
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: '离职用户' }));
    });
    expect(screen.getByRole('button', { name: '离职用户' })).toHaveAttribute(
      'data-active',
      'true',
    );

    // 新建用户（editing 为空）成功后 UserFormModal 触发 onReload
    await act(async () => {
      userFormModalState.props.onReload();
    });

    expect(screen.getByRole('button', { name: '在职用户' })).toHaveAttribute(
      'data-active',
      'true',
    );
    expect(serviceMocks.listUsers).toHaveBeenLastCalledWith(
      expect.objectContaining({ enabled: true, page: 1 }),
    );
  });
});
