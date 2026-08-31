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

  it('拥有导出权限但没有组织范围时不允许显示导出按钮', () => {
    expect(
      access(currentUserWithSelfScope(['system.finance.commission.export']))
        .canExportFinanceCommissions,
    ).toBe(false);
  });
});
