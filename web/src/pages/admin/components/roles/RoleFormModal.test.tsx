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
  Button: ({ children }: { children: React.ReactNode }) => (
    <button type="button">{children}</button>
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
});
