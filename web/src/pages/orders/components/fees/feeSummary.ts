import Decimal from 'decimal.js';

/** 原币层分组：不同币种不直接相加。 */
export type FeeCurrencyTotal = {
  currency: string;
  /** 含税原币金额 */
  gross: string;
  /** 不含税原币金额 */
  net: string;
};

/**
 * 单侧（应收或应付）已落库费用的本币汇总。含税来自 baseCurrencyAmount，
 * 未税来自 baseCurrencyNetAmount，二者取自同一行、同一汇率快照；任一缺失的行
 * 计入 unconvertedCount，不参与本币合计，也不按 0 处理。
 */
export type FeeSideSummary = {
  count: number;
  gross: string;
  net: string;
  unconvertedCount: number;
  currencies: FeeCurrencyTotal[];
};

export function summarizeFees(fees: API.OrderFee[]): FeeSideSummary {
  let gross = new Decimal(0);
  let net = new Decimal(0);
  let unconvertedCount = 0;
  const byCurrency = new Map<string, { gross: Decimal; net: Decimal }>();
  for (const fee of fees) {
    if (fee.baseCurrencyAmount && fee.baseCurrencyNetAmount) {
      gross = gross.plus(fee.baseCurrencyAmount);
      net = net.plus(fee.baseCurrencyNetAmount);
    } else {
      unconvertedCount += 1;
    }
    const currency = fee.currency || '-';
    const bucket = byCurrency.get(currency) ?? {
      gross: new Decimal(0),
      net: new Decimal(0),
    };
    bucket.gross = bucket.gross.plus(fee.totalAmount || 0);
    bucket.net = bucket.net.plus(fee.netAmount || 0);
    byCurrency.set(currency, bucket);
  }
  return {
    count: fees.length,
    gross: gross.toString(),
    net: net.toString(),
    unconvertedCount,
    currencies: [...byCurrency.entries()]
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([currency, value]) => ({
        currency,
        gross: value.gross.toString(),
        net: value.net.toString(),
      })),
  };
}

export type FeeProfitSummary = {
  gross: string;
  net: string;
  /** 毛利率（含税口径，百分数，一位小数） */
  rate: string;
  negative: boolean;
  unconvertedCount: number;
};

/** 预计毛利：两组分别为应收减应付；任一侧未取得数据返回 undefined。 */
export function computeFeeProfit(
  receivable: FeeSideSummary | undefined,
  payable: FeeSideSummary | undefined,
): FeeProfitSummary | undefined {
  if (!receivable || !payable) return undefined;
  const gross = new Decimal(receivable.gross).minus(payable.gross);
  const net = new Decimal(receivable.net).minus(payable.net);
  const recv = new Decimal(receivable.gross);
  return {
    gross: gross.toString(),
    net: net.toString(),
    rate: recv.gt(0) ? gross.div(recv).times(100).toFixed(1) : '0.0',
    negative: gross.lt(0),
    unconvertedCount: receivable.unconvertedCount + payable.unconvertedCount,
  };
}
