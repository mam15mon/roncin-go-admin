import { OrderFeeDirection, OrderFeeStatus } from '@/enums.generated';

export const RECEIVABLE = OrderFeeDirection.ORDER_FEE_DIRECTION_RECEIVABLE;
export const PAYABLE = OrderFeeDirection.ORDER_FEE_DIRECTION_PAYABLE;
export const FEE_DRAFT = OrderFeeStatus.ORDER_FEE_STATUS_DRAFT;
export const FEE_CONFIRMED = OrderFeeStatus.ORDER_FEE_STATUS_CONFIRMED;
export const FEE_BILLED = OrderFeeStatus.ORDER_FEE_STATUS_BILLED;
export const FEE_CANCELLED = OrderFeeStatus.ORDER_FEE_STATUS_CANCELLED;

export const FEE_STATUS_CODES: Record<string, number> = {
  ORDER_FEE_STATUS_DRAFT: FEE_DRAFT,
  ORDER_FEE_STATUS_CONFIRMED: FEE_CONFIRMED,
  ORDER_FEE_STATUS_BILLED: FEE_BILLED,
  ORDER_FEE_STATUS_CANCELLED: FEE_CANCELLED,
};

export const FEE_DIRECTION_CODES: Record<string, number> = {
  ORDER_FEE_DIRECTION_RECEIVABLE: RECEIVABLE,
  ORDER_FEE_DIRECTION_PAYABLE: PAYABLE,
};

export function feeDirectionCode(direction: unknown): number {
  if (typeof direction === 'number') return direction;
  return FEE_DIRECTION_CODES[String(direction)] ?? 0;
}

export function feeStatusCode(status: unknown): number {
  if (typeof status === 'number') return status;
  return FEE_STATUS_CODES[String(status)] ?? 0;
}
