import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import React from 'react';
import { describe, expect, it, vi } from 'vitest';
import { adminServiceListPermissions } from '@/services/roncin/adminService';

const accessState = vi.hoisted(() => ({
  value: {
    canCreateRoles: false,
    canUpdateRoles: false,
    canDeleteRoles: false,
    canReadPermissions: false,
    canReadOrganizations: false,
  },
}));

// 操作列渲染的目标角色由用例注入，用于断言删除入口的角色规则。
const operationRow = vi.hoisted(() => ({
  value: {} as API.AdminRole,
}));

// 表单弹窗 props 由用例捕获，用于断言权限树初始勾选。
const formModalProps = vi.hoisted(() => ({
  value: {} as Record<string, unknown>,
}));

vi.mock('@/app/access', () => ({
  useAccess: () => accessState.value,
}));

vi.mock('@ant-design/pro-components', () => ({
  ProTable: ({ columns }: { columns: Array<Record<string, unknown>> }) => {
    const operation = columns.find((column) => column.title === '操作');
    return (
      <div>
        {(
          operation?.render as
            | ((_: unknown, role: API.AdminRole) => React.ReactNode)
            | undefined
        )?.(null, operationRow.value)}
      </div>
    );
  },
}));

vi.mock('@/components/ui', () => ({
  SearchFilterTemplate: ({ extraRight }: { extraRight: React.ReactNode }) => (
    <div>{extraRight}</div>
  ),
}));

vi.mock('antd', async (importOriginal) => {
  const actual = await importOriginal<typeof import('antd')>();
  return {
    ...actual,
    App: { useApp: () => ({ message: { error: vi.fn(), success: vi.fn() } }) },
    Button: ({
      children,
      disabled,
      onClick,
    }: {
      children?: React.ReactNode;
      disabled?: boolean;
      onClick?: React.MouseEventHandler<HTMLButtonElement>;
    }) => (
      <button type="button" disabled={disabled} onClick={onClick}>
        {children}
      </button>
    ),
    Popconfirm: ({ children }: { children: React.ReactNode }) => (
      <>{children}</>
    ),
    Space: ({ children }: { children: React.ReactNode }) => (
      <div>{children}</div>
    ),
    Tag: ({ children }: { children: React.ReactNode }) => (
      <span>{children}</span>
    ),
    Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  };
});

vi.mock('@/services/roncin/adminService', () => ({
  adminServiceDeleteRole: vi.fn(),
  adminServiceListOrganizations: vi.fn(),
  adminServiceListPermissions: vi.fn().mockResolvedValue({ data: [] }),
  adminServiceListRoles: vi.fn(),
}));

vi.mock('./components/roles/RoleFormModal', () => ({
  default: (props: Record<string, unknown>) => {
    formModalProps.value = props;
    return null;
  },
}));

import RolesPanel from './roles';

describe('RolesPanel 角色配置权限', () => {
  it('缺少角色和字典读取权限时不暴露新增或编辑入口', () => {
    render(<RolesPanel />);

    expect(
      screen.queryByRole('button', { name: '新增角色' }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: '编辑' }),
    ).not.toBeInTheDocument();
  });

  it('administrator 角色不显示删除入口', async () => {
    accessState.value = {
      ...accessState.value,
      canDeleteRoles: true,
      canReadPermissions: true,
      canReadOrganizations: true,
    };
    operationRow.value = {
      id: 'role-1',
      code: 'administrator',
      name: '系统管理员',
    };
    render(<RolesPanel />);

    expect(
      screen.queryByRole('button', { name: '删除' }),
    ).not.toBeInTheDocument();

    // 等待权限字典加载的异步落地，避免用例结束后状态更新迟到触发 act 警告
    await waitFor(() => {
      expect(adminServiceListPermissions).toHaveBeenCalled();
    });
  });

  it('已分配成员的角色删除入口禁用', async () => {
    operationRow.value = {
      id: 'role-2',
      code: 'operator',
      name: '操作员',
      assignmentsCount: 2,
    };
    render(<RolesPanel />);

    expect(screen.getByRole('button', { name: '删除' })).toBeDisabled();

    // 等待权限字典加载的异步落地，避免用例结束后状态更新迟到触发 act 警告
    await waitFor(() => {
      expect(adminServiceListPermissions).toHaveBeenCalled();
    });
  });

  it('未分配成员的角色显示可用删除入口', async () => {
    operationRow.value = {
      id: 'role-3',
      code: 'operator',
      name: '操作员',
      assignmentsCount: 0,
    };
    render(<RolesPanel />);

    expect(screen.getByRole('button', { name: '删除' })).toBeEnabled();

    // 等待权限字典加载的异步落地，避免用例结束后状态更新迟到触发 act 警告
    await waitFor(() => {
      expect(adminServiceListPermissions).toHaveBeenCalled();
    });
  });

  it('新建角色默认勾选访问工作台基础权限', async () => {
    accessState.value = {
      ...accessState.value,
      canCreateRoles: true,
      canReadPermissions: true,
      canReadOrganizations: true,
    };
    render(<RolesPanel />);

    fireEvent.click(screen.getByRole('button', { name: '新增角色' }));

    await waitFor(() => {
      expect(formModalProps.value.selectedPermissionKeys).toEqual([
        'system.platform.access',
      ]);
    });
  });
});
