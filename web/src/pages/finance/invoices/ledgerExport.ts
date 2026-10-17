import { type CsvColumn, csvAmount, csvDecimal } from '@/utils/csv';
import { formatDate } from '@/utils/format';
import { invoiceStateText } from './components/invoiceConstants';

export const invoiceExportColumns: CsvColumn<API.FinanceInvoice>[] = [
  { header: '所属公司', value: (row) => row.organizationName },
  { header: '记录编号', value: (row) => row.recordNo },
  {
    header: '方向',
    value: (row) =>
      row.direction === 'RECEIVABLE'
        ? '销项'
        : row.direction === 'PAYABLE'
          ? '进项'
          : '',
  },
  {
    header: '状态',
    value: (row) => invoiceStateText(row.status, row.direction),
  },
  { header: '结算单位', value: (row) => row.settlementPartyName },
  {
    header: '发票类型',
    value: (row) =>
      row.invoiceType === 'SPECIAL'
        ? '专用发票'
        : row.invoiceType
          ? '普通发票'
          : '',
  },
  { header: '原币币种', value: (row) => row.currency },
  {
    header: '金额',
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
    header: '开票汇率',
    kind: 'number',
    value: (row) => csvDecimal(row.exchangeRate),
  },
  { header: '本位币币种', value: (row) => row.baseCurrency },
  {
    header: '折本币金额',
    kind: 'number',
    value: (row) => csvAmount(row.baseCurrencyAmount),
  },
  { header: '税务发票号', value: (row) => row.taxInvoiceNo },
  { header: '开票日期', value: (row) => row.invoiceDate },
  {
    header: '关联账单数',
    kind: 'number',
    value: (row) => csvDecimal(row.billCount),
  },
  {
    header: '关联账单',
    value: (row) => row.billLinks?.map((bill) => bill.billNo).join('；'),
  },
  { header: '开票抬头', value: (row) => row.invoiceTitle },
  { header: '红字发票号', value: (row) => row.redInvoiceNo },
  { header: '红冲日期', value: (row) => row.redInvoiceDate },
  { header: '备注', value: (row) => row.note },
  {
    header: '创建时间',
    value: (row) => (row.createdAt ? formatDate(row.createdAt) : ''),
  },
];
