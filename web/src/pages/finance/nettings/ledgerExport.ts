import { FinanceNettingStatus } from '@/enums.generated';
import { type CsvColumn, csvAmount } from '@/utils/csv';
import { formatDate } from '@/utils/format';

export function nettingStatusText(status?: number): string {
  switch (status) {
    case FinanceNettingStatus.FINANCE_NETTING_STATUS_DRAFT:
      return '草稿';
    case FinanceNettingStatus.FINANCE_NETTING_STATUS_CONFIRMED:
      return '已确认';
    case FinanceNettingStatus.FINANCE_NETTING_STATUS_CANCELLED:
      return '已取消';
    case FinanceNettingStatus.FINANCE_NETTING_STATUS_REVERSED:
      return '已反转';
    default:
      return '未知';
  }
}

export const nettingExportColumns: CsvColumn<API.FinanceNetting>[] = [
  { header: '所属公司', value: (row) => row.organizationName },
  { header: '对冲单号', value: (row) => row.nettingNo },
  { header: '状态', value: (row) => nettingStatusText(row.status) },
  { header: '结算单位', value: (row) => row.settlementPartyName },
  { header: '原币币种', value: (row) => row.currency },
  { header: '抵销金额', kind: 'number', value: (row) => csvAmount(row.amount) },
  { header: '本位币币种', value: (row) => row.baseCurrency },
  {
    header: '应收本币抵销额',
    kind: 'number',
    value: (row) => csvAmount(row.baseCurrencyAmount),
  },
  {
    header: '应付本币抵销额',
    kind: 'number',
    value: (row) => csvAmount(row.payableBaseAmount),
  },
  {
    header: '对冲汇差',
    kind: 'number',
    value: (row) => csvAmount(row.exchangeGainLoss ?? '0'),
  },
  {
    header: '分摊数',
    kind: 'number',
    value: (row) => row.allocations?.length ?? 0,
  },
  { header: '关联批次', value: (row) => row.batchNo },
  {
    header: '关联账单',
    value: (row) => row.allocations?.map((item) => item.billNo).join('；'),
  },
  { header: '备注', value: (row) => row.note },
  {
    header: '创建时间',
    value: (row) => (row.createdAt ? formatDate(row.createdAt) : ''),
  },
  {
    header: '确认时间',
    value: (row) => (row.confirmedAt ? formatDate(row.confirmedAt) : ''),
  },
  {
    header: '取消时间',
    value: (row) => (row.cancelledAt ? formatDate(row.cancelledAt) : ''),
  },
  {
    header: '反转时间',
    value: (row) => (row.reversedAt ? formatDate(row.reversedAt) : ''),
  },
];
