import { describe, expect, it } from 'vitest';
import {
  buildVerificationAllocations,
  sumVerificationAmounts,
} from './verification-allocation';

describe('buildVerificationAllocations', () => {
  it('支持一笔资金自动拆分到多张账单', () => {
    expect(
      buildVerificationAllocations(
        [{ id: 'cash-1', balance: '100.00000000' }],
        [
          { id: 'bill-1', balance: '30.00000000' },
          { id: 'bill-2', balance: '70.00000000' },
        ],
      ),
    ).toEqual([
      { cashflowId: 'cash-1', billId: 'bill-1', amount: '30.00000000' },
      { cashflowId: 'cash-1', billId: 'bill-2', amount: '70.00000000' },
    ]);
  });

  it('支持多笔资金补齐一张账单且不产生浮点尾差', () => {
    const allocations = buildVerificationAllocations(
      [
        { id: 'cash-1', balance: '0.10000000' },
        { id: 'cash-2', balance: '0.20000000' },
      ],
      [{ id: 'bill-1', balance: '0.30000000' }],
    );
    expect(allocations).toHaveLength(2);
    expect(sumVerificationAmounts(allocations.map((item) => item.amount)).toFixed(8)).toBe(
      '0.30000000',
    );
  });
});
