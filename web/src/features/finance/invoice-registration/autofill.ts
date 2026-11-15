import dayjs from 'dayjs';
import Decimal from 'decimal.js';
import { DEFAULT_INVOICE_TYPE } from './invoiceTypes';
import type {
  InvoiceRegistrationAllocationDraft,
  InvoiceRegistrationBillSource,
  InvoiceRegistrationDraft,
  InvoiceRegistrationFeeDraft,
  InvoiceRegistrationFeeSource,
} from './types';

/**
 * 登记表单的辅助带入：只改写来源为系统带入（未标记 manual）的字段，
 * 用户手改值保持原样；不补尾差、不补齐余额，提交仍走三层守恒与服务端校验。
 */
const AMOUNT = /^(?:0|[1-9]\d{0,19})(?:\.\d{1,8})?$/;
const ZERO = new Decimal(0);

export function parseAmount(value?: string): Decimal | undefined {
  const text = value?.trim();
  return text && AMOUNT.test(text) ? new Decimal(text) : undefined;
}

function decimalOf(value?: string): Decimal | undefined {
  if (value === undefined || value === null || value.trim() === '')
    return undefined;
  try {
    return new Decimal(value);
  } catch {
    return undefined;
  }
}

const plain = (value: Decimal) => value.toFixed();

/** 含税口径建议税额：G×r/(100+r)，四舍五入两位；税率未设置不建议。 */
export function suggestTax(gross?: string, rate?: string): string | undefined {
  const amount = parseAmount(gross);
  const percent = decimalOf(rate);
  if (!amount || !percent || percent.isNeg()) return undefined;
  return amount
    .mul(percent)
    .div(percent.plus(100))
    .toDecimalPlaces(2, Decimal.ROUND_HALF_UP)
    .toFixed(2);
}

/** 来源费用自身的可登记余额（服务端实时口径）。 */
export function feeBalance(fee: InvoiceRegistrationFeeSource): Decimal {
  const value = decimalOf(fee.invoiceableAmount ?? fee.totalAmount);
  return value?.gt(0) ? value : ZERO;
}

/** 扣除本会话其他发票对同一费用的占用后，当前发票还可分配的余额。 */
export function availableFor(
  fee: InvoiceRegistrationFeeSource,
  billId: string,
  records: InvoiceRegistrationDraft[] | undefined,
  selfIndex: number,
): Decimal {
  let used = ZERO;
  (records ?? []).forEach((record, index) => {
    if (index === selfIndex) return;
    for (const allocation of record?.allocations ?? []) {
      if (allocation?.billId !== billId) continue;
      for (const item of allocation.fees ?? []) {
        if (item?.orderFeeId !== fee.orderFeeId) continue;
        used = used.plus(parseAmount(item.amount) ?? ZERO);
      }
    }
  });
  const rest = feeBalance(fee).minus(used);
  return rest.gt(0) ? rest : ZERO;
}

export function registrableFees(bill: InvoiceRegistrationBillSource) {
  return bill.fees.filter((fee) => feeBalance(fee).gt(0));
}

function findFee(
  bills: InvoiceRegistrationBillSource[],
  billId?: string,
  orderFeeId?: string,
) {
  return bills
    .find((bill) => bill.id === billId)
    ?.fees.find((fee) => fee.orderFeeId === orderFeeId);
}

function sumFees(allocation: InvoiceRegistrationAllocationDraft) {
  return (allocation.fees ?? []).reduce(
    (sum, fee) => sum.plus(parseAmount(fee?.amount) ?? ZERO),
    ZERO,
  );
}

function sumAllocations(record: InvoiceRegistrationDraft) {
  return (record.allocations ?? []).reduce(
    (sum, allocation) => sum.plus(parseAmount(allocation?.amount) ?? ZERO),
    ZERO,
  );
}

