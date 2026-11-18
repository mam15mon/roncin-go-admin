import { describe, expect, it } from 'vitest';
import { paymentTermsFromForm, paymentTermsToForm } from './PaymentTermsFields';

describe('收款约定 wire 与表单转换', () => {
  it.each([
    { rule: {}, mode: 'none', expected: {} },
    {
      rule: { paymentTermsDays: 0 },
      mode: 'cash',
      expected: { paymentTermsDays: 0 },
    },
    {
      rule: { paymentTermsDays: 45 },
      mode: 'days',
      expected: { paymentTermsDays: 45 },
    },
  ])('保留 $mode 的存在性', ({ rule, mode, expected }) => {
    const values = paymentTermsToForm(rule);
    expect(values.paymentTermsMode).toBe(mode);
    expect(paymentTermsFromForm(values)).toEqual(expected);
  });

  it('切回未约定时清除旧天数，未约定额度时省略成对字段', () => {
    const payload = paymentTermsFromForm({
      paymentTermsMode: 'none',
      paymentTermsDays: 45,
      creditLimit: null,
    });
    expect(payload).not.toHaveProperty('paymentTermsDays');
    expect(payload).not.toHaveProperty('creditLimitMinor');
    expect(payload).not.toHaveProperty('creditCurrency');
  });

  it.each([0, -1, 3651, 1.5, undefined])(
    '拒绝 N 天的非法值 %s',
    (paymentTermsDays) => {
      expect(() =>
        paymentTermsFromForm({ paymentTermsMode: 'days', paymentTermsDays }),
      ).toThrow('1–3650');
    },
  );

  it('最大账期和本币额度可往返，显式零额度保持存在', () => {
    const rule = {
      paymentTermsDays: 3650,
      creditLimitMinor: '100001',
      creditCurrency: 'USD',
    };
    expect(
      paymentTermsFromForm(paymentTermsToForm(rule, 'USD'), 'USD'),
    ).toEqual(rule);
    expect(
      paymentTermsFromForm({ paymentTermsMode: 'cash', creditLimit: 0 }, 'CNY'),
    ).toEqual({
      paymentTermsDays: 0,
      creditLimitMinor: '0',
      creditCurrency: 'CNY',
    });
  });

  it('非本币或币种缺失的原额度不自动改挂为本币', () => {
    expect(
      paymentTermsToForm(
        { creditLimitMinor: '10000', creditCurrency: 'USD' },
        'CNY',
      ).creditLimit,
    ).toBeUndefined();
    expect(
      paymentTermsToForm({ creditLimitMinor: '10000' }, 'CNY').creditLimit,
    ).toBeUndefined();
    expect(() =>
      paymentTermsFromForm({ paymentTermsMode: 'none', creditLimit: 100 }),
    ).toThrow('本币未就绪');
  });
});
