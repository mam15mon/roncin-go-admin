import type { Dayjs } from 'dayjs';
import Decimal from 'decimal.js';
import { financeErrorReasons } from '@/errorReasons.generated';

export type BillInvoiceDraft = {
  invoiceProfileId?: string;
  invoiceType?: string;
  taxInvoiceNo?: string;
  invoiceDate?: Dayjs;
  totalAmount?: string;
  note?: string;
  allocations?: {
    billId?: string;
    amount?: string;
    fees?: { orderFeeId?: string; amount?: string; taxAmount?: string }[];
  }[];
};
export type BillVerificationDraft = {
  verificationDate?: Dayjs;
  note?: string;
  allocations?: { cashflowId?: string; amount?: string }[];
  shortfallEnabled?: boolean;
  shortfall?: {
    amount?: string;
    transactionDate?: Dayjs;
    ourAccount?: string;
    paymentMethod?: string;
    counterpartyAccount?: string;
    bankReferenceNo?: string;
  };
};
export type GroupFormValue = {
  statementTitle: string;
  billDate: Dayjs;
  paymentTermsDays?: number;
  note?: string;
  settlementAccountId?: string;
  estimatedInvoiceCurrency?: string;
  estimatedInvoiceRate?: string;
  invoices?: BillInvoiceDraft[];
  verifications?: BillVerificationDraft[];
};

export type WorkbenchFormValue = {
  groups: Record<string, GroupFormValue>;
};
export type WorkbenchValidationError = {
  errorFields?: { name?: (string | number)[] }[];
};

export type RequestError = Error & {
  data?: {
    reason?: string;
    message?: string;
    metadata?: Record<string, string>;
  };
  response?: {
    data?: {
      reason?: string;
      message?: string;
      metadata?: Record<string, string>;
    };
  };
};

export type BillCreationMode = 'NORMAL' | 'NETTING';

export function directionText(value?: string) {
  return value === 'RECEIVABLE' ? '应收' : '应付';
}

export function requestReason(error: RequestError) {
  return error.data?.reason || error.response?.data?.reason;
}

