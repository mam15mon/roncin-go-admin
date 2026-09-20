import { describe, expect, it } from 'vitest';
import access from './access';
import { AuthOrganizationKind, BackgroundTaskKind } from './enums.generated';

function workspace(kind: number, id: string, permissions: string[]) {
  return access({
    currentUser: {
      currentOrganization: { kind, id },
      permissions,
      roleScopes: [{ dataScope: 'all' }],
    },
  });
}

describe('总部与分公司有效业务能力', () => {
  it('总部保留经营读取、导出和提成配置，不展示办理能力', () => {
    const result = workspace(
      AuthOrganizationKind.ORGANIZATION_KIND_HEADQUARTERS,
      'hq',
      [
        'business.order.se.read',
        'system.finance.bill.read',
        'system.finance.commission.read',
        'system.finance.commission.configure',
        'system.finance.commission.export',
      ],
    );
    expect(result.canReadSEOrders).toBe(true);
    expect(result.canReadFinanceBills).toBe(true);
    expect(result.canExportFinanceCommissions).toBe(true);
    expect(result.canConfigureFinanceCommissions).toBe(true);
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

  it('同一用户切换工作台后以新有效权限重算，返回总部不保留办理能力', () => {
    const company = workspace(
      AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
      'company-a',
      ['business.order.se.create', 'business.order.se.read'],
    );
    const headquarters = workspace(
      AuthOrganizationKind.ORGANIZATION_KIND_HEADQUARTERS,
      'hq',
      ['business.order.se.read'],
    );
    expect(company.canOrder(1, 'create')).toBe(true);
    expect(headquarters.canOrder(1, 'create')).toBe(false);
    expect(headquarters.canReadSEOrders).toBe(true);
  });
});

describe('后台任务重试按用途区分', () => {
  it('总部允许公共任务重试，业务任务及未知类型不提供入口', () => {
    const hq = workspace(
      AuthOrganizationKind.ORGANIZATION_KIND_HEADQUARTERS,
      'hq',
      ['system.task.requeue'],
    );
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
