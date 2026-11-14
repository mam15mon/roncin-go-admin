import type { Dayjs } from 'dayjs';
import { type CsvColumn, csvAmount } from '@/utils/csv';
import { formatDate } from '@/utils/format';

export type ReportKind = 'profit' | 'cashflow' | 'aging';
export type ReportSearchValues = {
  keyword?: string;
  dateRange?: [Dayjs | null, Dayjs | null] | null;
  businessType?: string;
  customerId?: string;
  salesId?: string;
  operatorId?: string;
  customerServiceId?: string;
  direction?: string;
  settlementPartyId?: string;
  currency?: string;
  paymentMethod?: string;
  ourAccount?: string;
  bucket?: string;
};
export type ReportQuery = Omit<ReportSearchValues, 'dateRange'> & {
  organizationId: string;
  dateFrom?: string;
  dateTo?: string;
  page?: number;
  pageSize?: number;
};
export type ReportEnvelope<T> = {
  success?: boolean;
  data?: T[];
  total?: string;
  generatedAt?: string;
};
export const directionLabels: Record<string, string> = {
  RECEIVABLE: '应收',
  PAYABLE: '应付',
};
export const businessLabels: Record<string, string> = {
  SE: '海运出口',
  SI: '海运进口',
  AE: '空运出口',
  AI: '空运进口',
  LAND: '陆运',
  RAIL: '铁路',
};
export const agingLabels: Record<string, string> = {
  NOT_DUE: '未到期',
  DAYS_1_30: '1–30 天',
  DAYS_31_60: '31–60 天',
  DAYS_61_90: '61–90 天',
  OVER_90: '90 天以上',
  UNDATED: '未设置到期日',
};

/** 筛选与导出共用同一规范化入口；业务公司由工作区最后显式写入。 */
export function makeReportQuery(
  kind: ReportKind,
  values: ReportSearchValues,
  organizationId: string,
): ReportQuery {
  const keys: (keyof Omit<ReportSearchValues, 'dateRange'>)[] =
    kind === 'profit'
      ? [
          'keyword',
          'businessType',
          'customerId',
          'salesId',
          'operatorId',
          'customerServiceId',
        ]
      : kind === 'cashflow'
        ? [
            'keyword',
            'direction',
            'settlementPartyId',
            'currency',
            'paymentMethod',
            'ourAccount',
          ]
        : ['keyword', 'direction', 'settlementPartyId', 'currency', 'bucket'];
  const query: ReportQuery = { organizationId };
  for (const key of keys) {
    const value = values[key]?.trim();
    if (value) query[key] = key === 'currency' ? value.toUpperCase() : value;
  }
  if (kind !== 'aging') {
    query.dateFrom = values.dateRange?.[0]?.format('YYYY-MM-DD');
    query.dateTo = values.dateRange?.[1]?.format('YYYY-MM-DD');
  }
  return query;
}
function required(value: unknown, field: string): asserts value is string {
  if (typeof value !== 'string' || !value.trim())
    throw new Error(`报表缺少${field}`);
}
function money(value: unknown, field: string) {
  required(value, field);
  if (!/^-?\d+(?:\.\d+)?$/.test(value))
    throw new Error(`报表${field}格式不合法`);
}
export function reportPage<T extends { id?: string }>(
  response: ReportEnvelope<T> | undefined,
  validate: (row: T) => void,
): { data: T[]; total: number; success: true } {
  if (response?.success !== true) throw new Error('报表查询失败，请稍后重试');
  required(response.generatedAt, '生成时间');
  const total = Number(response.total ?? '0');
  if (!Number.isSafeInteger(total) || total < 0)
    throw new Error('报表记录数量不合法');
  const data = response.data ?? [];
  for (const row of data) {
    required(row.id, '行标识');
    validate(row);
  }
  if (data.length > total) throw new Error('报表明细与记录数量不一致');
  return { data, total, success: true };
}
export function validateProfitRow(row: API.ProfitReportRow) {
  required(row.orderId, '订单标识');
  required(row.baseCurrency, '本币');
  for (const key of [
    'receivableAmount',
    'payableAmount',
    'profitAmount',
    'netReceivableAmount',
    'netPayableAmount',
    'netProfitAmount',
  ] as const)
    money(row[key], key);
}
export function validateCashflowRow(row: API.CashflowReportRow) {
  required(row.currency, '原币');
  required(row.baseCurrency, '本币');
  required(row.direction, '方向');
  money(row.amount, '原币金额');
  money(row.baseAmount, '本币金额');
}
export function validateAgingRow(row: API.AgingReportRow) {
  required(row.currency, '原币');
  required(row.baseCurrency, '本币');
  required(row.bucket, '账龄分段');
  if (!agingLabels[row.bucket]) throw new Error('报表账龄分段不合法');
  for (const key of [
    'totalAmount',
    'settledAmount',
    'balance',
    'baseBalance',
  ] as const)
    money(row[key], key);
}
const amountColumn = <T>(
  header: string,
  value: (row: T) => string | undefined,
): CsvColumn<T> => ({
  header,
  kind: 'number',
  value: (row) => csvAmount(value(row)),
});
export const profitExportColumns: CsvColumn<API.ProfitReportRow>[] = [
  { header: '订单号', value: (r) => r.orderNo },
  {
    header: '业务类型',
    value: (r) => businessLabels[r.businessType ?? ''] ?? r.businessType,
  },
  { header: '业务日期', value: (r) => r.businessDate },
  { header: '客户', value: (r) => r.customerName },
  { header: '销售', value: (r) => r.salesName },
  { header: '操作', value: (r) => r.operatorName },
  { header: '客服', value: (r) => r.customerServiceName },
  { header: '本币', value: (r) => r.baseCurrency },
  amountColumn('含税应收', (r) => r.receivableAmount),
  amountColumn('未税应收', (r) => r.netReceivableAmount),
  amountColumn('含税应付', (r) => r.payableAmount),
  amountColumn('未税应付', (r) => r.netPayableAmount),
  amountColumn('含税利润', (r) => r.profitAmount),
  amountColumn('未税利润', (r) => r.netProfitAmount),
];
export const cashflowExportColumns: CsvColumn<API.CashflowReportRow>[] = [
  { header: '流水号', value: (r) => r.flowNo },
  {
    header: '方向',
    value: (r) => (r.direction === 'RECEIVABLE' ? '收款' : '付款'),
  },
  { header: '交易日期', value: (r) => r.transactionDate },
  { header: '结算单位', value: (r) => r.settlementPartyName },
  { header: '币种', value: (r) => r.currency },
  amountColumn('原币金额', (r) => r.amount),
  { header: '本币', value: (r) => r.baseCurrency },
  amountColumn('本币金额', (r) => r.baseAmount),
  { header: '本方账户', value: (r) => r.ourAccount },
  { header: '支付方式', value: (r) => r.paymentMethod },
  { header: '银行参考号', value: (r) => r.bankReferenceNo },
];
export const agingExportColumns: CsvColumn<API.AgingReportRow>[] = [
  { header: '账单号', value: (r) => r.billNo },
  { header: '方向', value: (r) => directionLabels[r.direction ?? ''] },
  { header: '结算单位', value: (r) => r.settlementPartyName },
  { header: '账单日', value: (r) => r.billDate },
  { header: '到期日', value: (r) => r.dueDate },
  { header: '币种', value: (r) => r.currency },
  amountColumn('账单金额', (r) => r.totalAmount),
  amountColumn('已清偿', (r) => r.settledAmount),
  amountColumn('当前余额', (r) => r.balance),
  { header: '本币', value: (r) => r.baseCurrency },
  amountColumn('本币余额', (r) => r.baseBalance),
  { header: '账龄', value: (r) => agingLabels[r.bucket ?? ''] },
];
export function reportFileName(title: string, generatedAt: string | undefined) {
  return `${title}_${formatDate(generatedAt, 'date')}.csv`;
}

