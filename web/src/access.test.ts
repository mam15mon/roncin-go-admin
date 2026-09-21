import { describe, expect, it } from 'vitest';
import access from './access';

function currentUser(permissions: string[]) {
  return {
    currentUser: {
      permissions,
      permissionCapabilities: permissions.map((key) => ({
        key,
        dataScope: 'organization',
      })),
      roleScopes: [{ dataScope: 'organization' }],
    } as API.CurrentUser,
  };
}

function currentUserWithSelfScope(permissions: string[]) {
  return {
    currentUser: {
      permissions,
      permissionCapabilities: permissions.map((key) => ({
        key,
        dataScope: 'self',
      })),
      roleScopes: [{ dataScope: 'self' }],
    } as API.CurrentUser,
  };
}

function currentUserWithAllScope(permissions: string[]) {
  return {
    currentUser: {
      permissions,
      permissionCapabilities: permissions.map((key) => ({
        key,
        dataScope: 'all',
      })),
      roleScopes: [{ dataScope: 'all' }],
    } as API.CurrentUser,
  };
}

function currentUserWithOrganizationKind(
  permissions: string[],
  kind: API.Organization['kind'],
) {
  return {
    currentUser: {
      permissions,
      permissionCapabilities: permissions.map((key) => ({
        key,
        dataScope: 'organization',
      })),
      roleScopes: [{ dataScope: 'organization' }],
      currentOrganization: { kind },
    } as API.CurrentUser,
  };
}

describe('组织身份判定（auth/me kind 契约）', () => {
  it('总部工作台判定为总部组织', () => {
    const result = access(
      currentUserWithOrganizationKind([], 1), // ORGANIZATION_KIND_HEADQUARTERS
    );
    expect(result.isHeadquartersOrganization).toBe(true);
  });

  it('公司/部门等分支工作台判定为非总部组织', () => {
    const company = access(currentUserWithOrganizationKind([], 2)); // ORGANIZATION_KIND_COMPANY
    const department = access(currentUserWithOrganizationKind([], 3)); // ORGANIZATION_KIND_DEPARTMENT
    const anonymous = access(currentUser([]));
    expect(company.isHeadquartersOrganization).toBe(false);
    expect(department.isHeadquartersOrganization).toBe(false);
    expect(anonymous.isHeadquartersOrganization).toBe(false);
  });
});

describe('新建订单入口权限（canCreateAnyOrders）', () => {
  it('组织范围且拥有任一业务类型 create 权限时放行新建订单路由', () => {
    const result = access(currentUser(['business.order.se.create']));
    expect(result.canCreateAnyOrders).toBe(true);
  });

  it('只有订单读取权限（如总部只读角色）时不得进入新建订单路由', () => {
    const result = access(currentUser(['business.order.se.read']));
    expect(result.canCreateAnyOrders).toBe(false);
  });

  it('无任何订单权限的总部用户不得进入新建订单路由', () => {
    const result = access(currentUserWithAllScope([]));
    expect(result.canCreateAnyOrders).toBe(false);
  });

  it('仅有本人数据范围时即使持有 create 权限码也不放行', () => {
    const result = access(
      currentUserWithSelfScope(['business.order.se.create']),
    );
    expect(result.canCreateAnyOrders).toBe(false);
  });
});

describe('费用录入工作台权限（canAccessAnyOrderFees）', () => {
  it('持任一业务类型 fee.read 时放行费用录入路由', () => {
    const result = access(currentUser(['business.order.se.fee.read']));
    expect(result.canAccessAnyOrderFees).toBe(true);
  });

  it('仅持 lock 权限的补录审批人同样放行（工作台「前往处理」入口）', () => {
    const result = access(currentUser(['business.order.se.lock']));
    expect(result.canAccessAnyOrderFees).toBe(true);
  });

  it('只有订单读取或创建权限时不得进入费用录入路由', () => {
    const result = access(
      currentUser(['business.order.se.read', 'business.order.se.create']),
    );
    expect(result.canAccessAnyOrderFees).toBe(false);
  });
});

