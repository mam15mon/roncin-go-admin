import capabilityFixture from '@root/tests/fixtures/role-permission-capabilities.json';
import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useInitialState } from '@/app/AppProvider';
import { OrganizationWorkspace } from '@/components/layout/OrganizationWorkspace';
import { AuthOrganizationKind } from '@/enums.generated';
import { manifestPermissionKeys } from '@/permissions.generated';
import {
  adminServiceCreateRole,
  adminServiceListPermissions,
  adminServiceUpdateRole,
} from '@/services/roncin/adminService';

const tableState = vi.hoisted(() => ({
  role: undefined as API.AdminRole | undefined,
}));
const formState = vi.hoisted(() => ({
  onFinish: undefined as
    | ((values: { name: string }) => Promise<boolean>)
    | undefined,
}));

vi.mock('@/components/layout/TagsView', () => ({ TagsView: () => null }));

vi.mock('@ant-design/pro-components', () => ({
  ProTable: ({
    columns,
  }: {
    columns: Array<{
      valueType?: string;
      render?: (value: unknown, row: API.AdminRole) => React.ReactNode;
    }>;
  }) => (
    <div>
      {tableState.role &&
        columns
          .find((column) => column.valueType === 'option')
          ?.render?.(null, tableState.role)}
    </div>
  ),
  ModalForm: ({
    open,
    onFinish,
    onOpenChange,
    submitter,
    children,
  }: {
    open: boolean;
    onFinish: (values: { name: string }) => Promise<boolean>;
    onOpenChange: (open: boolean) => void;
    submitter: { submitButtonProps: { disabled: boolean } };
    children: React.ReactNode;
  }) => {
    formState.onFinish = onFinish;
    return open ? (
      <div>
        {children}
        <button
          type="button"
          disabled={submitter.submitButtonProps.disabled}
          onClick={() => void onFinish({ name: '管理员' })}
        >
          保存角色
        </button>
        <button type="button" onClick={() => onOpenChange(false)}>
          取消配置
        </button>
      </div>
    ) : null;
  },
  ProFormSwitch: () => null,
  ProFormText: () => null,
}));

vi.mock('@/components/ui', async () => ({
  GuardedModalForm: (await import('@/components/ui/GuardedModalForm'))
    .GuardedModalForm,
  ResponsiveProTable: (await import('@ant-design/pro-components')).ProTable,
  SearchFilterTemplate: ({ extraRight }: { extraRight: React.ReactNode }) => (
    <div>{extraRight}</div>
  ),
  AsyncPopconfirm: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
}));

vi.mock('@/services/roncin/adminService', () => ({
  adminServiceListPermissions: vi.fn(),
  adminServiceCreateRole: vi.fn(),
  adminServiceUpdateRole: vi.fn(),
  adminServiceDeleteRole: vi.fn(),
  adminServiceListRoles: vi.fn(),
}));

import RolesPanel from './roles';

const dictionary = manifestPermissionKeys.map((key) => ({
  key,
  name: key,
  group: '权限字典',
}));
function userWithCapabilities(keys: string[], system = false): API.CurrentUser {
  return {
    id: 'admin-1',
    username: 'admin',
    currentOrganization: {
      id: system ? 'system' : 'beijing',
      kind: system
        ? AuthOrganizationKind.ORGANIZATION_KIND_SYSTEM
        : AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
    },
    permissionCapabilities: keys.map((key) => ({
      key,
      dataScope: system ? 'all' : 'organization',
    })),
  };
}
async function openCreate() {
  await waitFor(() =>
    expect(screen.getByRole('button', { name: /新增角色/ })).toBeEnabled(),
  );
  fireEvent.click(screen.getByRole('button', { name: /新增角色/ }));
}
async function saveRole() {
  await act(async () =>
    fireEvent.click(screen.getByRole('button', { name: '保存角色' })),
  );
}

