import { describe, expect, it } from 'vitest';
import { formatAmount, formatDate, trimDecimal } from './format';

describe('展示格式化', () => {
  it('按日期和日期时间两种口径格式化', () => {
    expect(formatDate('2026-08-30T12:34:56', 'date')).toBe('2026-08-30');
    expect(formatDate('2026-08-30T12:34:56')).toBe('2026-08-30 12:34:56');
    expect(formatDate(undefined)).toBe('-');
    expect(formatDate('invalid')).toBe('-');
  });

  it('格式化金额并拒绝无效数字', () => {
    expect(formatAmount('12345.6')).toBe('12,345.60');
    expect(formatAmount('12345.6789', 3)).toBe('12,345.679');
    expect(formatAmount('invalid')).toBe('-');
  });

  it('以字符串方式去除小数末尾零', () => {
    expect(trimDecimal('120.34000000')).toBe('120.34');
    expect(trimDecimal('120.00000000')).toBe('120');
    expect(trimDecimal(undefined)).toBe('-');
  });
});
