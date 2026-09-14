import { act, render } from '@testing-library/react';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const serviceMocks = vi.hoisted(() => ({
  createRole: vi.fn(),
  updateRole: vi.fn(),
}));

const modalState = vi.hoisted(() => ({
  props: undefined as Record<string, unknown> | undefined,
}));

vi.mock('@ant-design/pro-components', () => ({
  ModalForm: (props: Record<string, unknown>) => {
    modalState.props = props;
    return <div>{props.children as React.ReactNode}</div>;
  },
  ProFormSwitch: () => null,
  ProFormText: () => null,
}));

vi.mock('@/components/ui', () => ({
  ProFormSearchableSelect: () => null,
}));

vi.mock('antd', () => ({
  App: { useApp: () => ({ message: { error: vi.fn(), success: vi.fn() } }) },
  Button: ({
    children,
    onClick,
  }: {
    children: React.ReactNode;
    onClick?: () => void;
  }) => (
    <button type="button" onClick={onClick}>
      {children}
    </button>
  ),
  Checkbox: ({
    children,
    checked,
    onChange,
    disabled,
  }: {
    children?: React.ReactNode;
    checked?: boolean;
    onChange?: (e: { target: { checked: boolean } }) => void;
    disabled?: boolean;
  }) => (
    <label>
      <input
        type="checkbox"
        checked={checked}
        disabled={disabled}
        onChange={(e) => onChange?.({ target: { checked: e.target.checked } })}
      />
      {children}
    </label>
  ),
  Col: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Empty: () => null,
  Input: () => null,
  Row: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Space: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Tag: ({ children }: { children: React.ReactNode }) => <span>{children}</span>,
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  Tree: () => null,
  Typography: {
    Text: ({ children }: { children: React.ReactNode }) => (
      <span>{children}</span>
    ),
  },
}));

vi.mock('@/services/roncin/adminService', () => ({
  adminServiceCreateRole: serviceMocks.createRole,
  adminServiceUpdateRole: serviceMocks.updateRole,
}));

import RoleFormModal from './RoleFormModal';

function requiredOnFinish() {
  const onFinish = modalState.props?.onFinish;
  if (typeof onFinish !== 'function') {
    throw new Error('未找到角色表单提交处理器');
  }
  return onFinish as (values: Record<string, unknown>) => Promise<boolean>;
}

function RoleFormHarness({ editing }: { editing?: API.AdminRole }) {
  return (
    <RoleFormModal
      open
      onOpenChange={vi.fn()}
      editing={editing}
      formRef={{ current: undefined }}
      allLeafKeys={[]}
      allGroupKeys={[]}
      filteredTreeData={[]}
      requiresByPermission={{}}
      permissionNameByKey={{}}
      selectedPermissionKeys={['business.order.read']}
      setSelectedPermissionKeys={vi.fn()}
      expandedKeys={[]}
      setExpandedKeys={vi.fn()}
      autoExpandParent={false}
      setAutoExpandParent={vi.fn()}
      permissionKeyword=""
      setPermissionKeyword={vi.fn()}
      onSuccess={vi.fn()}
    />
  );
}

describe('RoleFormModal 角色提交载荷', () => {
  beforeEach(() => {
    modalState.props = undefined;
    serviceMocks.createRole.mockReset();
    serviceMocks.updateRole.mockReset();
    serviceMocks.createRole.mockResolvedValue({});
    serviceMocks.updateRole.mockResolvedValue({});
  });

  it('创建载荷只携带名称、数据范围与权限集', async () => {
    render(<RoleFormHarness />);

    await act(async () => {
      await requiredOnFinish()({
        name: '订单主管',
        dataScope: 2,
      });
    });
    expect(serviceMocks.createRole).toHaveBeenCalledTimes(1);
    expect(serviceMocks.createRole.mock.calls[0][0]).toEqual({
      code: '',
      name: '订单主管',
      dataScope: 2,
      permissionKeys: ['business.order.read'],
    });
  });

  it('更新载荷只携带名称、数据范围、启用状态与权限集', async () => {
    render(
      <RoleFormHarness
        editing={{
          id: 'role-2',
          name: '订单编辑员',
          code: 'order_editor',
          enabled: false,
        }}
      />,
    );

    await act(async () => {
      await requiredOnFinish()({
        name: '订单编辑员',
        dataScope: 2,
        enabled: false,
      });
    });
    expect(serviceMocks.updateRole).toHaveBeenCalledTimes(1);
    expect(serviceMocks.updateRole.mock.calls[0][0]).toEqual({ id: 'role-2' });
    expect(serviceMocks.updateRole.mock.calls[0][1]).toEqual({
      id: 'role-2',
      name: '订单编辑员',
      dataScope: 2,
      enabled: false,
      permissionKeys: ['business.order.read'],
    });
  });

  it('点击全选只读时仅选中只读类权限', () => {
    const setSelectedPermissionKeys = vi.fn();
    const permissions: API.AdminPermission[] = [
      { key: 'system.user.read', name: '查看用户', group: '系统管理 · 用户' },
      {
        key: 'system.user.create',
        name: '新建用户',
        group: '系统管理 · 用户',
        requires: ['system.user.read'],
      },
    ];

    const { getByRole } = render(
      <RoleFormModal
        open
        onOpenChange={vi.fn()}
        formRef={{ current: undefined }}
        permissions={permissions}
        selectedPermissionKeys={[]}
        setSelectedPermissionKeys={setSelectedPermissionKeys}
        onSuccess={vi.fn()}
      />,
    );

    const readOnlyBtn = getByRole('button', { name: /全选只读/ });
    act(() => {
      readOnlyBtn.click();
    });

    expect(setSelectedPermissionKeys).toHaveBeenCalledWith([
      'system.user.read',
    ]);
  });

  it('点击全选读写时选中全部权限', () => {
    const setSelectedPermissionKeys = vi.fn();
    const permissions: API.AdminPermission[] = [
      { key: 'system.user.read', name: '查看用户', group: '系统管理 · 用户' },
      {
        key: 'system.user.create',
        name: '新建用户',
        group: '系统管理 · 用户',
        requires: ['system.user.read'],
      },
    ];

    const { getByRole } = render(
      <RoleFormModal
        open
        onOpenChange={vi.fn()}
        formRef={{ current: undefined }}
        permissions={permissions}
        selectedPermissionKeys={[]}
        setSelectedPermissionKeys={setSelectedPermissionKeys}
        onSuccess={vi.fn()}
      />,
    );

    const fullBtn = getByRole('button', { name: /全选读写/ });
    act(() => {
      fullBtn.click();
    });

    expect(setSelectedPermissionKeys).toHaveBeenCalledWith([
      'system.user.read',
      'system.user.create',
    ]);
  });

  it('点击清空时将权限集置空', () => {
    const setSelectedPermissionKeys = vi.fn();
    const { getByRole } = render(
      <RoleFormModal
        open
        onOpenChange={vi.fn()}
        formRef={{ current: undefined }}
        selectedPermissionKeys={['system.user.read']}
        setSelectedPermissionKeys={setSelectedPermissionKeys}
        onSuccess={vi.fn()}
      />,
    );

    const clearBtn = getByRole('button', { name: /清空/ });
    act(() => {
      clearBtn.click();
    });

    expect(setSelectedPermissionKeys).toHaveBeenCalledWith([]);
  });
});
