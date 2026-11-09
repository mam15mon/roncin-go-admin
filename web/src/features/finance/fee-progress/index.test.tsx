import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import {
  getDefaultRowColors,
  RowColorSettings,
} from '@/components/ui/finance-ledger-template';
import { FeeLedgerFinancialProgress } from '@/enums.generated';
import { feeLedgerProgressLabels } from './index';

const expectedLabels = [
  [
    FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_UNBILLED,
    '账单未建立',
  ],
  [
    FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_UNVERIFIED_UNINVOICED,
    '未开票未付款',
  ],
  [
    FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_INVOICED_UNVERIFIED,
    '已开票未付款',
  ],
  [
    FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_INVOICED_PARTIALLY_VERIFIED,
    '已开票部分付款',
  ],
  [
    FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_PARTIALLY_VERIFIED_UNINVOICED,
    '未开票部分付款',
  ],
  [
    FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_VERIFIED_UNINVOICED,
    '未开票已付款',
  ],
  [
    FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_COMPLETED,
    '已开票已付款',
  ],
] as const;

describe('费用财务进度文案', () => {
  it.each(expectedLabels)(
    '状态 %s 明确表达开票和付款事实',
    (progress, label) => {
      expect(feeLedgerProgressLabels[progress].text).toBe(label);
    },
  );

  it('行配色设置与共享进度文案一致', () => {
    render(
      <RowColorSettings
        rowColors={getDefaultRowColors()}
        onColorChange={vi.fn()}
        onResetColors={vi.fn()}
      />,
    );
    for (const [, label] of expectedLabels) {
      expect(screen.getByText(label, { exact: true })).toBeVisible();
    }
    expect(screen.queryByText(/核销/)).not.toBeInTheDocument();
  });
});
