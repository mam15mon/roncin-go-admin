import { describe, expect, it } from 'vitest';
import {
  feeRowsBelongToOrganization,
  hasMixedBillDirections,
  resolveSingleBillCreationOrganization,
} from './index';

describe('费用台账建账所属公司', () => {
  it('只接受每行均有且完全相同的所属公司', () => {
    expect(
      resolveSingleBillCreationOrganization([
        { organizationId: 'A' },
        { organizationId: 'A' },
      ] as API.FeeLedgerItem[]),
    ).toBe('A');
    expect(
      resolveSingleBillCreationOrganization([
        { organizationId: 'A' },
        { organizationId: 'B' },
      ] as API.FeeLedgerItem[]),
    ).toBeUndefined();
    expect(
      resolveSingleBillCreationOrganization([
        { organizationId: 'A' },
        {},
      ] as API.FeeLedgerItem[]),
    ).toBeUndefined();
  });

  it('标签写入仅接受当前筛选公司的同组织费用', () => {
    expect(
      feeRowsBelongToOrganization(
        [
          { organizationId: 'A' },
          { organizationId: 'A' },
        ] as API.FeeLedgerItem[],
        'A',
      ),
    ).toBe(true);
    expect(
      feeRowsBelongToOrganization(
        [
          { organizationId: 'A' },
          { organizationId: 'B' },
        ] as API.FeeLedgerItem[],
        'A',
      ),
    ).toBe(false);
    expect(
      feeRowsBelongToOrganization(
        [{ organizationId: 'A' }] as API.FeeLedgerItem[],
        undefined,
      ),
    ).toBe(false);
  });

  it('普通账单提前识别混合收付方向且不限制同方向多选', () => {
    expect(
      hasMixedBillDirections([
        { direction: 'RECEIVABLE' },
        { direction: 'PAYABLE' },
      ] as API.FeeLedgerItem[]),
    ).toBe(true);
    expect(
      hasMixedBillDirections([
        { direction: 'RECEIVABLE' },
        { direction: 'RECEIVABLE' },
      ] as API.FeeLedgerItem[]),
    ).toBe(false);
  });
});
