import { describe, expect, it } from 'vitest';
import { FinanceBillStatus } from '@/enums.generated';
import { statusOptions } from './billConstants';

describe('账单状态选项', () => {
  it('仅包含后端账单状态机支持的状态', () => {
    expect(Object.keys(statusOptions).map(Number)).toEqual([
      FinanceBillStatus.FINANCE_BILL_STATUS_DRAFT,
      FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED,
      FinanceBillStatus.FINANCE_BILL_STATUS_CANCELLED,
    ]);
  });
});
