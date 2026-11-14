import { describe, expect, it } from 'vitest';
import access from './access';
import { AuthOrganizationKind } from './enums.generated';

function reportAccess(
  kind: number,
  keys: string[],
  dataScope = 'organization',
) {
  return access({
    currentUser: {
      currentOrganization: { id: 'company-a', kind },
      permissionCapabilities: keys.map((key) => ({ key, dataScope })),
    },
  });
}

describe('独立财务报表权限', () => {
  it('只持报表权限可进入公司报表和财务目录，不获得相邻实体权限', () => {
    const result = reportAccess(
      AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
      ['system.finance.report.read'],
    );
    expect(result.canReadFinanceReports).toBe(true);
    expect(result.canAccessFinanceManagement).toBe(true);
    expect(result.canReadFinanceBills).toBe(false);
    expect(result.canReadFinanceFees).toBe(false);
    expect(result.canReadFinanceInvoices).toBe(false);
    expect(result.canReadFinanceCashflows).toBe(false);
  });

  it('相邻读取权限、系统工作区、本人范围均不能放行报表', () => {
    expect(
      reportAccess(AuthOrganizationKind.ORGANIZATION_KIND_COMPANY, [
        'system.finance.bill.read',
        'system.finance.fee.read',
      ]).canReadFinanceReports,
    ).toBe(false);
    expect(
      reportAccess(AuthOrganizationKind.ORGANIZATION_KIND_SYSTEM, [
        'system.finance.report.read',
      ]).canReadFinanceReports,
    ).toBe(false);
    expect(
      reportAccess(
        AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
        ['system.finance.report.read'],
        'self',
      ).canReadFinanceReports,
    ).toBe(false);
  });
});
