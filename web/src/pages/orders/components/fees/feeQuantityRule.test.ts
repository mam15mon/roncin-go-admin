import { describe, expect, it } from 'vitest';
import {
  feeQuantityRuleError,
  integerQuantityMessage,
} from './feeQuantityRule';

const units = [
  { id: 'ticket', quantityMustBeInteger: true },
  { id: 'cbm', quantityMustBeInteger: false },
];

describe('feeQuantityRuleError', () => {
  it('按单位规则判定数值，保留尾零整数', () => {
    expect(feeQuantityRuleError('4.4', 'ticket', units)).toBe(
      integerQuantityMessage,
    );
    expect(feeQuantityRuleError('2.0000', 'ticket', units)).toBeUndefined();
    expect(feeQuantityRuleError('4.4', 'cbm', units)).toBeUndefined();
  });

  it('旧小数仅在数量或单位变化时重新校验', () => {
    const baseline = { billingUnitId: 'ticket', quantity: '4.4000' };
    expect(
      feeQuantityRuleError('4.4', 'ticket', units, baseline),
    ).toBeUndefined();
    expect(feeQuantityRuleError('4.5', 'ticket', units, baseline)).toBe(
      integerQuantityMessage,
    );
    expect(feeQuantityRuleError('4.4', 'cbm', units, baseline)).toBeUndefined();
  });
});