export function requestMessage(
  error: RequestError,
  fallback: string,
  groups?: Record<string, API.BillBatchPreviewGroup>,
) {
  const msg =
    error.response?.data?.message ||
    error.data?.message ||
    (error.message && !error.message.toLowerCase().includes('status code')
      ? error.message
      : '');
  if (msg) {
    const metadata = error.response?.data?.metadata ?? error.data?.metadata;
    if (
      metadata?.step === 'BILL' ||
      metadata?.step === 'INVOICE' ||
      metadata?.step === 'VERIFICATION'
    ) {
      const index = Number(metadata.item_index ?? 0);
      const label = billBatchItemLabel({
        step: metadata.step,
        itemIndex: Number.isInteger(index) && index > 0 ? index : undefined,
      });
      const group = groups?.[metadata.group_key ?? ''];
      const groupLabel = group
        ? `${group.settlementPartyName ?? '未命名结算单位'} · ${directionText(group.direction)} · ${group.currency ?? ''}：`
        : '';
      return `${groupLabel}${label}：${msg}`;
    }
    return msg;
  }
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
  itemErrors?: API.BillBatchItemError[];
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
      itemErrors: item.error.items,
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

/** 新账单组合只提交明确业务来源；尚未存在的账单身份由服务端内部填充。 */
export function billGroupOperationPayload(
  group: API.BillBatchPreviewGroup,
  values: GroupFormValue,
): Pick<API.CreateBillBatchGroupInput, 'invoices' | 'verifications'> {
  const money = (raw: string | undefined, label: string, allowZero = false) => {
    let value: Decimal;
    try {
      value = new Decimal(raw ?? '');
    } catch {
      throw new Error(`${label}不是有效金额`);
    }
    if (
      !value.isFinite() ||
      value.decimalPlaces() > 8 ||
      (allowZero ? value.lt(0) : value.lte(0))
    )
      throw new Error(
        `${label}必须${allowZero ? '大于等于' : '大于'}0且最多8位小数`,
      );
    return value.toFixed(8);
  };
  const invoices = (values.invoices ?? []).map(
    (record, index): API.BillBatchInvoiceInput => {
      if (
        !record.invoiceProfileId ||
        !record.invoiceType ||
        !record.taxInvoiceNo?.trim() ||
        !record.invoiceDate
      )
        throw new Error(`第${index + 1}张发票资料未填写完整`);
      if (
        record.allocations?.length !== 1 ||
        record.allocations[0].billId !== group.groupKey
      )
        throw new Error(`第${index + 1}张发票必须仅分配到当前账单组`);
      const totalAmount = money(record.totalAmount, '发票总额');
      const allocation = record.allocations[0];
      if (
        !new Decimal(money(allocation.amount, '本账单分配额')).eq(totalAmount)
      )
        throw new Error(`第${index + 1}张发票分配额与总额不符`);
      const fees = (allocation.fees ?? []).map(
        (fee): API.InvoiceRegistrationFeeInput => {
          if (
            !fee.orderFeeId ||
            !(group.fees ?? []).some((source) => source.id === fee.orderFeeId)
          )
            throw new Error('发票费用不属于当前账单组');
          const amount = money(fee.amount, '费用分配金额');
          const taxAmount = money(fee.taxAmount, '费用税额', true);
          if (new Decimal(taxAmount).gt(amount))
            throw new Error('税额不能超过费用分配金额');
          return { orderFeeId: fee.orderFeeId, amount, taxAmount };
        },
      );
      if (
        !fees.length ||
        !fees
          .reduce((sum, fee) => sum.plus(fee.amount), new Decimal(0))
          .eq(totalAmount)
      )
        throw new Error('发票费用分配合计必须等于本次发票金额');
      return {
        invoiceProfileId: record.invoiceProfileId,
        invoiceType: record.invoiceType,
        taxInvoiceNo: record.taxInvoiceNo.trim(),
        invoiceDate: record.invoiceDate.format('YYYY-MM-DD'),
        totalAmount,
        note: record.note?.trim() || undefined,
        fees,
      };
    },
  );
  const verifications = (values.verifications ?? []).map(
    (record, index): API.BillBatchVerificationInput => {
      if (!record.verificationDate)
        throw new Error(`第${index + 1}笔核销日期必填`);
      const allocations = (record.allocations ?? []).map(
        (item): API.BillBatchVerificationAllocationInput => {
          if (!item.cashflowId) throw new Error('请选择既有资金流水');
          return {
            cashflowId: item.cashflowId,
            amount: money(item.amount, '核销分配金额'),
          };
        },
      );
      const payload: API.BillBatchVerificationInput = {
        verificationDate: record.verificationDate.format('YYYY-MM-DD'),
        note: record.note?.trim() || undefined,
        allocations,
      };
      if (record.shortfallEnabled) {
        const cash = record.shortfall;
        if (
          !cash?.transactionDate ||
          !cash.ourAccount?.trim() ||
          !cash.paymentMethod
        )
          throw new Error(`第${index + 1}笔差额收付款资料必填`);
        const amount = money(cash.amount, '差额金额');
        payload.shortfallCashflow = {
          amount,
          transactionDate: cash.transactionDate.format('YYYY-MM-DD'),
          ourAccount: cash.ourAccount.trim(),
          paymentMethod: cash.paymentMethod,
          counterpartyAccount: cash.counterpartyAccount?.trim() || undefined,
          bankReferenceNo: cash.bankReferenceNo?.trim() || undefined,
        };
        allocations.push({ cashflowId: '', amount });
      }
      if (!allocations.length)
        throw new Error(`第${index + 1}笔核销至少填写一笔资金分配`);
      return payload;
    },
  );
  const totalVerified = verifications
    .flatMap((item) => item.allocations ?? [])
    .reduce((sum, item) => sum.plus(item.amount), new Decimal(0));
  if (totalVerified.gt(group.totalAmount ?? '0'))
    throw new Error('本组同次核销合计不能超过账单金额');
  return {
    invoices: invoices.length ? invoices : undefined,
    verifications: verifications.length ? verifications : undefined,
  };
}

export function billBatchItemLabel(item: API.BillBatchItemError) {
  const step =
    item.step === 'INVOICE'
      ? '发票登记'
      : item.step === 'VERIFICATION'
        ? '核销'
        : '建账';
  return item.itemIndex ? `第${item.itemIndex}条${step}` : step;
}
