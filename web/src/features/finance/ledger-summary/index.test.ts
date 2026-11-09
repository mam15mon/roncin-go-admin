import { describe, expect, it } from 'vitest';
import { OrderFeeDirection } from '@/enums.generated';
import {
  billLedgerSummaryConfig,
  cashflowLedgerSummaryConfig,
  feeLedgerSummaryConfig,
  invoiceLedgerSummaryConfig,
  mapMatchedSummary,
  nettingLedgerSummaryConfig,
  orderFeeSummaryConfig,
  verificationLedgerSummaryConfig,
} from './index';

const shared = {
  direction: 'RECEIVABLE',
  currency: 'USD',
  baseCurrency: 'CNY',
};

describe('六类财务字段投影', () => {
  it('费用/账单/发票用totalAmount，收付/核销用amount和baseAmount，对冲不猜方向', () => {
    for (const config of [
      feeLedgerSummaryConfig,
      billLedgerSummaryConfig,
      invoiceLedgerSummaryConfig,
    ]) {
      expect(
        config.projectRow({
          ...shared,
          totalAmount: '12.5',
          baseCurrencyAmount: '90',
        }),
      ).toMatchObject({
        amount: '12.5',
        baseAmount: '90',
        unconvertedCount: 0,
      });
    }
    for (const config of [
      cashflowLedgerSummaryConfig,
      verificationLedgerSummaryConfig,
    ]) {
      expect(
        config.projectRow({ ...shared, amount: '7', baseAmount: '50' }),
      ).toMatchObject({ amount: '7', baseAmount: '50' });
    }
    expect(
      nettingLedgerSummaryConfig.projectRow({
        currency: 'EUR',
        amount: '9',
        baseCurrency: 'CNY',
        baseCurrencyAmount: '70',
      }),
    ).toMatchObject({ direction: '', amount: '9', baseAmount: '70' });
    expect(
      orderFeeSummaryConfig.projectRow({
        ...shared,
        direction: OrderFeeDirection.ORDER_FEE_DIRECTION_RECEIVABLE,
        totalAmount: '12',
        baseCurrencyAmount: '80',
      }),
    ).toMatchObject({ direction: 'RECEIVABLE' });
    for (const config of [
      billLedgerSummaryConfig,
      invoiceLedgerSummaryConfig,
      cashflowLedgerSummaryConfig,
      verificationLedgerSummaryConfig,
      nettingLedgerSummaryConfig,
    ])
      expect(config.profit).toBeUndefined();
  });

  it('发票缺折本币保持缺失，显式零不是未折算', () => {
    const missing = invoiceLedgerSummaryConfig.projectRow({
      ...shared,
      totalAmount: '3',
    });
    expect(missing.baseAmount).toBeUndefined();
    expect(missing.unconvertedCount).toBe(1);
    expect(
      invoiceLedgerSummaryConfig.projectRow({
        ...shared,
        totalAmount: '3',
        baseCurrencyAmount: '0',
      }).unconvertedCount,
    ).toBe(0);
  });

  it('正式筛选汇总还原省略count为0，缺失整个汇总必须拒绝，不能回退本页', () => {
    expect(mapMatchedSummary({})).toEqual({ count: 0, amounts: [] });
    expect(
      mapMatchedSummary({
        count: '20',
        amounts: [{ currency: 'USD', amount: '123', unconvertedCount: '2' }],
      }),
    ).toMatchObject({
      count: 20,
      amounts: [{ direction: '', unconvertedCount: 2 }],
    });
    expect(() => mapMatchedSummary(undefined)).toThrow('缺少筛选汇总');
  });
});
