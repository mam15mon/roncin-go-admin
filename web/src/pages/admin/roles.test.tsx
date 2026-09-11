import { render, screen } from '@testing-library/react';
import React from 'react';
import { describe, expect, it, vi } from 'vitest';

const accessState = vi.hoisted(() => ({
  value: {
    canCreateRoles: false,
    canUpdateRoles: false,
    canReadPermissions: false,
    canReadOrganizations: false,
  },
}));

vi.mock('@umijs/max', () => ({
  useAccess: () => accessState.value,
}));

vi.mock('@ant-design/pro-components', () => ({
  ProTable: ({ columns }: { columns: Array<Record<string, unknown>> }) => {
    const operation = columns.find((column) => column.title === '操作');
    return <div>{(operation?.render as ((_: unknown, role: API.AdminRole) => React.ReactNode) | undefined)?.(null, {})}</div>;
  },
}));

vi.mock('@/components/ui', () => ({
  SearchFilterTemplate: ({ extraRight }: { extraRight: React.ReactNode }) => <div>{extraRight}</div>,
}));

vi.mock('antd', () => ({
  App: { useApp: () => ({ message: { error: vi.fn() } }) },
  Button: ({ children }: { children: React.ReactNode }) => (
    <button type="button">{children}</button>
  ),
  Space: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Tag: ({ children }: { children: React.ReactNode }) => <span>{children}</span>,
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

vi.mock('@/services/roncin/adminService', () => ({
  adminServiceListOrganizations: vi.fn(),
  adminServiceListPermissions: vi.fn(),
  adminServiceListRoles: vi.fn(),
}));

vi.mock('./components/roles/RoleFormModal', () => ({
  default: () => null,
}));

import RolesPanel from './roles';

describe('RolesPanel 角色配置权限', () => {
  it('缺少角色和字典读取权限时不暴露新增或编辑入口', () => {
    render(<RolesPanel />);

    expect(screen.queryByRole('button', { name: '新增角色' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '编辑' })).not.toBeInTheDocument();
  });
});
