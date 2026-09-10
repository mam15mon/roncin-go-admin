import { act, render } from '@testing-library/react';
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

vi.mock('@umijs/max', () => ({
  useAccess: () => accessState.value,
  useModel: (namespace: string) =>
    namespace === '@@initialState' ? initialStateState.model : {},
}));

vi.mock('@ant-design/pro-components', () => ({
  ProTable: () => <div />,
}));

vi.mock('@/components/ui', () => ({
  SearchFilterTemplate: () => <div />,
}));

vi.mock('antd', () => ({
  App: { useApp: () => ({ message: { success: vi.fn() } }) },
  Button: ({ children }: { children: React.ReactNode }) => (
    <button type="button">{children}</button>
  ),
  Space: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
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
  default: () => null,
}));

vi.mock('./components/users/userColumns', () => ({
  buildUserColumns: () => [],
}));

import UsersPanel from './users';

describe('UsersPanel 角色与组织数据源按权限分流', () => {
  beforeEach(() => {
    serviceMocks.listRoles.mockReset();
    serviceMocks.listOrganizations.mockReset();
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
