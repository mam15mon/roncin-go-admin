import Decimal from 'decimal.js';

export const quantityOrPricePattern = /^(0|[1-9][0-9]{0,9})(\.[0-9]{1,4})?$/;
export const exchangeRatePattern = /^(0|[1-9][0-9]{0,9})(\.[0-9]{1,8})?$/;

/** 去掉首尾空白与小数尾部多余的零（如 "211.04500" → "211.045"），便于与固定位数正则兼容。 */
export function normalizeDecimalInput(value: string): string {
  return value
    .trim()
    .replace(/(\.\d*?)0+$/, '$1')
    .replace(/\.$/, '');
}

/** 费用金额展示口径：两位小数并去尾零（25.5 → "25.5"，76.5 → "76.5"，23 → "23"）。 */
export function calculateExactFeeTotal(
  quantity?: string,
  unitPrice?: string,
): string | undefined {
  const normalizedQuantity = quantity ? normalizeDecimalInput(quantity) : '';
  const normalizedUnitPrice = unitPrice ? normalizeDecimalInput(unitPrice) : '';
  if (
    !normalizedQuantity ||
    !normalizedUnitPrice ||
    !quantityOrPricePattern.test(normalizedQuantity) ||
    !quantityOrPricePattern.test(normalizedUnitPrice)
  ) {
    return undefined;
  }
  const total = new Decimal(normalizedQuantity).mul(
    new Decimal(normalizedUnitPrice),
  );
  if (!total.isPositive()) return undefined;
  return total
    .toFixed(2)
    .replace(/(\.\d*?)0+$/, '$1')
    .replace(/\.$/, '');
}

export function isPositiveExactDecimal(
  value: string,
  pattern: RegExp,
): boolean {
  return pattern.test(value) && new Decimal(value).isPositive();
}
