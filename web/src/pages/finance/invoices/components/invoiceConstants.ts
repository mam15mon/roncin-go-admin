import { FinanceInvoiceStatus } from '@/enums.generated';

export const invoiceStates: Record<
  number,
  { text: string; color: string }
> = {
  [FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_DRAFT]: { text: '草稿', color: 'gold' },
  [FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_ISSUED]: { text: '已开具', color: 'green' },
  [FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_CANCELLED]: { text: '已作废', color: 'default' },
  [FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_RED_FLUSHED]: { text: '已红冲', color: 'error' },
};

// 销项（应收方向）用「开具」语义，进项（应付方向）用「收票」语义；
// 同一后端状态在不同方向下不得呈现方向相反的操作文案。
export const isReceivableInvoice = (direction?: string): boolean =>
  direction === 'RECEIVABLE';

export const invoiceIssueActionText = (direction?: string): string =>
  isReceivableInvoice(direction) ? '确认开具' : '确认收票';

export const invoiceIssueVerb = (direction?: string): string =>
  isReceivableInvoice(direction) ? '开具' : '收票';

export const invoiceIssueDateLabel = (direction?: string): string =>
  isReceivableInvoice(direction) ? '开票日期' : '收票日期';

export const invoiceIssuedStateText = (direction?: string): string =>
  isReceivableInvoice(direction) ? '已开具' : '已收票';

export const invoiceRecordNoun = (direction?: string): string =>
  isReceivableInvoice(direction) ? '开票记录' : '收票记录';

export const invoiceStateText = (
  status?: number,
  direction?: string,
): string => {
  if (status === FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_ISSUED) {
    return invoiceIssuedStateText(direction);
  }
  const state =
    invoiceStates[status ?? FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_DRAFT];
  return state?.text ?? '';
};
