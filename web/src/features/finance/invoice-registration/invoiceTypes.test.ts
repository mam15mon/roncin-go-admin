import { describe, expect, it } from 'vitest';
import {
  INVOICE_TYPE_OPTIONS,
  invoiceTypeLabel,
  isInvoiceType,
  isSpecialInvoiceType,
} from './invoiceTypes';

describe('发票票种', () => {
  it('六类税票中文名与专票判定，不含形式发票与旧值', () => {
    expect(INVOICE_TYPE_OPTIONS.map((item) => item.label)).toEqual([
      '数电普票',
      '数电专票',
      '纸质普票',
      '纸质专票',
      '电子普票',
      '电子专票',
    ]);
    expect(invoiceTypeLabel('ELECTRONIC_SPECIAL')).toBe('电子专票');
    expect(invoiceTypeLabel(undefined)).toBe('-');
    expect(invoiceTypeLabel('UNKNOWN')).toBe('UNKNOWN');
    expect(isInvoiceType('NORMAL')).toBe(false);
    expect(
      INVOICE_TYPE_OPTIONS.filter((item) => isSpecialInvoiceType(item.value))
        .length,
    ).toBe(3);
    expect(isSpecialInvoiceType('SPECIAL')).toBe(false);
  });
});
