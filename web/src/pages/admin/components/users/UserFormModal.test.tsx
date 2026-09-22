import { act, render } from '@testing-library/react';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const serviceMocks = vi.hoisted(() => ({
  authorizeDingTalkUser: vi.fn(),
  authorizeWeComUser: vi.fn(),
  createUser: vi.fn(),
  deleteUserMembership: vi.fn(),
  listOrganizationRoles: vi.fn(),
  listUserMemberships: vi.fn(),
  updateUser: vi.fn(),
}));

const modalState = vi.hoisted(() => ({
  props: undefined as Record<string, unknown> | undefined,
}));

const searchableSelectState = vi.hoisted(
  () => new Map<string, Record<string, unknown>>(),
);

vi.mock('@ant-design/pro-components', () => ({
  ModalForm: (props: Record<string, unknown>) => {
    modalState.props = props;
    return <div>{props.children as React.ReactNode}</div>;
  },
  ProFormText: () => null,
}));

vi.mock('@/components/ui', () => ({
  ProFormSearchableSelect: (props: Record<string, unknown>) => {
    searchableSelectState.set(String(props.name), props);
    return null;
  },
}));

vi.mock('antd', () => ({
  Alert: () => null,
  App: { useApp: () => ({ message: { success: vi.fn(), error: vi.fn() } }) },
  Button: ({ children }: { children: React.ReactNode }) => (
    <button type="button">{children}</button>
  ),
  Popconfirm: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  Space: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Table: () => null,
  Tag: ({ children }: { children: React.ReactNode }) => <span>{children}</span>,
  Typography: {
    Text: ({ children }: { children?: React.ReactNode }) => (
      <span>{children}</span>
    ),
  },
}));

vi.mock('@/services/roncin/adminService', () => ({
  adminServiceAuthorizeDingTalkUser: serviceMocks.authorizeDingTalkUser,
  adminServiceAuthorizeWeComUser: serviceMocks.authorizeWeComUser,
  adminServiceCreateUser: serviceMocks.createUser,
  adminServiceDeleteUserMembership: serviceMocks.deleteUserMembership,
  adminServiceListOrganizationRoles: serviceMocks.listOrganizationRoles,
  adminServiceListUserMemberships: serviceMocks.listUserMemberships,
  adminServiceUpdateUser: serviceMocks.updateUser,
}));

vi.mock('./UserMembershipModal', () => ({
  default: () => null,
}));

import UserFormModal from './UserFormModal';

const currentOrgRoles: API.AdminRole[] = [
  { id: 'role-1', name: '订单操作员', code: 'order_operator' },
];

const pendingDingTalkUser: API.AdminUser = {
  id: 'user-9',
  displayName: '钉钉新成员',
  status: 2,
  dingtalkUnionid: 'union-1',
  dingtalkName: '钉钉新成员',
};

const normalUser: API.AdminUser = {
  id: 'user-2',
  username: 'zhangsan',
  displayName: '张三',
  status: 1,
};

function renderModal(
  editing: API.AdminUser | undefined,
  flags: Record<string, unknown>,
) {
  render(
    <UserFormModal
      open
      onOpenChange={vi.fn()}
      editing={editing}
      formRef={{ current: undefined }}
      roles={currentOrgRoles}
      organizations={[{ id: 'org-1', name: '天津分公司', code: 'TJ' }]}
      canReadUserMemberships={false}
      canManageUserMemberships={false}
      canUpdateUserProfile={true}
      canAuthorizeWeComUsers={false}
      canAuthorizeDingTalkUsers={false}
      currentUserId="user-1"
      defaultOrganizationId="org-1"
      onReload={vi.fn()}
      {...flags}
    />,
  );
}

function requiredSelect(name: string) {
  const props = searchableSelectState.get(name);
  if (!props) {
    throw new Error(`未找到字段 ${name} 的选择组件`);
  }
  return props;
}

async function submitForm(values: Record<string, unknown>) {
  const onFinish = modalState.props?.onFinish;
  if (typeof onFinish !== 'function') {
    throw new Error('未找到用户表单提交处理器');
  }
  await act(async () => {
    await (onFinish as (input: Record<string, unknown>) => Promise<boolean>)(
      values,
    );
  });
}

