import { act, render } from '@testing-library/react';
import React, { useState } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const serviceMocks = vi.hoisted(() => ({
  createRole: vi.fn(),
  updateRole: vi.fn(),
}));

const modalState = vi.hoisted(() => ({
  props: undefined as Record<string, unknown> | undefined,
}));

const selectState = vi.hoisted(
  () => new Map<string, Record<string, unknown>>(),
);

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
  Select: (props: Record<string, unknown>) => {
    selectState.set(String(props.placeholder), props);
    return <div />;
  },
  Space: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Tag: ({ children }: { children: React.ReactNode }) => <span>{children}</span>,
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  Tree: () => null,
  Typography: { Text: ({ children }: { children: React.ReactNode }) => <span>{children}</span> },
}));

vi.mock('@/services/roncin/adminService', () => ({
  adminServiceCreateRole: serviceMocks.createRole,
  adminServiceUpdateRole: serviceMocks.updateRole,
}));

import RoleFormModal from './RoleFormModal';
import type { OrganizationAccess } from './roleConstants';

function requiredSelect(placeholder: string) {
  const props = selectState.get(placeholder);
  if (!props) {
    throw new Error(`未找到 ${placeholder} 选择框`);
  }
  return props;
}

function requiredOnFinish() {
  const onFinish = modalState.props?.onFinish;
  if (typeof onFinish !== 'function') {
    throw new Error('未找到角色表单提交处理器');
  }
  return onFinish as (values: Record<string, unknown>) => Promise<boolean>;
}

function RoleFormHarness({ editing }: { editing?: API.AdminRole }) {
  const [organizationAccesses, setOrganizationAccesses] = useState<
    OrganizationAccess[]
  >(
    (editing?.organizationAccesses ?? []).map((access) => ({
      organizationId: access.organizationId ?? '',
      writable: access.writable ?? false,
    })),
  );

  return (
    <RoleFormModal
      open
      onOpenChange={vi.fn()}
      editing={editing}
      formRef={{ current: undefined }}
      companyOptions={[
        { label: '北京公司', value: 'beijing' },
        { label: '天津公司', value: 'tianjin' },
      ]}
      allLeafKeys={[]}
      allGroupKeys={[]}
      filteredTreeData={[]}
      requiresByPermission={{}}
      permissionNameByKey={{}}
      selectedPermissionKeys={['business.order.read']}
      setSelectedPermissionKeys={vi.fn()}
      organizationAccesses={organizationAccesses}
      setOrganizationAccesses={setOrganizationAccesses}
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

describe('RoleFormModal 组织访问项', () => {
  beforeEach(() => {
    modalState.props = undefined;
    selectState.clear();
    serviceMocks.createRole.mockReset();
    serviceMocks.updateRole.mockReset();
    serviceMocks.createRole.mockResolvedValue({});
    serviceMocks.updateRole.mockResolvedValue({});
  });

  it('加载 organizationAccesses，并能单独切换 writable', async () => {
    render(
      <RoleFormHarness
        editing={{
          id: 'role-1',
          name: '订单查看员',
          code: 'order_viewer',
          organizationAccesses: [
            { organizationId: 'beijing', writable: false },
          ],
        }}
      />,
    );

    const readable = requiredSelect('不选择时仅可访问当前组织');
    const writable = requiredSelect('不选择时跨组织均为仅查看');
    expect(readable?.value).toEqual(['beijing']);
    expect(writable?.value).toEqual([]);

    await act(async () => {
      (writable.onChange as (values: string[]) => void)(['beijing']);
    });

    expect(selectState.get('不选择时跨组织均为仅查看')?.value).toEqual([
      'beijing',
    ]);
  });

  it('创建与更新都提交通用 organizationAccesses 字段', async () => {
    const { rerender } = render(<RoleFormHarness />);

    await act(async () => {
      (
        requiredSelect('不选择时仅可访问当前组织')
          .onChange as (values: string[]) => void
      )(['beijing']);
      (
        requiredSelect('不选择时跨组织均为仅查看')
          .onChange as (values: string[]) => void
      )(['beijing']);
    });
    await act(async () => {
      await requiredOnFinish()({
        name: '订单主管',
        dataScope: 2,
      });
    });
    expect(serviceMocks.createRole).toHaveBeenCalledWith(
      expect.objectContaining({
        organizationAccesses: [
          { organizationId: 'beijing', writable: true },
        ],
      }),
    );
    expect(serviceMocks.createRole.mock.calls[0][0]).not.toHaveProperty(
      'orderOrganizationAccesses',
    );

    rerender(
      <RoleFormHarness
        key="edit-role-2"
        editing={{
          id: 'role-2',
          name: '订单编辑员',
          code: 'order_editor',
          enabled: false,
          organizationAccesses: [
            { organizationId: 'tianjin', writable: true },
          ],
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
    expect(serviceMocks.updateRole).toHaveBeenCalledWith(
      { id: 'role-2' },
      expect.objectContaining({
        organizationAccesses: [
          { organizationId: 'tianjin', writable: true },
        ],
      }),
    );
    expect(serviceMocks.updateRole.mock.calls[0][1]).not.toHaveProperty(
      'orderOrganizationAccesses',
    );
  });
});