function feeRow(
  fee: InvoiceRegistrationFeeSource,
  amount?: Decimal,
): InvoiceRegistrationFeeDraft {
  const text = amount ? plain(amount) : '';
  return {
    orderFeeId: fee.orderFeeId,
    amount: text,
    amountSource: 'auto',
    taxAmount: suggestTax(text, fee.sourceTaxRate) ?? '',
    taxSource: 'auto',
  };
}

/** 税额未手改时跟随本行含税额与来源税率；未设置税率则清空待财务填写。 */
function followTax(
  fee: InvoiceRegistrationFeeDraft,
  source?: InvoiceRegistrationFeeSource,
): InvoiceRegistrationFeeDraft {
  if (fee.taxSource === 'manual') return fee;
  return {
    ...fee,
    taxSource: 'auto',
    taxAmount: suggestTax(fee.amount, source?.sourceTaxRate) ?? '',
  };
}

/** 所选账单扣除本会话其他发票占用后的可登记余额合计。 */
export function remainingBalance(
  bills: InvoiceRegistrationBillSource[],
  records: InvoiceRegistrationDraft[] | undefined,
  selfIndex: number,
): Decimal {
  return bills.reduce(
    (sum, bill) =>
      bill.fees.reduce(
        (inner, fee) =>
          inner.plus(availableFor(fee, bill.id, records, selfIndex)),
        sum,
      ),
    ZERO,
  );
}

/**
 * 新发票：总额默认带出所选账单尚未分配的余额合计并按余额自动分配；
 * 无余额时总额留空，单账单仍预选账单（仅一条可登记费用时同时选好费用）。
 */
export function newInvoiceRecord(
  bills: InvoiceRegistrationBillSource[],
  records?: InvoiceRegistrationDraft[],
  selfIndex = records?.length ?? 0,
): InvoiceRegistrationDraft {
  const record: InvoiceRegistrationDraft = {
    invoiceType: DEFAULT_INVOICE_TYPE,
    invoiceDate: dayjs(),
    allocations: [],
  };
  const total = remainingBalance(bills, records, selfIndex);
  if (total.gt(0))
    return autoAllocate(
      { ...record, totalAmount: plain(total), totalSource: 'auto' },
      bills,
      records,
      selfIndex,
    );
  if (bills.length !== 1) return record;
  const fees = registrableFees(bills[0]);
  record.allocations = [
    {
      billId: bills[0].id,
      amount: '',
      amountSource: 'auto',
      fees: fees.length === 1 ? [feeRow(fees[0])] : [],
    },
  ];
  return record;
}

/** 总额与全部分配均为系统带入、且不是原票修订项时，才允许随账单选择重建。 */
export function isUntouchedRecord(record?: InvoiceRegistrationDraft) {
  return Boolean(
    record &&
      !record.sourceInvoiceId &&
      record.totalSource !== 'manual' &&
      !hasManualAllocation(record),
  );
}

/** 账单选择变化：重建未手改的发票（保留抬头、号码、日期、类型、备注），手改过的保持不变。 */
export function rebuildUntouchedRecords(
  records: InvoiceRegistrationDraft[],
  bills: InvoiceRegistrationBillSource[],
): InvoiceRegistrationDraft[] {
  // 先释放所有待重建发票的旧占用，再按顺序逐张分配，避免后序旧值挤占前序余额。
  const next = records.map((record) =>
    isUntouchedRecord(record) ? { ...record, allocations: [] } : record,
  );
  next.forEach((record, index) => {
    if (!isUntouchedRecord(records[index])) return;
    const fresh = newInvoiceRecord(bills, next, index);
    next[index] = {
      ...record,
      totalAmount: fresh.totalAmount,
      totalSource: fresh.totalSource,
      allocations: fresh.allocations,
    };
  });
  return next;
}

