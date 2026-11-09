import { useState } from 'react';
import type {
  FinanceLedgerAmount,
  FinanceLedgerMatchedSummary,
  FinanceLedgerSummaryConfig,
} from '@/components/ui/finance-ledger-template';
import { OrderFeeDirection } from '@/enums.generated';

const receivableGroups = [
  { direction: 'RECEIVABLE', label: '应收', color: '#1677ff' },
  { direction: 'PAYABLE', label: '应付', color: '#fa8c16' },
];
const profit = {
  label: '费用利润',
  receivableDirection: 'RECEIVABLE',
  payableDirection: 'PAYABLE',
};

function projectAmount(
  direction: string | undefined,
  currency: string | undefined,
  amount: string | undefined,
  baseCurrency?: string,
  baseAmount?: string,
): FinanceLedgerAmount {
  if (direction === undefined || !currency || amount === undefined)
    throw new Error('财务汇总缺少方向、币种或金额');
  return {
    direction,
    currency,
    amount,
    baseCurrency,
    baseAmount,
    unconvertedCount:
      baseCurrency === undefined || baseAmount === undefined ? 1 : 0,
  };
}

export const feeLedgerSummaryConfig: FinanceLedgerSummaryConfig<API.FeeLedgerItem> =
  {
    groups: receivableGroups,
    profit: { ...profit, label: '费用明细差额' },
    projectRow: (row) =>
      projectAmount(
        row.direction,
        row.currency,
        row.totalAmount,
        row.baseCurrency,
        row.baseCurrencyAmount,
      ),
  };
export const billLedgerSummaryConfig: FinanceLedgerSummaryConfig<API.FinanceBill> =
  {
    groups: receivableGroups,
    projectRow: (row) =>
      projectAmount(
        row.direction,
        row.currency,
        row.totalAmount,
        row.baseCurrency,
        row.baseCurrencyAmount,
      ),
  };
export const invoiceLedgerSummaryConfig: FinanceLedgerSummaryConfig<API.FinanceInvoice> =
  {
    groups: [
      { direction: 'RECEIVABLE', label: '销项发票' },
      { direction: 'PAYABLE', label: '进项发票' },
    ],
    projectRow: (row) =>
      projectAmount(
        row.direction,
        row.currency,
        row.totalAmount,
        row.baseCurrency,
        row.baseCurrencyAmount,
      ),
  };
export const cashflowLedgerSummaryConfig: FinanceLedgerSummaryConfig<API.FinanceCashflow> =
  {
    groups: [
      { direction: 'RECEIVABLE', label: '收款' },
      { direction: 'PAYABLE', label: '付款' },
    ],
    projectRow: (row) =>
      projectAmount(
        row.direction,
        row.currency,
        row.amount,
        row.baseCurrency,
        row.baseAmount,
      ),
  };
export const verificationLedgerSummaryConfig: FinanceLedgerSummaryConfig<API.FinanceVerification> =
  {
    groups: [
      { direction: 'RECEIVABLE', label: '应收核销' },
      { direction: 'PAYABLE', label: '应付核销' },
    ],
    projectRow: (row) =>
      projectAmount(
        row.direction,
        row.currency,
        row.amount,
        row.baseCurrency,
        row.baseAmount,
      ),
  };
export const nettingLedgerSummaryConfig: FinanceLedgerSummaryConfig<API.FinanceNetting> =
  {
    groups: [{ direction: '', label: '抵销金额' }],
    projectRow: (row) =>
      projectAmount(
        '',
        row.currency,
        row.amount,
        row.baseCurrency,
        row.baseCurrencyAmount,
      ),
  };
export const orderFeeSummaryConfig: FinanceLedgerSummaryConfig<API.OrderFee> = {
  groups: receivableGroups,
  profit,
  projectRow: (row) =>
    projectAmount(
      row.direction === OrderFeeDirection.ORDER_FEE_DIRECTION_RECEIVABLE
        ? 'RECEIVABLE'
        : row.direction === OrderFeeDirection.ORDER_FEE_DIRECTION_PAYABLE
          ? 'PAYABLE'
          : undefined,
      row.currency,
      row.totalAmount,
      row.baseCurrency,
      row.baseCurrencyAmount,
    ),
};

/** 正式列表契约必须返回完整匹配汇总，不能退回本页或业务有效金额。 */
export function mapMatchedSummary(
  summary: API.FinanceLedgerMatchedSummary | undefined,
): FinanceLedgerMatchedSummary {
  if (!summary) throw new Error('财务列表缺少筛选汇总');
  return {
    count: Number(summary.count ?? 0),
    amounts: (summary.amounts ?? []).map((item) => ({
      ...projectAmount(
        item.direction ?? '',
        item.currency,
        item.amount,
        item.baseCurrency,
        item.baseAmount,
      ),
      unconvertedCount: Number(item.unconvertedCount ?? 0),
    })),
  };
}

/** 同一业务模块的查询快照提交，附带可渲染的选择身份；未提交输入不调用它。 */
export function useLedgerSelectionScope<TArgs extends unknown[]>(
  commit: (...args: TArgs) => void,
  getFilters: () => unknown,
) {
  const [scope, setScope] = useState(() => JSON.stringify(getFilters()));
  const commitFilters = (...args: TArgs) => {
    commit(...args);
    setScope(JSON.stringify(getFilters()));
  };
  return { selectionScope: scope, commitFilters };
}
