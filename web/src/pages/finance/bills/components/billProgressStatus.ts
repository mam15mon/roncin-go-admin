import type { StatusMeta } from '@/constants/statusMeta';
import {
  FinanceBillInvoiceProgress,
  FinanceBillSettlementProgress,
} from '@/enums.generated';
import { billInvoiceProgressMeta } from '@/features/finance/bill-status';

/** 主状态按尚未完成的开票、核销待办显示，不限制业务操作先后。 */
export function getBillProgressStatus(
  bill: API.FinanceBill,
): StatusMeta | undefined {
  if (bill.completed === true) return { text: '已完成', color: 'success' };

  if (
    bill.invoiceProgress ===
      FinanceBillInvoiceProgress.FINANCE_BILL_INVOICE_PROGRESS_NONE ||
    bill.invoiceProgress ===
      FinanceBillInvoiceProgress.FINANCE_BILL_INVOICE_PROGRESS_PARTIAL
  ) {
    return billInvoiceProgressMeta[bill.invoiceProgress];
  }

  if (
    bill.invoiceProgress !==
      FinanceBillInvoiceProgress.FINANCE_BILL_INVOICE_PROGRESS_FULL &&
    bill.invoiceProgress !==
      FinanceBillInvoiceProgress.FINANCE_BILL_INVOICE_PROGRESS_NOT_REQUIRED
  ) {
    return undefined;
  }
  if (
    bill.settlementProgress ===
    FinanceBillSettlementProgress.FINANCE_BILL_SETTLEMENT_PROGRESS_UNSETTLED
  ) {
    return { text: '未核销', color: 'default' };
  }
  if (
    bill.settlementProgress ===
    FinanceBillSettlementProgress.FINANCE_BILL_SETTLEMENT_PROGRESS_PARTIAL
  ) {
    // 纯对冲不冒充核销；剩余余额仍待核销，结清与完成以服务端投影为准。
    return Number(bill.verifiedAmount ?? '0') !== 0
      ? { text: '部分核销', color: 'orange' }
      : { text: '未核销', color: 'default' };
  }
  return undefined;
}