function requireTotals<T>(
  items: T[] | undefined,
  total: string | undefined,
  field: string,
): T[] {
  if (Number(total ?? 0) > 0 && !items?.length)
    throw new Error(`报表缺少${field}`);
  return items ?? [];
}
function validateProfitAmount(item: API.ProfitReportAmount) {
  required(item.baseCurrency, '汇总本币');
  for (const key of [
    'receivableAmount',
    'payableAmount',
    'profitAmount',
    'netReceivableAmount',
    'netPayableAmount',
    'netProfitAmount',
  ] as const)
    money(item[key], key);
}
export function validateProfitResponse(response: API.ListProfitReportResponse) {
  requireTotals(response.totals, response.total, '完整筛选合计').forEach(
    validateProfitAmount,
  );
  for (const item of requireTotals(
    response.nativeTotals,
    response.total,
    '原币合计',
  )) {
    required(item.currency, '原币合计币种');
    money(item.amount, '原币含税合计');
    money(item.netAmount, '原币未税合计');
  }
  for (const month of requireTotals(
    response.months,
    response.total,
    '完整月序列',
  )) {
    required(month.month, '月序列月份');
    (month.amounts ?? []).forEach(validateProfitAmount);
  }
}
export function validateCashflowResponse(
  response: API.ListCashflowReportResponse,
) {
  for (const item of [
    ...requireTotals(response.nativeTotals, response.total, '原币合计'),
    ...requireTotals(response.baseTotals, response.total, '本币合计'),
  ]) {
    required(item.currency, '收付合计币种');
    for (const key of [
      'receivableAmount',
      'payableAmount',
      'netAmount',
    ] as const)
      money(item[key], key);
  }
}
export function validateAgingResponse(response: API.ListAgingReportResponse) {
  required(response.asOfDate, '当前业务日');
  for (const item of [
    ...requireTotals(response.nativeTotals, response.total, '原币账龄合计'),
    ...requireTotals(response.baseTotals, response.total, '本币账龄合计'),
  ]) {
    required(item.currency, '账龄合计币种');
    for (const key of [
      'notDue',
      'days130',
      'days3160',
      'days6190',
      'over90',
      'undated',
      'total',
    ] as const)
      money(item[key], key);
  }
}
