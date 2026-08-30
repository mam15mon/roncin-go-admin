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
