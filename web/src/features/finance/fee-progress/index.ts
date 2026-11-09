import { FeeLedgerFinancialProgress } from '@/enums.generated';

/**
 * 费用台账财务进度的展示文案映射。
 *
 * 口径：该进度是**整张关联账单**的实际开票与付款状态，付款事实来自有效
 * 收付款核销；没有收付款核销时仍为未付款，有实际收付且剩余余额结清后为
 * 已付款。对冲金额单独展示，不能将纯对冲称为付款，见服务端
 * `internal/data/settlement.go`；该状态不是单条费用的金额或对账确认。
 * 财务费用台账与订单费用页的关联账单列共用本映射，禁止两套文案。
 */
export interface FeeLedgerProgressLabel {
  text: string;
  color: string;
}

export const feeLedgerProgressLabels: Record<number, FeeLedgerProgressLabel> = {
  [FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_UNBILLED]: {
    text: '账单未建立',
    color: 'gold',
  },
  [FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_UNVERIFIED_UNINVOICED]:
    {
      text: '未开票未付款',
      color: 'orange',
    },
  [FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_INVOICED_UNVERIFIED]:
    {
      text: '已开票未付款',
      color: 'blue',
    },
  [FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_INVOICED_PARTIALLY_VERIFIED]:
    {
      text: '已开票部分付款',
      color: 'cyan',
    },
  [FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_PARTIALLY_VERIFIED_UNINVOICED]:
    {
      text: '未开票部分付款',
      color: 'geekblue',
    },
  [FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_VERIFIED_UNINVOICED]:
    {
      text: '未开票已付款',
      color: 'purple',
    },
  [FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_COMPLETED]: {
    text: '已开票已付款',
    color: 'green',
  },
};
