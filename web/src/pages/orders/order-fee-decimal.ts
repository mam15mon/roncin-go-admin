import Decimal from 'decimal.js';

export const quantityOrPricePattern = /^(0|[1-9][0-9]{0,9})(\.[0-9]{1,4})?$/;
export const exchangeRatePattern = /^(0|[1-9][0-9]{0,9})(\.[0-9]{1,8})?$/;

export function calculateExactFeeTotal(
  quantity?: string,
  unitPrice?: string,
): string | undefined {
  if (
    !quantity ||
    !unitPrice ||
    !quantityOrPricePattern.test(quantity) ||
    !quantityOrPricePattern.test(unitPrice)
  ) {
    return undefined;
  }
  const total = new Decimal(quantity).mul(new Decimal(unitPrice));
  return total.isPositive() ? total.toFixed(8) : undefined;
}

export function isPositiveExactDecimal(
  value: string,
  pattern: RegExp,
): boolean {
  return pattern.test(value) && new Decimal(value).isPositive();
}

export function trimExactDecimal(value?: string): string {
  if (!value) return '-';
  return value.replace(/(\.\d*?)0+$/, '$1').replace(/\.$/, '');
}
