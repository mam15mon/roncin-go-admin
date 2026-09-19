import type { Dayjs } from 'dayjs';
import { financeErrorReasons } from '@/errorReasons.generated';

export type GroupFormValue = {
  statementTitle: string;
  billDate: Dayjs;
  paymentTermsDays?: number;
  note?: string;
  settlementAccountId?: string;
  estimatedInvoiceCurrency?: string;
  estimatedInvoiceRate?: string;
};

export type WorkbenchFormValue = {
  groups: Record<string, GroupFormValue>;
};
export type WorkbenchValidationError = {
  errorFields?: { name?: (string | number)[] }[];
};

export type RequestError = Error & {
  data?: { reason?: string; message?: string };
  response?: { data?: { reason?: string; message?: string } };
};

export type BillCreationMode = 'NORMAL' | 'NETTING';

export function directionText(value?: string) {
  return value === 'RECEIVABLE' ? '应收' : '应付';
}

export function requestReason(error: RequestError) {
  return error.data?.reason || error.response?.data?.reason;
}

export function requestMessage(error: RequestError, fallback: string) {
  const msg =
    error.response?.data?.message ||
    error.data?.message ||
    (error.message && !error.message.toLowerCase().includes('status code')
      ? error.message
      : '');
  if (msg) return msg;
  const reason = requestReason(error);
  if (reason === financeErrorReasons.FINANCE_BILL_FEE_INVALID) {
    return '所选费用必须为已确认状态且尚未进入其他账单';
  }
  if (reason === financeErrorReasons.FINANCE_BILL_PREVIEW_STALE) {
    return '费用已发生变化，请重新预览后再生成账单';
  }
  return fallback;
}

export function makeGroupConfig(
  groupKey: string,
  value?: GroupFormValue,
): API.BillBatchPreviewGroupConfigInput {
  return {
    groupKey,
    billDate: value?.billDate?.format('YYYY-MM-DD'),
    settlementAccountId: value?.settlementAccountId,
    estimatedInvoiceCurrency: value?.estimatedInvoiceCurrency,
    estimatedInvoiceRate: value?.estimatedInvoiceRate,
  };
}

export function isGroupComplete(
  value?: GroupFormValue,
  groupCurrency?: string,
) {
  if (
    !value?.statementTitle?.trim() ||
    !value.billDate ||
    !value.settlementAccountId
  ) {
    return false;
  }
  const estimatedCurrency = value.estimatedInvoiceCurrency?.trim();
  const estimatedRate = value.estimatedInvoiceRate?.trim();
  if (
    estimatedCurrency &&
    groupCurrency &&
    estimatedCurrency !== groupCurrency
  ) {
    if (
      !estimatedRate ||
      Number.isNaN(Number(estimatedRate)) ||
      Number(estimatedRate) <= 0
    ) {
      return false;
    }
  }
  if (
    estimatedCurrency &&
    groupCurrency &&
    estimatedCurrency === groupCurrency &&
    estimatedRate
  ) {
    if (Number(estimatedRate) !== 1) {
      return false;
    }
  }
  return true;
}