describe('用户页数据源分流权限', () => {
  it('普通组织管理员可读当前组织角色，但无全组织读取与外部成员授权能力', () => {
    const result = access(
      currentUser([
        'system.role.read',
        'system.user.read',
        'system.user.update',
      ]),
    );
    expect(result.canReadRoles).toBe(true);
    expect(result.canReadOrganizations).toBe(false);
    expect(result.canReadAllUserMemberships).toBe(false);
    expect(result.canManageUserMemberships).toBe(false);
    expect(result.canAuthorizeWeComUsers).toBe(false);
    expect(result.canAuthorizeDingTalkUsers).toBe(false);
  });

  it('全局管理员具备全组织读取、成员关系与外部成员授权能力', () => {
    const result = access(
      currentUserWithAllScope([
        'system.role.read',
        'system.organization.read',
        'system.user.read',
        'system.user.update',
        'system.user.authorize_wecom',
        'system.user.authorize_dingtalk',
      ]),
    );
    expect(result.canReadRoles).toBe(true);
    expect(result.canReadOrganizations).toBe(true);
    expect(result.canReadAllUserMemberships).toBe(true);
    expect(result.canManageUserMemberships).toBe(true);
    expect(result.canAuthorizeWeComUsers).toBe(true);
    expect(result.canAuthorizeDingTalkUsers).toBe(true);
  });
});

describe('提成导出权限', () => {
  it('组织范围且拥有导出权限时允许显示导出按钮', () => {
    expect(
      access(currentUser(['system.finance.commission.export']))
        .canExportFinanceCommissions,
    ).toBe(true);
  });

  it('只有提成读取权限时不允许显示导出按钮', () => {
    expect(
      access(currentUser(['system.finance.commission.read']))
        .canExportFinanceCommissions,
    ).toBe(false);
  });

  it('已停用的本人范围不能显示组织提成导出按钮', () => {
    expect(
      access(currentUserWithSelfScope(['system.finance.commission.export']))
        .canExportFinanceCommissions,
    ).toBe(false);
  });
});

describe('钉钉邀请与注册审批权限', () => {
  it('组织范围且拥有邀请管理权限时可管理钉钉邀请，并计入用户管理聚合', () => {
    const result = access(
      currentUser([
        'system.user.read',
        'system.user.dingtalk_invitation.manage',
      ]),
    );
    expect(result.canManageDingTalkInvitations).toBe(true);
    expect(result.canManageUsers).toBe(true);
  });

  it('只有用户读取权限或本人范围时不可管理钉钉邀请', () => {
    expect(
      access(currentUser(['system.user.read'])).canManageDingTalkInvitations,
    ).toBe(false);
    expect(
      access(
        currentUserWithSelfScope(['system.user.dingtalk_invitation.manage']),
      ).canManageDingTalkInvitations,
    ).toBe(false);
  });
});

describe('角色删除权限', () => {
  it('组织范围且拥有删除权限时允许删除角色，并计入角色管理聚合', () => {
    const result = access(currentUser(['system.role.delete']));
    expect(result.canDeleteRoles).toBe(true);
    expect(result.canManageRoles).toBe(true);
  });

  it('只有删除权限但超出组织范围时不可删除角色', () => {
    expect(
      access(currentUserWithSelfScope(['system.role.delete'])).canDeleteRoles,
    ).toBe(false);
  });
});

describe('权限范围来源一致性', () => {
  it('缺少能力契约时不能使用旧权限与角色范围放行', () => {
    const result = access({
      currentUser: {
        permissions: ['system.user.create'],
        roleScopes: [{ dataScope: 'all' }],
      },
    });
    expect(result.canCreateUsers).toBe(false);
  });
  it('不能借无关权限的 all 范围访问全局用户管理', () => {
    const result = access({
      currentUser: {
        permissionCapabilities: [
          { key: 'system.user.update', dataScope: 'organization' },
          { key: 'system.role.read', dataScope: 'all' },
        ],
        roleScopes: [{ dataScope: 'all' }],
      },
    });
    expect(result.canUpdateUsers).toBe(true);
    expect(result.canManageUserMemberships).toBe(false);
  });
  it('self 能力即使与其他组织能力并存也不允许组织资源', () => {
    const result = access({
      currentUser: {
        permissionCapabilities: [
          { key: 'system.user.create', dataScope: 'self' },
          { key: 'system.finance.bill.read', dataScope: 'self' },
          { key: 'business.order.se.create', dataScope: 'self' },
          { key: 'system.role.read', dataScope: 'all' },
        ],
      },
    });
    expect(result.canCreateUsers).toBe(false);
    expect(result.canReadFinanceBills).toBe(false);
    expect(result.canCreateAnyOrders).toBe(false);
  });
});
