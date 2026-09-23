import Decimal from 'decimal.js';

export const integerQuantityMessage = '该计费单位的数量必须为正整数';

type BillingUnitRule = { id?: string; quantityMustBeInteger?: boolean };
type FeeQuantityBaseline = { billingUnitId?: string; quantity?: string };

export function feeQuantityRuleError(
  quantity: string | number | undefined,
  billingUnitId: string | undefined,
  billingUnits: BillingUnitRule[],
  baseline?: FeeQuantityBaseline,
): string | undefined {
  if (quantity === undefined || quantity === '' || !billingUnitId)
    return undefined;
  const unit = billingUnits.find((item) => item.id === billingUnitId);
  if (!unit?.quantityMustBeInteger) return undefined;
  try {
    const value = new Decimal(quantity);
    if (
      baseline?.billingUnitId === billingUnitId &&
      baseline.quantity !== undefined &&
      value.equals(new Decimal(baseline.quantity))
    ) {
      return undefined;
    }
    return value.isPositive() && value.isInteger()
      ? undefined
      : integerQuantityMessage;
  } catch {
    return integerQuantityMessage;
  }
}
