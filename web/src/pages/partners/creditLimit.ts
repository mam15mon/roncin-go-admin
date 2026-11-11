import Decimal from 'decimal.js';

export function hasInvalidCreditCurrency(
  rule: API.PartnerSettlementRule | null | undefined,
  baseCurrency: string | undefined,
) {
  return (
    rule?.creditLimitMinor != null &&
    (!baseCurrency || rule.creditCurrency !== baseCurrency)
  );
}

export function creditLimitFields(
  amount: string | number | null | undefined,
  baseCurrency: string | undefined,
): Pick<API.PartnerSettlementRuleInput, 'creditLimitMinor' | 'creditCurrency'> {
  if (amount == null || amount === '') return {};
  if (!baseCurrency) throw new Error('当前公司本币未就绪，不能保存信用额度');
  return {
    creditLimitMinor: new Decimal(amount).mul(100).toFixed(0),
    creditCurrency: baseCurrency,
  };
}