describe('角色可授予权限与真实权限清单', () => {
  beforeEach(() => {
    tableState.role = undefined;
    formState.onFinish = undefined;
    vi.mocked(adminServiceListPermissions)
      .mockReset()
      .mockResolvedValue({ data: dictionary });
    vi.mocked(adminServiceCreateRole)
      .mockReset()
      .mockResolvedValue({ data: { id: 'role-new' } });
    vi.mocked(adminServiceUpdateRole)
      .mockReset()
      .mockResolvedValue({ data: { id: 'role-existing' } });
  });

  it('真实工作台能力夹具覆盖完整 281 项权限且边界计数一致', () => {
    expect(manifestPermissionKeys).toHaveLength(281);
    expect(capabilityFixture.company).toHaveLength(263);
    expect(capabilityFixture.system).toHaveLength(44);
    expect(
      new Set([...capabilityFixture.company, ...capabilityFixture.system]),
    ).toEqual(new Set(manifestPermissionKeys));
  });

  it.each([
    {
      label: '北京公司',
      system: false,
      keys: capabilityFixture.company,
      count: 263,
    },
    {
      label: '系统管理',
      system: true,
      keys: capabilityFixture.system,
      count: 44,
    },
  ])(
    '$label 的全选读写只提交当前工作台全部 $count 项权限',
    async ({ system, keys, count }) => {
      renderWithApp(<RolesPanel />, {
        user: userWithCapabilities(keys, system),
      });
      await openCreate();
      expect(
        screen.getByText(
          `仅展示你在当前工作台可授予的 ${count} 项权限；「全选读写」选择此范围内的全部权限。`,
        ),
      ).toBeInTheDocument();
      fireEvent.click(screen.getByRole('button', { name: /全选读写/ }));
      await saveRole();
      await waitFor(() =>
        expect(adminServiceCreateRole).toHaveBeenCalledOnce(),
      );
      expect(vi.mocked(adminServiceCreateRole).mock.calls[0][0]).toEqual({
        code: '',
        name: '管理员',
        permissionKeys: keys,
      });
      expect(
        screen.queryByRole('button', { name: '保存角色' }),
      ).not.toBeInTheDocument();
    },
  );

  it('普通用户只选有效能力，self 与未知范围不会因旧权限集而放行', async () => {
    const keys = [
      'system.role.create',
      'system.role.read',
      'system.organization.read',
      'system.permission.read',
    ];
    const user = userWithCapabilities(keys);
    user.permissions = [...manifestPermissionKeys];
    user.permissionCapabilities?.push(
      { key: 'system.user.read', dataScope: 'self' },
      { key: 'system.user.create', dataScope: 'unknown' },
    );
    if (user.permissionCapabilities?.[0])
      user.permissionCapabilities[0].dataScope = 'organization_tree';
    if (user.permissionCapabilities?.[1])
      user.permissionCapabilities[1].dataScope = 'all';
    renderWithApp(<RolesPanel />, { user });
    await openCreate();
    expect(screen.getByText('已选 0 / 4 项')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /全选读写/ }));
    await saveRole();
    expect(
      new Set(
        vi.mocked(adminServiceCreateRole).mock.calls[0][0].permissionKeys,
      ),
    ).toEqual(new Set(keys));
  });

  it('缺少能力时不使用 permissions 或管理员角色回退，也不请求字典', () => {
    renderWithApp(<RolesPanel />, {
      user: {
        ...userWithCapabilities([]),
        permissionCapabilities: undefined,
        permissions: [...manifestPermissionKeys],
        roleScopes: [{ roleCode: 'administrator', roleName: '系统管理员' }],
      },
    });
    expect(
      screen.queryByRole('button', { name: /新增角色/ }),
    ).not.toBeInTheDocument();
    expect(adminServiceListPermissions).not.toHaveBeenCalled();
  });

  it('字典加载或重新加载失败时禁止保存，恢复后保留选择并允许继续', async () => {
    vi.mocked(adminServiceListPermissions).mockRejectedValueOnce(
      new Error('权限字典请求失败'),
    );
    const { queryClient } = renderWithApp(<RolesPanel />, {
      user: userWithCapabilities(capabilityFixture.company),
    });
    await waitFor(() =>
      expect(queryClient.getQueryState(['admin', 'permissions'])?.status).toBe(
        'error',
      ),
    );
    expect(screen.getByRole('button', { name: /新增角色/ })).toBeDisabled();
    await act(async () =>
      expect(await formState.onFinish?.({ name: '管理员' })).toBe(false),
    );
    expect(adminServiceCreateRole).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole('button', { name: /刷新/ }));
    await openCreate();
    fireEvent.click(screen.getByRole('button', { name: /全选读写/ }));
    vi.mocked(adminServiceListPermissions).mockRejectedValueOnce(
      new Error('权限字典重新加载失败'),
    );
    fireEvent.click(screen.getByRole('button', { name: /刷新/ }));
    expect(
      await screen.findByText(
        '加载可授予权限失败，暂不能保存，请重新加载权限后重试。',
      ),
    ).toBeInTheDocument();
    expect(screen.getByText('已选 263 / 263 项')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '保存角色' })).toBeDisabled();
    await act(async () =>
      expect(await formState.onFinish?.({ name: '管理员' })).toBe(false),
    );
    expect(adminServiceCreateRole).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole('button', { name: '重新加载权限' }));
    await waitFor(() =>
      expect(screen.getByRole('button', { name: '保存角色' })).toBeEnabled(),
    );
    await saveRole();
    expect(
      vi.mocked(adminServiceCreateRole).mock.calls[0][0].permissionKeys,
    ).toEqual(capabilityFixture.company);
  });

  it('正常编辑保留原权限并可保存', async () => {
    tableState.role = {
      id: 'role-existing',
      code: 'manager',
      name: '公司管理员',
      enabled: true,
      permissionKeys: capabilityFixture.company,
    };
    renderWithApp(<RolesPanel />, {
      user: userWithCapabilities(capabilityFixture.company),
    });
    await waitFor(() =>
      expect(adminServiceListPermissions).toHaveBeenCalledOnce(),
    );
    fireEvent.click(screen.getByRole('button', { name: /编辑/ }));
    await waitFor(() =>
      expect(screen.getByRole('button', { name: '保存角色' })).toBeEnabled(),
    );
    await saveRole();
    expect(vi.mocked(adminServiceUpdateRole).mock.calls[0]).toEqual([
      { id: 'role-existing' },
      {
        id: 'role-existing',
        name: '管理员',
        enabled: true,
        permissionKeys: capabilityFixture.company,
      },
    ]);
  });

  it('编辑包含不可授予权限的角色时，批量操作保留事实并阻止直接提交', async () => {
    const unavailable = 'system.user.reset_password';
    tableState.role = {
      id: 'role-existing',
      code: 'manager',
      name: '公司管理员',
      permissionKeys: ['system.platform.access', unavailable],
    };
    renderWithApp(<RolesPanel />, {
      user: userWithCapabilities(capabilityFixture.company),
    });
    await waitFor(() =>
      expect(adminServiceListPermissions).toHaveBeenCalledOnce(),
    );
    fireEvent.click(screen.getByRole('button', { name: /编辑/ }));
    expect(
      await screen.findByText(
        `以下权限已保留，请由具备相应权限的管理员处理：${unavailable}`,
      ),
    ).toBeInTheDocument();
    for (const action of [/全选读写/, /全选只读/, /清空/]) {
      fireEvent.click(screen.getByRole('button', { name: action }));
      expect(screen.getByRole('button', { name: '保存角色' })).toBeDisabled();
      expect(
        screen.getByText(
          `以下权限已保留，请由具备相应权限的管理员处理：${unavailable}`,
        ),
      ).toBeInTheDocument();
      await act(async () =>
        expect(await formState.onFinish?.({ name: '管理员' })).toBe(false),
      );
    }
    expect(adminServiceUpdateRole).not.toHaveBeenCalled();
    expect(tableState.role.permissionKeys).toEqual([
      'system.platform.access',
      unavailable,
    ]);
  });

  it('同一工作台能力变化后立即重算候选，已选越界项阻止保存', async () => {
    function CapabilityChangeHarness() {
      const { setInitialState } = useInitialState();
      return (
        <>
          <button
            type="button"
            onClick={() =>
              setInitialState({
                currentUser: userWithCapabilities([
                  'system.role.create',
                  'system.role.read',
                  'system.organization.read',
                  'system.permission.read',
                ]),
              })
            }
          >
            更新当前能力
          </button>
          <RolesPanel />
        </>
      );
    }
    renderWithApp(<CapabilityChangeHarness />, {
      user: userWithCapabilities(capabilityFixture.company),
    });
    await openCreate();
    fireEvent.click(screen.getByRole('button', { name: /全选读写/ }));
    fireEvent.click(screen.getByRole('button', { name: '更新当前能力' }));
    expect(
      screen.getByText(
        '仅展示你在当前工作台可授予的 4 项权限；「全选读写」选择此范围内的全部权限。',
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '保存角色' })).toBeDisabled();
    await act(async () =>
      expect(await formState.onFinish?.({ name: '管理员' })).toBe(false),
    );
    expect(adminServiceCreateRole).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: /全选读写/ }));
    await saveRole();
    expect(
      vi.mocked(adminServiceCreateRole).mock.calls[0][0].permissionKeys,
    ).toHaveLength(4);
  });

  it('真实工作区切换重建角色表单，公司选择不带入系统管理的 44 项候选', async () => {
    function WorkspaceChangeHarness() {
      const { setInitialState } = useInitialState();
      return (
        <>
          <button
            type="button"
            onClick={() =>
              setInitialState({
                currentUser: userWithCapabilities(
                  capabilityFixture.system,
                  true,
                ),
              })
            }
          >
            切换系统管理
          </button>
          <OrganizationWorkspace>
            <RolesPanel />
          </OrganizationWorkspace>
        </>
      );
    }
    renderWithApp(<WorkspaceChangeHarness />, {
      user: userWithCapabilities(capabilityFixture.company),
    });
    await openCreate();
    fireEvent.click(screen.getByRole('button', { name: /全选读写/ }));
    fireEvent.click(screen.getByRole('button', { name: '切换系统管理' }));
    expect(
      screen.queryByRole('button', { name: '保存角色' }),
    ).not.toBeInTheDocument();
    await openCreate();
    expect(screen.getByText('已选 1 / 44 项')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /全选读写/ }));
    await saveRole();
    expect(
      vi.mocked(adminServiceCreateRole).mock.calls[0][0].permissionKeys,
    ).toEqual(capabilityFixture.system);
    expect(adminServiceListPermissions).toHaveBeenCalledTimes(2);
  });
});
