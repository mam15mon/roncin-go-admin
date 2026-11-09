import Decimal from 'decimal.js';
import type { FinanceLedgerAmount, FinanceLedgerMatchedSummary } from './types';

// numeric(28,8) 加上 int64 记录数的累加位数，避免大额正负相抵前丢掉小数。
const LedgerDecimal = Decimal.clone({ precision: 48 });

/** 原币和本币身份共同分桶，保留每组未折算的记录数。 */
export function aggregateLedgerAmounts(
  amounts: FinanceLedgerAmount[],
  count = amounts.length,
): FinanceLedgerMatchedSummary {
  const buckets = new Map<string, FinanceLedgerAmount>();
  for (const item of amounts) {
    const key = JSON.stringify([
      item.direction,
      item.currency,
      item.baseCurrency,
    ]);
    const current = buckets.get(key);
    const converted =
      item.baseCurrency !== undefined && item.baseAmount !== undefined;
    const missing = item.unconvertedCount ?? (converted ? 0 : 1);
    buckets.set(key, {
      direction: item.direction,
      currency: item.currency,
      amount: new LedgerDecimal(current?.amount ?? '0')
        .plus(item.amount)
        .toFixed(),
      baseCurrency: item.baseCurrency,
      baseAmount: converted
        ? new LedgerDecimal(current?.baseAmount ?? '0')
            .plus(item.baseAmount ?? '0')
            .toFixed()
        : current?.baseAmount,
      unconvertedCount: (current?.unconvertedCount ?? 0) + missing,
    });
  }
  return { count, amounts: [...buckets.values()] };
}

export function sumAmountsByCurrency(
  amounts: FinanceLedgerAmount[],
  base: boolean,
  sign: (item: FinanceLedgerAmount) => number = () => 1,
): Map<string, Decimal> {
  const sums = new Map<string, Decimal>();
  for (const item of amounts) {
    const currency = base ? item.baseCurrency : item.currency;
    const value = base ? item.baseAmount : item.amount;
    if (currency === undefined || value === undefined) continue;
    sums.set(
      currency,
      (sums.get(currency) ?? new LedgerDecimal(0)).plus(
        new LedgerDecimal(value).mul(sign(item)),
      ),
    );
  }
  return sums;
}
