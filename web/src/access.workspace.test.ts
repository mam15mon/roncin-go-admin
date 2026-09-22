import { describe, expect, it } from 'vitest';
import access from './access';
import { AuthOrganizationKind, BackgroundTaskKind } from './enums.generated';

function workspace(kind: number, id: string, permissions: string[]) {
  return access({
    currentUser: {
      currentOrganization: { kind, id },
      permissions,
      permissionCapabilities: permissions.map((key) => ({
        key,
        dataScope: 'all',
      })),
      roleScopes: [{ dataScope: 'all' }],
    },
  });
}

describe('系统管理与分公司有效业务能力', () => {
  it('系统工作台不展示任何经营读取、导出或办理能力', () => {
    const result = workspace(
      AuthOrganizationKind.ORGANIZATION_KIND_SYSTEM,
      'hq',
      [
        'business.order.se.read',
        'system.finance.bill.read',
        'system.finance.commission.read',
        'system.finance.commission.configure',
        'system.finance.commission.export',
      ],
    );
    expect(result.canReadSEOrders).toBe(false);
    expect(result.canReadFinanceBills).toBe(false);
    expect(result.canExportFinanceCommissions).toBe(false);
    expect(result.canConfigureFinanceCommissions).toBe(false);
    expect(result.canManageFinanceCommissions).toBe(false);
    expect(result.canOrder(1, 'create')).toBe(false);
    expect(result.canOperateBusiness).toBe(false);
    expect(result.canOperateOrganization('company-a')).toBe(false);
  });

  it('分公司工作台只办理当前公司，缺失归属和其他公司拒绝', () => {
    const result = workspace(
      AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
      'company-a',
      ['system.finance.bill.update'],
    );
    expect(result.canUpdateFinanceBills).toBe(true);
    expect(result.canOperateOrganization('company-a')).toBe(true);
    expect(result.canOperateOrganization('company-b')).toBe(false);
    expect(result.canOperateOrganization()).toBe(false);
    expect(result.canManageFinanceCommissions).toBe(false);
  });

  it('公司组织权限可管理本公司内部组织，无需全局数据范围', () => {
    const result = access({
      currentUser: {
        currentOrganization: {
          id: 'company-a',
          kind: AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
        },
        permissionCapabilities: ['read', 'create', 'update'].map(
          (operation) => ({
            key: `system.organization.${operation}`,
            dataScope: 'organization',
          }),
        ),
      },
    });
    expect(result.canReadOrganizations).toBe(true);
    expect(result.canCreateOrganizations).toBe(true);
    expect(result.canUpdateOrganizations).toBe(true);
  });

  it('同一用户切换工作台后以新有效权限重算，返回系统管理不保留办理能力', () => {
    const company = workspace(
      AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
      'company-a',
      ['business.order.se.create', 'business.order.se.read'],
    );
    const headquarters = workspace(
      AuthOrganizationKind.ORGANIZATION_KIND_SYSTEM,
      'hq',
      ['business.order.se.read'],
    );
    expect(company.canOrder(1, 'create')).toBe(true);
    expect(headquarters.canOrder(1, 'create')).toBe(false);
    expect(headquarters.canReadSEOrders).toBe(false);
  });
});

describe('后台任务重试按用途区分', () => {
  it('系统管理允许公共任务重试，业务任务及未知类型不提供入口', () => {
    const hq = workspace(AuthOrganizationKind.ORGANIZATION_KIND_SYSTEM, 'hq', [
      'system.task.requeue',
    ]);
    expect(
      hq.canRequeueTask(
        BackgroundTaskKind.BACKGROUND_TASK_KIND_MASTER_DATA_IMPORT,
      ),
    ).toBe(true);
    expect(
      hq.canRequeueTask(
        BackgroundTaskKind.BACKGROUND_TASK_KIND_DINGTALK_NOTIFICATION,
      ),
    ).toBe(true);
    expect(
      hq.canRequeueTask(BackgroundTaskKind.BACKGROUND_TASK_KIND_ORDER_REMINDER),
    ).toBe(false);
    expect(
      hq.canRequeueTask(BackgroundTaskKind.BACKGROUND_TASK_KIND_INTEGRATION),
    ).toBe(false);
    expect(hq.canRequeueTask()).toBe(false);
    const company = workspace(
      AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
      'company-a',
      ['system.task.requeue'],
    );
    expect(
      company.canRequeueTask(
        BackgroundTaskKind.BACKGROUND_TASK_KIND_INTEGRATION,
      ),
    ).toBe(true);
  });
});

it('公司管理员不能操作全局账号，但保留本公司成员管理', () => {
  const permissions = [
    'system.user.read',
    'system.user.update',
    'system.user.reset_password',
    'system.user.delete',
  ];
  const company = workspace(
    AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
    'company',
    permissions,
  );
  expect(company.canUpdateUsers).toBe(false);
  expect(company.canResetUserPasswords).toBe(false);
  expect(company.canTerminateUsers).toBe(false);
  expect(company.canReadUserMemberships).toBe(true);
  expect(company.canManageUserMemberships).toBe(true);
  const system = workspace(
    AuthOrganizationKind.ORGANIZATION_KIND_SYSTEM,
    'system',
    permissions,
  );
  expect(system.canUpdateUsers).toBe(true);
  expect(system.canResetUserPasswords).toBe(true);
  expect(system.canTerminateUsers).toBe(true);
});

it('公司不得通过外部授权重激活全局账号', () => {
  const permissions = [
    'system.user.authorize_wecom',
    'system.user.authorize_dingtalk',
  ];
  const company = workspace(
    AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
    'company',
    permissions,
  );
  expect(company.canAuthorizeWeComUsers).toBe(false);
  expect(company.canAuthorizeDingTalkUsers).toBe(false);
  const system = workspace(
    AuthOrganizationKind.ORGANIZATION_KIND_SYSTEM,
    'system',
    permissions,
  );
  expect(system.canAuthorizeWeComUsers).toBe(true);
  expect(system.canAuthorizeDingTalkUsers).toBe(true);
});
