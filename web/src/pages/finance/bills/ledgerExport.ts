import { FinanceBillStatus } from '@/enums.generated';
import { billStatusMeta } from '@/features/finance/bill-status';
import { type CsvColumn, csvAmount, csvDecimal } from '@/utils/csv';
import { formatDate } from '@/utils/format';

export const billExportColumns: CsvColumn<API.FinanceBill>[] = [
  {
    header: '标签',
    value: (row) => row.tags?.map((tag) => tag.name).join('；'),
  },
  {
    header: '方向',
    value: (row) =>
      row.direction === 'RECEIVABLE'
        ? '应收'
        : row.direction === 'PAYABLE'
          ? '应付'
          : '',
  },
  {
    header: '账单用途',
    value: (row) =>
      row.purpose === 'CORRECTION_REVERSAL' ? '更正冲回' : '正常费用',
  },
  {
    header: '状态',
    value: (row) =>
      billStatusMeta[row.status ?? FinanceBillStatus.FINANCE_BILL_STATUS_DRAFT]
        ?.text ?? '',
  },
  { header: '账单编号', value: (row) => row.billNo },
  { header: '所属公司', value: (row) => row.organizationName },
  { header: '建单批次', value: (row) => row.batchNo },
  { header: '结算单位', value: (row) => row.settlementPartyName },
  { header: '对账抬头', value: (row) => row.statementTitle },
  { header: '原币币种', value: (row) => row.currency },
  {
    header: '账单金额',
    kind: 'number',
    value: (row) => csvAmount(row.totalAmount),
  },
  { header: '税额', kind: 'number', value: (row) => csvAmount(row.taxAmount) },
  {
    header: '不含税金额',
    kind: 'number',
    value: (row) => csvAmount(row.netAmount),
  },
  {
    header: '账单汇率',
    kind: 'number',
    value: (row) => csvDecimal(row.exchangeRate),
  },
  {
    header: '汇率来源',
    value: (row) =>
      row.exchangeRateSource === 'MANUAL'
        ? '手工'
        : row.exchangeRateSource === 'BASE_CURRENCY'
          ? '本币'
          : row.exchangeRateSource
            ? '系统'
            : '',
  },
  { header: '本位币币种', value: (row) => row.baseCurrency },
  {
    header: '折本币金额',
    kind: 'number',
    value: (row) => csvAmount(row.baseCurrencyAmount),
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
  {
    header: '费用数',
    kind: 'number',
    value: (row) => csvDecimal(row.feeCount),
  },
  { header: '账单日期', value: (row) => row.billDate },
  { header: '到期日', value: (row) => row.dueDate },
  {
    header: '逾期天数',
    kind: 'number',
    value: (row) => csvDecimal(row.overdueDays),
  },
  { header: '备注', value: (row) => row.note },
  {
    header: '创建时间',
    value: (row) => (row.createdAt ? formatDate(row.createdAt) : ''),
  },
];