/** 单账单单费用时，未手改的账单分配与费用金额跟随发票总额。 */
export function applyTotalChange(
  record: InvoiceRegistrationDraft,
  bills: InvoiceRegistrationBillSource[],
): InvoiceRegistrationDraft {
  const allocations = record.allocations ?? [];
  if (allocations.length !== 1) return record;
  let allocation = allocations[0];
  if (allocation.amountSource !== 'manual')
    allocation = {
      ...allocation,
      amountSource: 'auto',
      amount: record.totalAmount ?? '',
    };
  return {
    ...record,
    allocations: [followAllocationDown(allocation, bills)],
  };
}

function followAllocationDown(
  allocation: InvoiceRegistrationAllocationDraft,
  bills: InvoiceRegistrationBillSource[],
): InvoiceRegistrationAllocationDraft {
  const fees = allocation.fees ?? [];
  if (fees.length !== 1) return allocation;
  let fee = fees[0];
  if (fee.amountSource !== 'manual')
    fee = { ...fee, amountSource: 'auto', amount: allocation.amount ?? '' };
  return {
    ...allocation,
    fees: [followTax(fee, findFee(bills, allocation.billId, fee.orderFeeId))],
  };
}

/** 用户改了账单分配：单费用时未手改的费用金额与税额跟随。 */
export function applyAllocationChange(
  record: InvoiceRegistrationDraft,
  allocationIndex: number,
  bills: InvoiceRegistrationBillSource[],
): InvoiceRegistrationDraft {
  const allocations = [...(record.allocations ?? [])];
  const allocation = allocations[allocationIndex];
  if (!allocation) return record;
  allocations[allocationIndex] = followAllocationDown(allocation, bills);
  return { ...record, allocations };
}

/** 用户改了费用含税额：税额跟随；未手改的账单分配取费用合计。 */
export function applyFeeChange(
  record: InvoiceRegistrationDraft,
  allocationIndex: number,
  feeIndex: number,
  bills: InvoiceRegistrationBillSource[],
): InvoiceRegistrationDraft {
  const allocations = [...(record.allocations ?? [])];
  const allocation = allocations[allocationIndex];
  const fee = allocation?.fees?.[feeIndex];
  if (!allocation || !fee) return record;
  const fees = [...allocation.fees];
  fees[feeIndex] = followTax(
    fee,
    findFee(bills, allocation.billId, fee.orderFeeId),
  );
  allocations[allocationIndex] = { ...allocation, fees };
  return syncAllocationTotal({ ...record, allocations }, allocationIndex);
}

/** 未手改的账单分配取其费用含税额合计；无费用时清空。 */
export function syncAllocationTotal(
  record: InvoiceRegistrationDraft,
  allocationIndex: number,
): InvoiceRegistrationDraft {
  const allocation = record.allocations?.[allocationIndex];
  if (!allocation || allocation.amountSource === 'manual') return record;
  const allocations = [...record.allocations];
  const sum = sumFees(allocation);
  allocations[allocationIndex] = {
    ...allocation,
    amountSource: 'auto',
    amount: allocation.fees?.length && sum.gt(0) ? plain(sum) : '',
  };
  return { ...record, allocations };
}

/**
 * 用户选择了来源费用：未手改的含税额带入发票尚未分配的金额（不超可用余额），
 * 税额按新来源税率重算。
 */
export function applyFeeSelection(
  record: InvoiceRegistrationDraft,
  allocationIndex: number,
  feeIndex: number,
  bills: InvoiceRegistrationBillSource[],
  records: InvoiceRegistrationDraft[] | undefined,
  recordIndex: number,
): InvoiceRegistrationDraft {
  const allocation = record.allocations?.[allocationIndex];
  const fee = allocation?.fees?.[feeIndex];
  const source = findFee(bills, allocation?.billId, fee?.orderFeeId);
  if (!allocation || !fee || !source) return record;
  let next = fee;
  if (fee.amountSource !== 'manual') {
    const others = (record.allocations ?? []).reduce(
      (sum, item, index) =>
        sum.plus(
          (item.fees ?? []).reduce(
            (inner, row, rowIndex) =>
              index === allocationIndex && rowIndex === feeIndex
                ? inner
                : inner.plus(parseAmount(row?.amount) ?? ZERO),
            ZERO,
          ),
        ),
      ZERO,
    );
    const gap = (parseAmount(record.totalAmount) ?? ZERO).minus(others);
    const available = availableFor(
      source,
      allocation.billId,
      records,
      recordIndex,
    );
    const amount = Decimal.min(gap, available);
    next = {
      ...fee,
      amountSource: 'auto',
      amount: amount.gt(0) ? plain(amount) : '',
    };
  }
  const fees = [...allocation.fees];
  fees[feeIndex] = { ...next, taxSource: fee.taxSource };
  const allocations = [...(record.allocations ?? [])];
  allocations[allocationIndex] = { ...allocation, fees };
  return applyFeeChange(
    { ...record, allocations },
    allocationIndex,
    feeIndex,
    bills,
  );
}

