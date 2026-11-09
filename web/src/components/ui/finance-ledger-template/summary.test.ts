import { describe, expect, it } from 'vitest';
import { aggregateLedgerAmounts, sumAmountsByCurrency } from './summary';

describe('台账金额累加精度', () => {
  it('合法大额正负相抵后仍保留小数，不受 Decimal 默认20位有效数截断', () => {
    const amounts = ['10000000000000000000.12', '-10000000000000000000'].map(
      (amount) => ({
        direction: 'RECEIVABLE',
        currency: 'CNY',
        amount,
        baseCurrency: 'CNY',
        baseAmount: amount,
      }),
    );
    expect(aggregateLedgerAmounts(amounts).amounts[0]).toMatchObject({
      amount: '0.12',
      baseAmount: '0.12',
    });
    expect(sumAmountsByCurrency(amounts, false).get('CNY')?.toFixed()).toBe(
      '0.12',
    );
    expect(sumAmountsByCurrency(amounts, true).get('CNY')?.toFixed()).toBe(
      '0.12',
    );
  });
});
