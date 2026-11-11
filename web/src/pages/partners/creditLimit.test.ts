import { describe, expect, it } from 'vitest';
import { creditLimitFields, hasInvalidCreditCurrency } from './creditLimit';

describe('信用额度本币映射', () => {
  it.each(['CNY', 'USD'])(
    '额度以公司本币%s提交，不绑定结算币种',
    (currency) => {
      expect(creditLimitFields('1000.01', currency)).toEqual({
        creditLimitMinor: '100001',
        creditCurrency: currency,
      });
      expect(creditLimitFields(0, currency)).toEqual({
        creditLimitMinor: '0',
        creditCurrency: currency,
      });
    },
  );
  it('未配置不注入额度，身份本币未就绪时拒绝提交金额', () => {
    expect(creditLimitFields(undefined, 'CNY')).toEqual({});
    expect(creditLimitFields(null, undefined)).toEqual({});
    expect(() => creditLimitFields(100, undefined)).toThrow(
      '当前公司本币未就绪',
    );
  });
  it('非法原币额度必须明确重录，不把原金额重新标成本币', () => {
    expect(
      hasInvalidCreditCurrency(
        { creditLimitMinor: '100000', creditCurrency: 'USD' },
        'CNY',
      ),
    ).toBe(true);
    expect(hasInvalidCreditCurrency({ creditLimitMinor: '0' }, 'CNY')).toBe(
      true,
    );
    expect(
      hasInvalidCreditCurrency(
        { creditLimitMinor: '100000', creditCurrency: 'USD' },
        'USD',
      ),
    ).toBe(false);
    expect(hasInvalidCreditCurrency({}, 'CNY')).toBe(false);
  });
});
