import { FinanceVerificationStatus } from '@/enums.generated';
import { type CsvColumn, csvAmount } from '@/utils/csv';
import { formatDate } from '@/utils/format';

export const verificationStatusText = (status?: number) =>
  status === FinanceVerificationStatus.FINANCE_VERIFICATION_STATUS_ACTIVE
    ? '有效'
    : '已反核销';

export const verificationExportColumns: CsvColumn<API.FinanceVerification>[] = [
  { header: '所属公司', value: (row) => row.organizationName },
  { header: '核销编号', value: (row) => row.verificationNo },
  { header: '状态', value: (row) => verificationStatusText(row.status) },
  {
    header: '方向',
    value: (row) =>
      row.direction === 'RECEIVABLE'
        ? '应收核销'
        : row.direction === 'PAYABLE'
          ? '应付核销'
          : '',
  },
  { header: '结算单位', value: (row) => row.settlementPartyName },
  { header: '原币币种', value: (row) => row.currency },
  { header: '核销金额', kind: 'number', value: (row) => csvAmount(row.amount) },
  { header: '本位币币种', value: (row) => row.baseCurrency },
  {
    header: '流水本位币合计',
    kind: 'number',
    value: (row) => csvAmount(row.baseAmount),
  },
  {
    header: '账单本位币合计',
    kind: 'number',
    value: (row) => csvAmount(row.billBaseAmount),
  },
  {
    header: '资金本位币合计',
    kind: 'number',
    value: (row) => csvAmount(row.cashflowBaseAmount),
  },
  {
    header: '汇兑损益',
    kind: 'number',
    value: (row) => csvAmount(row.exchangeGainLoss ?? '0'),
  },
  {
    header: '分配数',
    kind: 'number',
    value: (row) => row.allocations?.length ?? 0,
  },
  { header: '核销日期', value: (row) => row.verificationDate },
  {
    header: '关联明细',
    value: (row) =>
      row.allocations
        ?.map(
          (item) =>
            `${item.cashflowNo ?? ''} → ${item.billNo ?? ''}: ${csvAmount(item.amount)} ${row.currency ?? ''}`,
        )
        .join('；'),
  },
  { header: '备注', value: (row) => row.note },
  {
    header: '创建时间',
    value: (row) => (row.createdAt ? formatDate(row.createdAt) : ''),
  },
  {
    header: '反核销时间',
    value: (row) => (row.reversedAt ? formatDate(row.reversedAt) : ''),
  },
];
