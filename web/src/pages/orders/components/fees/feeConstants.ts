import { OrderFeeDirection } from '@/enums.generated';

export const RECEIVABLE = OrderFeeDirection.ORDER_FEE_DIRECTION_RECEIVABLE;
export const PAYABLE = OrderFeeDirection.ORDER_FEE_DIRECTION_PAYABLE;

export const FEE_DIRECTION_CODES: Record<string, number> = {
  ORDER_FEE_DIRECTION_RECEIVABLE: RECEIVABLE,
  ORDER_FEE_DIRECTION_PAYABLE: PAYABLE,
};

export function feeDirectionCode(direction: unknown): number {
  if (typeof direction === 'number') return direction;
  return FEE_DIRECTION_CODES[String(direction)] ?? 0;
}
