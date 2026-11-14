import { describe, expect, it } from 'vitest';
import {
  FinanceBillFeeLockFact,
  FinanceBillInvoiceProgress,
  FinanceBillStatus,
} from '@/enums.generated';
import {
  billFeeLockFactLabels,
  billInvoiceProgressMeta,
  billStatusMeta,
  isBillFieldProtected,
} from './index';

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

describe('账单费用锁定事实', () => {
  it('草稿对冲只锁定费用增删，不保护账单字段', () => {
    const draftOnly = {
      feeLockFacts: [
        FinanceBillFeeLockFact.FINANCE_BILL_FEE_LOCK_FACT_NETTING_DRAFT,
      ],
    };
    expect(isBillFieldProtected(draftOnly)).toBe(false);
    expect(billFeeLockFactLabels(draftOnly)).toEqual(['草稿对冲']);
    expect(
      isBillFieldProtected({
        feeLockFacts: [
          FinanceBillFeeLockFact.FINANCE_BILL_FEE_LOCK_FACT_NETTING_DRAFT,
          FinanceBillFeeLockFact.FINANCE_BILL_FEE_LOCK_FACT_INVOICE,
        ],
      }),
    ).toBe(true);
  });

  it('省略锁定事实视为无事实', () => {
    expect(isBillFieldProtected({})).toBe(false);
    expect(billFeeLockFactLabels(undefined)).toEqual([]);
  });

  it('零额无需开票不伪装为已开票', () => {
    expect(
      billInvoiceProgressMeta[
        FinanceBillInvoiceProgress.FINANCE_BILL_INVOICE_PROGRESS_NOT_REQUIRED
      ].text,
    ).toBe('无需开票');
  });
});