/** 用户选择了账单：仅一条可登记费用时直接带入该费用。 */
export function applyBillSelection(
  record: InvoiceRegistrationDraft,
  allocationIndex: number,
  bills: InvoiceRegistrationBillSource[],
  records: InvoiceRegistrationDraft[] | undefined,
  recordIndex: number,
): InvoiceRegistrationDraft {
  const allocation = record.allocations?.[allocationIndex];
  const bill = bills.find((item) => item.id === allocation?.billId);
  if (!allocation || !bill) return record;
  const fees = registrableFees(bill);
  const allocations = [...(record.allocations ?? [])];
  allocations[allocationIndex] = {
    ...allocation,
    fees: fees.length === 1 ? [feeRow(fees[0])] : [],
  };
  const next = { ...record, allocations };
  return fees.length === 1
    ? applyFeeSelection(next, allocationIndex, 0, bills, records, recordIndex)
    : syncAllocationTotal(next, allocationIndex);
}

/**
 * 用户主动触发：按账单、费用列表顺序把发票总额逐项填至可用余额，
 * 余额不足时保留未分配差额，不补齐。
 */
export function autoAllocate(
  record: InvoiceRegistrationDraft,
  bills: InvoiceRegistrationBillSource[],
  records: InvoiceRegistrationDraft[] | undefined,
  recordIndex: number,
): InvoiceRegistrationDraft {
  let remaining = parseAmount(record.totalAmount) ?? ZERO;
  const allocations: InvoiceRegistrationAllocationDraft[] = [];
  for (const bill of bills) {
    const fees: InvoiceRegistrationFeeDraft[] = [];
    for (const fee of bill.fees) {
      if (!remaining.gt(0)) break;
      const available = availableFor(fee, bill.id, records, recordIndex);
      if (!available.gt(0)) continue;
      const take = Decimal.min(available, remaining);
      remaining = remaining.minus(take);
      fees.push(feeRow(fee, take));
    }
    if (!fees.length) continue;
    allocations.push({
      billId: bill.id,
      amountSource: 'auto',
      amount: plain(sumFees({ billId: bill.id, amount: '', fees })),
      fees,
    });
  }
  return { ...record, allocations };
}

export function hasManualAllocation(record?: InvoiceRegistrationDraft) {
  return (record?.allocations ?? []).some(
    (allocation) =>
      allocation?.amountSource === 'manual' ||
      (allocation?.fees ?? []).some(
        (fee) => fee?.amountSource === 'manual' || fee?.taxSource === 'manual',
      ),
  );
}

/** 发票待分配差额与各账单的费用差额；总额未填时发票差额为空。 */
export function recordGaps(record?: InvoiceRegistrationDraft) {
  const total = parseAmount(record?.totalAmount);
  const allocated = record ? sumAllocations(record) : ZERO;
  return {
    allocated,
    invoiceGap: total ? total.minus(allocated) : undefined,
    billGaps: (record?.allocations ?? []).map((allocation) => {
      const amount = parseAmount(allocation?.amount);
      return amount && allocation
        ? amount.minus(sumFees(allocation))
        : undefined;
    }),
  };
}
