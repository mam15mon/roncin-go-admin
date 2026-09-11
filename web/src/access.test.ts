import { describe, expect, it } from 'vitest';
import access from './access';

function currentUser(permissions: string[]) {
  return {
    currentUser: {
      permissions,
      roleScopes: [{ dataScope: 'organization' }],
    } as API.CurrentUser,
  };
}

function currentUserWithSelfScope(permissions: string[]) {
  return {
    currentUser: {
      permissions,
      roleScopes: [{ dataScope: 'self' }],
    } as API.CurrentUser,
  };
}

function currentUserWithAllScope(permissions: string[]) {
  return {
    currentUser: {
      permissions,
      roleScopes: [{ dataScope: 'all' }],
    } as API.CurrentUser,
  };
}

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

  it('本人范围在组织维度覆盖当前组织时允许显示导出按钮', () => {
    expect(
      access(currentUserWithSelfScope(['system.finance.commission.export']))
        .canExportFinanceCommissions,
    ).toBe(true);
  });
});
