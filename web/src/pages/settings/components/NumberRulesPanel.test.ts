import { describe, expect, it } from 'vitest';
import { DOC_TYPES, filterVisibleNumberRules } from './NumberRulesPanel';

describe('编号规则可见范围', () => {
  it('仅开放已经接入订单流程的订单编号', () => {
    expect(DOC_TYPES.map((item) => item.key)).toEqual(['DOCUMENT_TYPE_ORDER']);
  });

  it('过滤接口返回的未接入编号规则', () => {
    const rules = [
      { id: 'order', documentType: 1 },
      { id: 'bill', documentType: 2 },
      { id: 'house', documentType: 9 },
    ] as API.NumberRule[];

    expect(filterVisibleNumberRules(rules).map((item) => item.id)).toEqual([
      'order',
    ]);
  });
});
