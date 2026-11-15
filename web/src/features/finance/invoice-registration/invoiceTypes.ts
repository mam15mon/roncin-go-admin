/** 发票票种：仅用于登记记录与查询，不对接税务；与服务端 biz.FinanceInvoiceType 一一对应。 */
export const INVOICE_TYPE_OPTIONS = [
  { value: 'DIGITAL_NORMAL', label: '数电普票' },
  { value: 'DIGITAL_SPECIAL', label: '数电专票' },
  { value: 'PAPER_NORMAL', label: '纸质普票' },
  { value: 'PAPER_SPECIAL', label: '纸质专票' },
  { value: 'ELECTRONIC_NORMAL', label: '电子普票' },
  { value: 'ELECTRONIC_SPECIAL', label: '电子专票' },
] as const;

export const DEFAULT_INVOICE_TYPE = 'DIGITAL_NORMAL';

const labels = new Map<string, string>(
  INVOICE_TYPE_OPTIONS.map((item) => [item.value, item.label]),
);

/** 票种中文名；未知值原样展示，空值展示 `-`。 */
export function invoiceTypeLabel(value?: string) {
  if (!value) return '-';
  return labels.get(value) ?? value;
}

export function isInvoiceType(value?: string) {
  return Boolean(value && labels.has(value));
}

/** 专票类要求抬头登记地址、电话、开户行与账号齐全。 */
export function isSpecialInvoiceType(value?: string) {
  return Boolean(value?.endsWith('_SPECIAL') && labels.has(value));
}
