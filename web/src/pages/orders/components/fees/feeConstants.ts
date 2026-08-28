export const RECEIVABLE = 1;
export const PAYABLE = 2;
export const FEE_DRAFT = 1;
export const FEE_CONFIRMED = 2;
export const FEE_BILLED = 3;
export const FEE_CANCELLED = 4;

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
