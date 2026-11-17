/** 账单页编辑表单的表单值类型；账单状态映射已上提至 features/finance/bill-status。 */
export type BillFormValues = {
  statementTitle: string;
  paymentTermsDays?: number;
  note?: string;
  settlementAccountId: string;
  estimatedInvoiceCurrency?: string;
  estimatedInvoiceRate?: string;
};
