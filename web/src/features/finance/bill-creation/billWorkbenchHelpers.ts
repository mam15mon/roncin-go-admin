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
    return '所选费用必须为未建账状态且尚未进入其他账单';
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

/** 单组建账结果：成功时有 bill，失败时有 errorMessage。 */
export type BillBatchGroupOutcome = {
  groupKey: string;
  bill?: API.FinanceBill;
  errorReason?: string;
  errorMessage?: string;
};

export type BillBatchOutcome = {
  batch: API.FinanceBillBatch;
  groups: BillBatchGroupOutcome[];
  succeededCount: number;
  failedCount: number;
};

/**
 * 把批量建账响应收敛为逐组结果。HTTP 200 不代表全部成功，必须逐组读取 results：
 * ProtoJSON 省略 false 的 success，省略即视为失败；成功组缺账单或请求组缺结果属于契约错误，
 * 显式抛出而不是当作成功或空结果。
 */
export function normalizeBillBatchResponse(
  response: API.CreateBillBatchResponse,
  requestedGroupKeys: string[],
): BillBatchOutcome {
  if (!response.data?.id) throw new Error('服务端未返回建单批次');
  const byKey = new Map<string, API.BillBatchGroupResult>();
  for (const item of response.results ?? []) {
    if (!item.groupKey) throw new Error('服务端返回的建单结果缺少分组标识');
    if (byKey.has(item.groupKey))
      throw new Error('服务端返回重复的建单分组结果');
    byKey.set(item.groupKey, item);
  }
  if (byKey.size !== requestedGroupKeys.length) {
    throw new Error('服务端返回的建单分组与本次请求不一致');
  }
  const groups = requestedGroupKeys.map((groupKey): BillBatchGroupOutcome => {
    const item = byKey.get(groupKey);
    if (!item) throw new Error('服务端未返回完整的逐组建单结果');
    if (item.success === true) {
      if (!item.bill?.id) throw new Error('服务端未返回成功分组的账单');
      return { groupKey, bill: item.bill };
    }
    if (!item.error?.reason || !item.error.message) {
      throw new Error('服务端未返回失败分组的原因');
    }
    const reason = item.error.reason;
    return {
      groupKey,
      errorReason: reason,
      errorMessage:
        reason === financeErrorReasons.FINANCE_BILL_PREVIEW_STALE
          ? '费用已发生变化，请返回调整失败组后重新预览'
          : item.error.message,
    };
  });
  const succeededCount = groups.filter((group) => group.bill).length;
  return {
    batch: response.data,
    groups,
    succeededCount,
    failedCount: groups.length - succeededCount,
  };
}
