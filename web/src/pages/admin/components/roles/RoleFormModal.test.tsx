import { act, fireEvent, render, screen } from '@testing-library/react';
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
  ProFormSwitch: ({ label }: { label: string }) => <span>{label}</span>,
  ProFormText: ({ label }: { label: string }) => <span>{label}</span>,
}));

vi.mock('@/components/ui', () => ({
  ProFormSearchableSelect: ({ label }: { label: string }) => (
    <span>{label}</span>
  ),
}));

vi.mock('antd', () => ({
  Alert: ({
    title,
    description,
  }: {
    title: React.ReactNode;
    description?: React.ReactNode;
  }) => (
    <div role="alert">
      {title}
      {description}
    </div>
  ),
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
  Empty: ({ description }: { description: string }) => <div>{description}</div>,
  Input: ({
    value,
    onChange,
    'aria-label': ariaLabel,
  }: React.InputHTMLAttributes<HTMLInputElement>) => (
    <input aria-label={ariaLabel} value={value} onChange={onChange} />
  ),
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

function RoleFormHarness({
  editing,
  onOpenChange = vi.fn(),
  onSuccess = vi.fn(),
}: {
  editing?: API.AdminRole;
  onOpenChange?: (open: boolean) => void;
  onSuccess?: () => void;
}) {
  return (
    <RoleFormModal
      open
      onOpenChange={onOpenChange}
      editing={editing}
      formRef={{ current: undefined }}
      allLeafKeys={[]}
      permissions={[{ key: 'business.order.read' }]}
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
      onSuccess={onSuccess}
    />
  );
}

describe('RoleFormModal 角色提交载荷', () => {
  beforeEach(() => {
    modalState.props = undefined;
    serviceMocks.createRole.mockReset();
    serviceMocks.updateRole.mockReset();
    serviceMocks.createRole.mockResolvedValue({ data: { id: 'role-new' } });
    serviceMocks.updateRole.mockResolvedValue({ data: { id: 'role-2' } });
  });

  it('新增角色只有启用默认值，不显示或预填数据范围', () => {
    render(<RoleFormHarness />);
    expect(modalState.props?.initialValues).toEqual({ enabled: true });
    expect(screen.getByText('角色名称')).toBeInTheDocument();
    expect(screen.queryByText('数据访问范围')).not.toBeInTheDocument();
  });

  it('创建成功时只提交名称与权限集，并关闭刷新', async () => {
    const onOpenChange = vi.fn();
    const onSuccess = vi.fn();
    render(
      <RoleFormHarness onOpenChange={onOpenChange} onSuccess={onSuccess} />,
    );

    await act(async () => {
      await requiredOnFinish()({
        name: '订单主管',
      });
    });
    expect(serviceMocks.createRole).toHaveBeenCalledTimes(1);
    expect(serviceMocks.createRole.mock.calls[0][0]).toEqual({
      code: '',
      name: '订单主管',
      permissionKeys: ['business.order.read'],
    });
    expect(onOpenChange).toHaveBeenCalledWith(false);
    expect(onSuccess).toHaveBeenCalledOnce();
  });

  it('更新停用角色时只提交名称、启用状态与权限集', async () => {
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
        enabled: false,
      });
    });
    expect(serviceMocks.updateRole).toHaveBeenCalledTimes(1);
    expect(serviceMocks.updateRole.mock.calls[0][0]).toEqual({ id: 'role-2' });
    expect(serviceMocks.updateRole.mock.calls[0][1]).toEqual({
      id: 'role-2',
      name: '订单编辑员',
      enabled: false,
      permissionKeys: ['business.order.read'],
    });
  });

  it.each(['create', 'update'] as const)(
    '%s 保存失败时保留弹窗和已选权限',
    async (operation) => {
      const onOpenChange = vi.fn();
      const onSuccess = vi.fn();
      const service =
        operation === 'create'
          ? serviceMocks.createRole
          : serviceMocks.updateRole;
      service.mockRejectedValueOnce(new Error('请求失败'));
      render(
        <RoleFormHarness
          editing={
            operation === 'update'
              ? { id: 'role-2', enabled: false }
              : undefined
          }
          onOpenChange={onOpenChange}
          onSuccess={onSuccess}
        />,
      );

      await act(async () => {
        expect(
          await requiredOnFinish()({ name: '操作员', enabled: false }),
        ).toBe(false);
      });
      expect(onOpenChange).not.toHaveBeenCalled();
      expect(onSuccess).not.toHaveBeenCalled();
      const payload = service.mock.calls[0][operation === 'create' ? 0 : 1];
      expect(payload.permissionKeys).toEqual(['business.order.read']);
      expect(payload).not.toHaveProperty('dataScope');
    },
  );

  it.each(['create', 'update'] as const)(
    '%s 消费错误、失败信封及缺实体均不能关闭或刷新',
    async (operation) => {
      const onOpenChange = vi.fn();
      const onSuccess = vi.fn();
      const service =
        operation === 'create'
          ? serviceMocks.createRole
          : serviceMocks.updateRole;
      render(
        <RoleFormHarness
          editing={operation === 'update' ? { id: 'role-2' } : undefined}
          onOpenChange={onOpenChange}
          onSuccess={onSuccess}
        />,
      );
      for (const response of [
        undefined,
        { success: false, data: { id: 'role-2' } },
        { success: true, data: {} },
      ]) {
        service.mockResolvedValueOnce(response);
        await act(async () =>
          expect(await requiredOnFinish()({ name: '重试名称' })).toBe(false),
        );
        expect(onOpenChange).not.toHaveBeenCalled();
        expect(onSuccess).not.toHaveBeenCalled();
        expect(
          service.mock.calls.at(-1)?.[operation === 'create' ? 0 : 1],
        ).toMatchObject({
          name: '重试名称',
          permissionKeys: ['business.order.read'],
        });
      }
    },
  );

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

  it('搜索显示完整归属，清除与切换模块保留勾选及依赖联动', async () => {
    const permissions: API.AdminPermission[] = [
      { key: 'bill.read', name: '查看账单', group: '费用管理 · 应收 · 账单' },
      {
        key: 'bill.create',
        name: '创建账单',
        group: '费用管理 · 应收 · 账单',
        requires: ['bill.read'],
      },
      { key: 'user.read', name: '查看用户', group: '系统管理 · 用户' },
    ];
    function DiscoveryHarness() {
      const [selectedKeys, setSelectedKeys] = React.useState(['user.read']);
      return (
        <RoleFormModal
          open
          onOpenChange={vi.fn()}
          formRef={{ current: undefined }}
          permissions={permissions}
          selectedPermissionKeys={selectedKeys}
          setSelectedPermissionKeys={setSelectedKeys}
          onSuccess={vi.fn()}
        />
      );
    }
    render(<DiscoveryHarness />);
    expect(screen.getByText('费用管理 → 应收 → 账单')).toBeInTheDocument();
    expect(screen.queryByText(/华为云|IAM/)).not.toBeInTheDocument();
    const searchInput = screen.getByRole('textbox', { name: '搜索角色权限' });
    fireEvent.change(searchInput, { target: { value: '创建账单' } });
    expect(screen.getByText('费用管理 → 应收 → 账单')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('checkbox', { name: '创建账单' }));
    expect(screen.getByRole('checkbox', { name: '查看账单' })).toBeChecked();

    fireEvent.change(searchInput, { target: { value: '没有匹配' } });
    expect(screen.getByText('未找到匹配的权限项')).toBeInTheDocument();
    fireEvent.change(searchInput, { target: { value: '' } });
    fireEvent.click(screen.getByText('系统管理'));
    fireEvent.click(screen.getByRole('button', { name: /展开操作/ }));
    expect(screen.getByRole('checkbox', { name: '查看用户' })).toBeChecked();
    await act(async () => {
      await requiredOnFinish()({ name: '业务员' });
    });
    expect(serviceMocks.createRole.mock.calls[0][0].permissionKeys).toEqual(
      expect.arrayContaining(['user.read', 'bill.read', 'bill.create']),
    );

    fireEvent.click(screen.getByText('费用管理'));
    fireEvent.click(screen.getByRole('button', { name: /展开操作/ }));
    fireEvent.click(screen.getByRole('checkbox', { name: '查看账单' }));
    expect(
      screen.getByRole('checkbox', { name: '创建账单' }),
    ).not.toBeChecked();
  });
});
