import { describe, expect, it } from 'vitest';
import { DOC_TYPES, filterVisibleNumberRules } from './NumberRulesPanel';

describe('单据编号规则体系', () => {
  it('全量支持 13 类业务单据编号规则', () => {
    expect(DOC_TYPES).toHaveLength(13);
    const keys = DOC_TYPES.map((item) => item.key);
    expect(keys).toContain('DOCUMENT_TYPE_ORDER');
    expect(keys).toContain('DOCUMENT_TYPE_BILL');
    expect(keys).toContain('DOCUMENT_TYPE_BILL_BATCH');
    expect(keys).toContain('DOCUMENT_TYPE_INVOICE');
    expect(keys).toContain('DOCUMENT_TYPE_RECEIPT_PAYMENT');
    expect(keys).toContain('DOCUMENT_TYPE_WRITE_OFF');
    expect(keys).toContain('DOCUMENT_TYPE_COMMISSION');
    expect(keys).toContain('DOCUMENT_TYPE_HOUSE_BILL');
    expect(keys).toContain('DOCUMENT_TYPE_QUOTATION');
    expect(keys).toContain('DOCUMENT_TYPE_CONTRACT');
    expect(keys).toContain('DOCUMENT_TYPE_FREIGHT_RATE');
    expect(keys).toContain('DOCUMENT_TYPE_INTERNAL_REFERENCE');
    expect(keys).toContain('DOCUMENT_TYPE_CUSTOMER_REFERENCE');
  });

  it('保留所有 13 类合法单据规则', () => {
    const rules = [
      { id: 'order', documentType: 1 },
      { id: 'bill', documentType: 2 },
      { id: 'batch', documentType: 14 },
      { id: 'house', documentType: 9 },
      { id: 'unknown', documentType: 999 },
    ] as API.NumberRule[];

    expect(filterVisibleNumberRules(rules).map((item) => item.id)).toEqual([
      'order',
      'bill',
      'batch',
      'house',
    ]);
  });
});
