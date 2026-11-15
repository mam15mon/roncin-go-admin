import type { Dayjs } from 'dayjs';
import Decimal from 'decimal.js';

export type InvoiceRegistrationFeeSource = {
  orderFeeId: string;
  feeCode?: string;
  feeName?: string;
  totalAmount?: string;
  invoiceableAmount?: string;
  sourceTaxRate?: string;
};
export type InvoiceRegistrationBillSource = {
  id: string;
  version: string;
  billNo?: string;
  currency?: string;
  fees: InvoiceRegistrationFeeSource[];
};
/** 字段来源：auto 为系统带入可被联动改写；manual 为用户手改，联动不再覆盖。 */
export type InvoiceRegistrationFieldSource = 'auto' | 'manual';
export type InvoiceRegistrationFeeDraft = {
  orderFeeId: string;
  amount: string;
  taxAmount: string;
  amountSource?: InvoiceRegistrationFieldSource;
  taxSource?: InvoiceRegistrationFieldSource;
};
export type InvoiceRegistrationAllocationDraft = {
  billId: string;
  amount: string;
  amountSource?: InvoiceRegistrationFieldSource;
  fees: InvoiceRegistrationFeeDraft[];
};
export type InvoiceRegistrationDraft = {
  sourceInvoiceId?: string;
  invoiceProfileId?: string;
  invoiceType: string;
  taxInvoiceNo?: string;
  invoiceDate: Dayjs;
  totalAmount?: string;
  note?: string;
  allocations: InvoiceRegistrationAllocationDraft[];
};

export function invoiceBillSource(
  bill: API.FinanceBill,
): InvoiceRegistrationBillSource {
  if (!bill.id || !bill.version) throw new Error('账单缺少登记所需身份');
  return {
    id: bill.id,
    version: bill.version,
    billNo: bill.billNo,
    currency: bill.currency,
    fees: (bill.lines ?? [])
      .filter((line) => line.active === true)
      .map((line) => {
        if (!line.orderFeeId) throw new Error('来源费用缺少业务标识');
        return {
          orderFeeId: line.orderFeeId,
          feeCode: line.feeCode,
          feeName: line.feeName,
          totalAmount: line.totalAmount,
          invoiceableAmount: line.invoiceableAmount,
          sourceTaxRate: line.taxRate,
        };
      }),
  };
}

function registrationAmount(
  value: string | undefined,
  positive = true,
): Decimal {
  const text = value?.trim() ?? '';
  if (!/^(?:0|[1-9]\d{0,19})(?:\.\d{1,8})?$/.test(text))
    throw new Error('请填写有效金额，最多8位小数');
  const amount = new Decimal(text);
  if (positive && !amount.gt(0)) throw new Error('含税金额必须大于0');
  return amount;
}

/** 只规范化明确填写的值并检查三层守恒，不自动分摊或修改税额。 */
export function invoiceRegistrationPayload(
  records: InvoiceRegistrationDraft[],
  bills: InvoiceRegistrationBillSource[],
): API.InvoiceRegistrationInput[] {
  const used = new Map<string, Decimal>();
  const numbers = new Set<string>();
  return records.map((record, index) => {
    const prefix = `第${index + 1}张发票：`;
    const total = registrationAmount(record.totalAmount);
    const number = record.taxInvoiceNo?.trim();
    if (!number || numbers.has(number))
      throw new Error(`${prefix}请填写不重复的发票号码`);
    numbers.add(number);
    if (!record.allocations?.length)
      throw new Error(`${prefix}至少添加一项账单分配`);
    const billIDs = new Set<string>();
    let allocated = new Decimal(0);
    const allocations = record.allocations.map((allocation) => {
      const source = bills.find((bill) => bill.id === allocation.billId);
      if (!source || billIDs.has(source.id))
        throw new Error(`${prefix}账单来源缺失或重复`);
      billIDs.add(source.id);
      const amount = registrationAmount(allocation.amount);
      allocated = allocated.plus(amount);
      if (!allocation.fees?.length)
        throw new Error(`${prefix}至少添加一项费用分配`);
      let feeTotal = new Decimal(0);
      const feeIDs = new Set<string>();
      const fees = allocation.fees.map((fee) => {
        const line = source.fees.find(
          (item) => item.orderFeeId === fee.orderFeeId,
        );
        if (!line || feeIDs.has(fee.orderFeeId))
          throw new Error(`${prefix}费用来源缺失或重复`);
        feeIDs.add(fee.orderFeeId);
        const gross = registrationAmount(fee.amount);
        const tax = registrationAmount(fee.taxAmount, false);
        if (tax.gt(gross))
          throw new Error(`${prefix}登记税额不能超过本次含税金额`);
        feeTotal = feeTotal.plus(gross);
        const key = `${source.id}:${fee.orderFeeId}`;
        const sum = (used.get(key) ?? new Decimal(0)).plus(gross);
        const available = line.invoiceableAmount ?? line.totalAmount;
        if (available !== undefined && sum.gt(new Decimal(available)))
          throw new Error(`${prefix}费用累计分配超过可登记余额`);
        used.set(key, sum);
        return {
          orderFeeId: fee.orderFeeId,
          amount: gross.toFixed(8),
          taxAmount: tax.toFixed(8),
        };
      });
      if (!feeTotal.eq(amount))
        throw new Error(`${prefix}费用合计须等于账单分配金额`);
      return {
        billId: source.id,
        expectedVersion: source.version,
        amount: amount.toFixed(8),
        fees: fees.sort((left, right) =>
          left.orderFeeId.localeCompare(right.orderFeeId),
        ),
      };
    });
    if (!allocated.eq(total))
      throw new Error(`${prefix}账单分配合计须等于发票总额`);
    if (!record.invoiceProfileId || !record.invoiceDate?.isValid())
      throw new Error(`${prefix}请完整填写开票抬头与日期`);
    return {
      invoiceProfileId: record.invoiceProfileId,
      invoiceType: record.invoiceType,
      taxInvoiceNo: number,
      invoiceDate: record.invoiceDate.format('YYYY-MM-DD'),
      totalAmount: total.toFixed(8),
      note: record.note?.trim() || undefined,
      allocations: allocations.sort((left, right) =>
        left.billId.localeCompare(right.billId),
      ),
    };
  });
}
