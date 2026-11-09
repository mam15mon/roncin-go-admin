import { describe, expect, it } from 'vitest';
import {
  hasMixedBillDirections,
  resolveSingleBillCreationOrganization,
} from './index';

describe('费用台账建账所属公司', () => {
  it('只接受每行都属于当前工作区的公司', () => {
    expect(
      resolveSingleBillCreationOrganization(
        [
          { organizationId: 'A' },
          { organizationId: 'A' },
        ] as API.FeeLedgerItem[],
        'A',
      ),
    ).toBe('A');
    expect(
      resolveSingleBillCreationOrganization(
        [
          { organizationId: 'A' },
          { organizationId: 'B' },
        ] as API.FeeLedgerItem[],
        'A',
      ),
    ).toBeUndefined();
    expect(
      resolveSingleBillCreationOrganization(
        [{ organizationId: 'A' }, {}] as API.FeeLedgerItem[],
        'A',
      ),
    ).toBeUndefined();
  });

  it('单一外公司、缺当前公司和空选中集合都拒绝', () => {
    expect(
      resolveSingleBillCreationOrganization([{ organizationId: 'B' }], 'A'),
    ).toBeUndefined();
    expect(
      resolveSingleBillCreationOrganization([{ organizationId: 'A' }]),
    ).toBeUndefined();
    expect(resolveSingleBillCreationOrganization([], 'A')).toBeUndefined();
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
