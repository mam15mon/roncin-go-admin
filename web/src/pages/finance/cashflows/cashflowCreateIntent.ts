import dayjs, { type Dayjs } from 'dayjs';
import Decimal from 'decimal.js';

type CashflowCreateValues = {
  direction: string;
  settlementPartyId: string;
  currency: string;
  amount: string;
  exchangeRate?: string;
  transactionDate: Dayjs;
  ourAccount: string;
  counterpartyAccount?: string;
  paymentMethod: string;
  bankReferenceNo?: string;
  note?: string;
};

/** 指纹与真正提交的载荷共用规范化，金额等价、空白差异不会变成另一笔登记。 */
export function buildCashflowCreateIntent(
  organizationId: string,
  values: CashflowCreateValues,
): Omit<API.CreateCashflowRequest, 'idempotencyKey'> {
  const optionalText = (value?: string) => value?.trim() || undefined;
  return {
    organizationId,
    direction: values.direction,
    settlementPartyId: values.settlementPartyId,
    currency: values.currency.trim().toUpperCase(),
    amount: new Decimal(values.amount).toFixed(),
    exchangeRate: optionalText(values.exchangeRate)
      ? new Decimal(values.exchangeRate as string).toFixed()
      : undefined,
    transactionDate: dayjs(values.transactionDate).format('YYYY-MM-DD'),
    ourAccount: values.ourAccount.trim(),
    counterpartyAccount: optionalText(values.counterpartyAccount),
    paymentMethod: values.paymentMethod.trim(),
    bankReferenceNo: optionalText(values.bankReferenceNo),
    note: optionalText(values.note),
  };
}
