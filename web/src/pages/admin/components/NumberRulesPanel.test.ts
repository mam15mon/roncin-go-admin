import { describe, expect, it } from 'vitest';
import { DOC_TYPES, filterVisibleNumberRules } from './NumberRulesPanel';

describe('单据编号规则体系', () => {
  it('仅展示当前支持的 9 类业务单据编号规则', () => {
    expect(DOC_TYPES).toHaveLength(9);
    const keys = DOC_TYPES.map((item) => item.key);
    expect(keys).toContain('DOCUMENT_TYPE_ORDER');
    expect(keys).toContain('DOCUMENT_TYPE_BILL');
    expect(keys).toContain('DOCUMENT_TYPE_BILL_BATCH');
    expect(keys).toContain('DOCUMENT_TYPE_INVOICE');
    expect(keys).toContain('DOCUMENT_TYPE_RECEIPT_PAYMENT');
    expect(keys).toContain('DOCUMENT_TYPE_WRITE_OFF');
    expect(keys).toContain('DOCUMENT_TYPE_COMMISSION');
    expect(keys).toContain('DOCUMENT_TYPE_NETTING');
    expect(keys).toContain('DOCUMENT_TYPE_CONTRACT');
  });

  it('隐藏已删除的编号规则与未知类型', () => {
    const rules = [
      { id: 'order', documentType: 1 },
      { id: 'bill', documentType: 2 },
      { id: 'batch', documentType: 14 },
      { id: 'house', documentType: 9 },
      { id: 'quotation', documentType: 3 },
      { id: 'freight-rate', documentType: 12 },
      { id: 'internal-reference', documentType: 7 },
      { id: 'customer-reference', documentType: 8 },
      { id: 'unknown', documentType: 999 },
    ] as API.NumberRule[];

    expect(filterVisibleNumberRules(rules).map((item) => item.id)).toEqual([
      'order',
      'bill',
      'batch',
    ]);
  });

  it('前端新建规则默认前缀与后端保持一致', () => {
    const prefixes = Object.fromEntries(
      DOC_TYPES.map((item) => [item.key, item.defaultPrefix]),
    );

    expect(prefixes.DOCUMENT_TYPE_INVOICE).toBe('');
  });
});