describe('UserFormModal 角色数据源与外部授权流程分流', () => {
  beforeEach(() => {
    modalState.props = undefined;
    searchableSelectState.clear();
    for (const mock of Object.values(serviceMocks)) {
      mock.mockReset();
      mock.mockResolvedValue({ data: [] });
    }
  });

  it('普通编辑流程只用当前组织角色，不触发跨组织角色请求', async () => {
    renderModal(normalUser, {});

    const roleSelect = requiredSelect('roleIds');
    expect(roleSelect.options).toEqual([
      expect.objectContaining({ value: 'role-1' }),
    ]);
    expect(searchableSelectState.get('organizationId')).toBeUndefined();
    expect(serviceMocks.listOrganizationRoles).not.toHaveBeenCalled();

    await submitForm({ displayName: '张三', roleIds: ['role-1'] });
    expect(serviceMocks.updateUser).toHaveBeenCalled();
    expect(serviceMocks.listOrganizationRoles).not.toHaveBeenCalled();
  });

  it('编辑时按锚定成员关系所在组织拉取角色选项', async () => {
    serviceMocks.listUserMemberships.mockResolvedValue({
      data: [
        { id: 'm-1', organizationId: 'org-2', primary: true, enabled: true },
      ],
    });
    serviceMocks.listOrganizationRoles.mockResolvedValue({
      data: [{ id: 'role-9', name: '锚定组织角色', code: 'anchor_role' }],
    });

    renderModal(normalUser, { canReadUserMemberships: true });

    await act(async () => {});
    expect(serviceMocks.listOrganizationRoles).toHaveBeenCalledWith({
      organizationId: 'org-2',
    });
    const roleSelect = requiredSelect('roleIds');
    expect(roleSelect.options).toEqual([
      expect.objectContaining({ value: 'role-9' }),
    ]);

    await submitForm({ displayName: '张三', roleIds: ['role-9'] });
    expect(serviceMocks.updateUser).toHaveBeenCalled();
  });

  it('编辑时无 primary 成员关系则取第一条启用关系的组织', async () => {
    serviceMocks.listUserMemberships.mockResolvedValue({
      data: [
        { id: 'm-1', organizationId: 'org-9', primary: false, enabled: false },
        { id: 'm-2', organizationId: 'org-3', primary: false, enabled: true },
      ],
    });

    renderModal(normalUser, { canReadUserMemberships: true });

    await act(async () => {});
    expect(serviceMocks.listOrganizationRoles).toHaveBeenCalledWith({
      organizationId: 'org-3',
    });
  });

  it('具备钉钉授权权限时，外部成员流程按目标组织调用 ListOrganizationRoles', async () => {
    renderModal(pendingDingTalkUser, { canAuthorizeDingTalkUsers: true });

    const organizationSelect = requiredSelect('organizationId');
    const fieldProps = organizationSelect.fieldProps as {
      onChange: (organizationId: string) => Promise<void>;
    };
    await act(async () => {
      await fieldProps.onChange('org-2');
    });

    expect(serviceMocks.listOrganizationRoles).toHaveBeenCalledWith({
      organizationId: 'org-2',
    });

    await submitForm({
      organizationId: 'org-2',
      displayName: '钉钉新成员',
      roleIds: ['role-1'],
    });
    expect(serviceMocks.authorizeDingTalkUser).toHaveBeenCalledWith(
      { id: 'user-9' },
      expect.objectContaining({ organizationId: 'org-2' }),
    );
    expect(serviceMocks.updateUser).not.toHaveBeenCalled();
  });

  it('缺少授权权限时外部成员不进入授权流程，回落普通分支且不发跨组织请求', async () => {
    renderModal(pendingDingTalkUser, {});

    expect(searchableSelectState.get('organizationId')).toBeUndefined();
    expect(serviceMocks.listOrganizationRoles).not.toHaveBeenCalled();

    await submitForm({ displayName: '钉钉新成员', roleIds: [] });
    expect(serviceMocks.updateUser).toHaveBeenCalled();
    expect(serviceMocks.authorizeDingTalkUser).not.toHaveBeenCalled();
    expect(serviceMocks.listOrganizationRoles).not.toHaveBeenCalled();
  });

  it('企业微信外部成员按对应权限分流，未授权时不进入授权流程', async () => {
    const pendingWeComUser: API.AdminUser = {
      ...pendingDingTalkUser,
      dingtalkUnionid: undefined,
      wecomUserid: 'wecom-1',
      wecomName: '企微新成员',
    };
    renderModal(pendingWeComUser, { canAuthorizeWeComUsers: true });

    expect(searchableSelectState.get('organizationId')).toBeDefined();

    await submitForm({
      organizationId: 'org-1',
      displayName: '企微新成员',
      roleIds: ['role-1'],
    });
    expect(serviceMocks.authorizeWeComUser).toHaveBeenCalledWith(
      { id: 'user-9' },
      expect.objectContaining({ organizationId: 'org-1' }),
    );
  });
});

it('公司管理员只能管理成员关系，不能提交全局账号资料', async () => {
  renderModal(normalUser, {
    canUpdateUserProfile: false,
    canReadUserMemberships: true,
    canManageUserMemberships: true,
  });
  await act(async () => {});
  expect(modalState.props?.submitter).toBe(false);
  const finish = modalState.props?.onFinish as (
    values: Record<string, unknown>,
  ) => Promise<boolean>;
  expect(
    await finish({
      displayName: '改名',
      email: 'other@example.com',
      roleIds: [],
    }),
  ).toBe(false);
  expect(serviceMocks.updateUser).not.toHaveBeenCalled();
  expect(serviceMocks.listUserMemberships).toHaveBeenCalledWith({
    userId: normalUser.id,
  });
});
