import { describe, expect, it } from 'vitest';
import { FinanceBillStatus } from '@/enums.generated';
import { billStatusMeta } from './index';

describe('账单状态映射', () => {
  it('仅包含免确认后仍会产生的账单状态', () => {
    expect(Object.keys(billStatusMeta).map(Number)).toEqual([
      FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED,
      FinanceBillStatus.FINANCE_BILL_STATUS_CANCELLED,
    ]);
    expect(
      billStatusMeta[FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED].text,
    ).toBe('正常');
  });
});
