import { FinanceCashflowStatus } from '@/enums.generated';
import { type CsvColumn, csvAmount, csvDecimal } from '@/utils/csv';
import { formatDate } from '@/utils/format';

export const paymentMethodOptions = [
  { label: '银行转账', value: 'BANK_TRANSFER' },
  { label: '支票', value: 'CHECK' },
  { label: '现金', value: 'CASH' },
  { label: '第三方支付', value: 'THIRD_PARTY' },
];

export const cashflowStates: Record<number, { text: string; color: string }> = {
  [FinanceCashflowStatus.FINANCE_CASHFLOW_STATUS_DRAFT]: {
    text: '草稿',
    color: 'default',
  },
  [FinanceCashflowStatus.FINANCE_CASHFLOW_STATUS_CONFIRMED]: {
    text: '已确认',
    color: 'blue',
  },
  [FinanceCashflowStatus.FINANCE_CASHFLOW_STATUS_CANCELLED]: {
    text: '已取消',
    color: 'red',
  },
};

export const cashflowExportColumns: CsvColumn<API.FinanceCashflow>[] = [
  { header: '所属公司', value: (row) => row.organizationName },
  {
    header: '方向',
    value: (row) =>
      row.direction === 'RECEIVABLE'
        ? '收款'
        : row.direction === 'PAYABLE'
          ? '付款'
          : '',
  },
  {
    header: '状态',
    value: (row) =>
      cashflowStates[
        row.status ?? FinanceCashflowStatus.FINANCE_CASHFLOW_STATUS_DRAFT
      ]?.text ?? '',
  },
  { header: '流水编号', value: (row) => row.flowNo },
  { header: '结算单位', value: (row) => row.settlementPartyName },
  { header: '原币币种', value: (row) => row.currency },
  { header: '金额', kind: 'number', value: (row) => csvAmount(row.amount) },
  {
    header: '结算汇率',
    kind: 'number',
    value: (row) => csvDecimal(row.exchangeRate),
  },
  { header: '本位币币种', value: (row) => row.baseCurrency },
  {
    header: '折本币',
    kind: 'number',
    value: (row) => csvAmount(row.baseAmount),
  },
  {
    header: '已核销',
    kind: 'number',
    value: (row) => csvAmount(row.verifiedAmount ?? '0'),
  },
  {
    header: '未核销',
    kind: 'number',
    value: (row) => csvAmount(row.unverifiedAmount ?? '0'),
  },
  { header: '交易日期', value: (row) => row.transactionDate },
  { header: '我方账户', value: (row) => row.ourAccount },
  { header: '对方账户', value: (row) => row.counterpartyAccount },
  {
    header: '结算方式',
    value: (row) =>
      paymentMethodOptions.find((option) => option.value === row.paymentMethod)
        ?.label ?? row.paymentMethod,
  },
  { header: '银行流水号', value: (row) => row.bankReferenceNo },
  { header: '备注', value: (row) => row.note },
  {
    header: '创建时间',
    value: (row) => (row.createdAt ? formatDate(row.createdAt) : ''),
  },
];
