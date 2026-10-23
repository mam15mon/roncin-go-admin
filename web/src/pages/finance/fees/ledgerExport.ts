import {
  businessTypeMeta,
  normalizeBusinessType,
  statusText,
} from '@/constants/statusMeta';
import { FeeLedgerFinancialProgress } from '@/enums.generated';
import { feeLedgerProgressLabels } from '@/features/finance/fee-progress';
import { type CsvColumn, csvAmount, csvDecimal } from '@/utils/csv';
import { formatDate } from '@/utils/format';

export const feeExportColumns: CsvColumn<API.FeeLedgerItem>[] = [
  {
    header: '标签',
    value: (row) => row.tags?.map((tag) => tag.name).join('；'),
  },
  { header: '所属公司', value: (row) => row.organizationName },
  {
    header: '属性',
    value: (row) =>
      row.direction === 'RECEIVABLE'
        ? '应收'
        : row.direction === 'PAYABLE'
          ? '应付'
          : '',
  },
  { header: '委托单位', value: (row) => row.customerName },
  { header: '结算单位', value: (row) => row.settlementPartyName },
  {
    header: '业务类型',
    value: (row) =>
      statusText(
        businessTypeMeta,
        normalizeBusinessType(row.businessType),
        row.businessType || '',
      ),
  },
  { header: '费用编码', value: (row) => row.feeCode },
  { header: '费用名称', value: (row) => row.feeName },
  {
    header: '费用性质',
    value: (row) =>
      row.economicEffect === 'DECREASE' ? '更正冲回' : '正常费用',
  },
  { header: '计费单位', value: (row) => row.billingUnit },
  { header: '数量', kind: 'number', value: (row) => csvDecimal(row.quantity) },
  { header: '单价', kind: 'number', value: (row) => csvDecimal(row.unitPrice) },
  { header: '原币币种', value: (row) => row.currency },
  {
    header: '金额',
    kind: 'number',
    value: (row) => csvAmount(row.totalAmount),
  },
  {
    header: '税率(%)',
    kind: 'number',
    value: (row) => csvDecimal(row.taxRate),
  },
  { header: '税金', kind: 'number', value: (row) => csvAmount(row.taxAmount) },
  {
    header: '不含税总价',
    kind: 'number',
    value: (row) => csvAmount(row.netAmount),
  },
  {
    header: '汇率',
    kind: 'number',
    value: (row) => csvDecimal(row.exchangeRate),
  },
  { header: '本位币币种', value: (row) => row.baseCurrency },
  {
    header: '折本币总价',
    kind: 'number',
    value: (row) => csvAmount(row.baseCurrencyAmount),
  },
  {
    header: '订单审核',
    value: (row) => (row.orderLocked ? '已审核锁定' : '未审核'),
  },
  {
    header: '待审补录',
    value: (row) => (row.hasPendingFeeSupplement ? '是' : '否'),
  },
  {
    header: '待审更正',
    value: (row) => (row.hasPendingFeeCorrection ? '是' : '否'),
  },
  {
    header: '建账限制',
    value: (row) => (row.canBill ? '可建账' : row.blockedReasons?.join('；')),
  },
  {
    header: '财务进度',
    value: (row) =>
      feeLedgerProgressLabels[
        row.financialProgress ??
          FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_UNBILLED
      ]?.text ?? '',
  },
  { header: '账单编号', value: (row) => row.billNo },
  { header: '订单编号', value: (row) => row.orderNo },
  {
    header: '费用日期',
    value: (row) =>
      row.expenseDate ? formatDate(row.expenseDate, 'date') : '',
  },
  { header: '备注', value: (row) => row.note },
  {
    header: '创建时间',
    value: (row) => (row.createdAt ? formatDate(row.createdAt) : ''),
  },
  {
    header: '更新时间',
    value: (row) => (row.updatedAt ? formatDate(row.updatedAt) : ''),
  },
];
