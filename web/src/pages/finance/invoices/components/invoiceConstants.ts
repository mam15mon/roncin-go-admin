import { FinanceInvoiceStatus } from '@/enums.generated';

export const invoiceStates: Record<number, { text: string; color: string }> = {
  [FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_DRAFT]: {
    text: '草稿',
    color: 'default',
  },
  [FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_ISSUED]: {
    text: '已开具',
    color: 'green',
  },
  [FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_CANCELLED]: {
    text: '已作废',
    color: 'red',
  },
  [FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_RED_FLUSHED]: {
    text: '已红冲',
    color: 'volcano',
  },
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

// 草稿「取消」与已开票「作废」是两个语义不同的终态操作，确认文案不得混用。
export const invoiceDraftCancelTitle = (
  direction: string | undefined,
  organizationName?: string,
): string =>
  `取消 ${organizationName || '所属公司未标识'} 的${invoiceRecordNoun(direction)}草稿并释放关联账单？`;

export const invoiceVoidTitle = (
  direction: string | undefined,
  organizationName?: string,
): string =>
  `作废 ${organizationName || '所属公司未标识'} 的已${invoiceIssueVerb(direction)}发票并释放关联账单？`;

export const invoiceDraftCancelDescription =
  '取消草稿并释放关联账单，取消后不可恢复。';

// 系统不判断税期：作废只允许在已满足线下税务条件时由操作者确认执行。
export const invoiceVoidTaxConditionNotice =
  '仅当已于线下税控/开票系统完成作废或满足当期作废条件时才能执行；系统不判断税期，不会代替线下税务判断。作废后关联账单将被释放且不可恢复。';

export const invoiceCancelSuccessText = (direction?: string): string =>
  `${invoiceRecordNoun(direction)}草稿已取消，关联账单已释放`;

export const invoiceVoidSuccessText = (direction?: string): string =>
  `${invoiceRecordNoun(direction)}已作废，关联账单已释放`;
