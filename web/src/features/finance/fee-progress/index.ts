import { FeeLedgerFinancialProgress } from '@/enums.generated';

/**
 * 费用台账财务进度的展示文案映射。
 *
 * 口径：该进度是**整张关联账单**的开票与核销综合状态（活动账单行、非取消
 * 账单、有效开票、有效核销与已确认对冲共同投影，见服务端
 * `internal/data/settlement.go`），不是单条费用的核销金额或对账确认。
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
      text: '未核销未开票',
      color: 'orange',
    },
  [FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_INVOICED_UNVERIFIED]:
    {
      text: '已开票未核销',
      color: 'blue',
    },
  [FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_INVOICED_PARTIALLY_VERIFIED]:
    {
      text: '已开票部分核销',
      color: 'cyan',
    },
  [FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_PARTIALLY_VERIFIED_UNINVOICED]:
    {
      text: '部分核销未开票',
      color: 'geekblue',
    },
  [FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_VERIFIED_UNINVOICED]:
    {
      text: '已核销未开票',
      color: 'purple',
    },
  [FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_COMPLETED]: {
    text: '已完成',
    color: 'green',
  },
};
