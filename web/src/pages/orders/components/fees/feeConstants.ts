import { normalizeOrderFeeStatus } from '@/constants/statusMeta';
import { OrderFeeDirection, OrderFeeStatus } from '@/enums.generated';

export const RECEIVABLE = OrderFeeDirection.ORDER_FEE_DIRECTION_RECEIVABLE;
export const PAYABLE = OrderFeeDirection.ORDER_FEE_DIRECTION_PAYABLE;
export const FEE_UNBILLED = OrderFeeStatus.ORDER_FEE_STATUS_UNBILLED;
export const FEE_BILLED = OrderFeeStatus.ORDER_FEE_STATUS_BILLED;
export const FEE_CANCELLED = OrderFeeStatus.ORDER_FEE_STATUS_CANCELLED;

export const FEE_DIRECTION_CODES: Record<string, number> = {
  ORDER_FEE_DIRECTION_RECEIVABLE: RECEIVABLE,
  ORDER_FEE_DIRECTION_PAYABLE: PAYABLE,
};

export function feeDirectionCode(direction: unknown): number {
  if (typeof direction === 'number') return direction;
  return FEE_DIRECTION_CODES[String(direction)] ?? 0;
}

export function feeStatusCode(status: unknown): number {
  return normalizeOrderFeeStatus(status);
}
