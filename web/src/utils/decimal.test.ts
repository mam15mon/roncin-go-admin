import { describe, expect, it } from 'vitest';
import {
  calculateExactFeeTotal,
  exchangeRatePattern,
  isPositiveExactDecimal,
  normalizeDecimalInput,
} from './decimal';

describe('订单费用十进制计算', () => {
  it('不经过 Number 精确计算常见浮点陷阱', () => {
    expect(calculateExactFeeTotal('0.1', '0.2')).toBe('0.02');
  });

  it('按展示口径输出两位小数并去尾零', () => {
    expect(calculateExactFeeTotal('1.2345', '6.7891')).toBe('8.38');
    expect(calculateExactFeeTotal('25.5', '3')).toBe('76.5');
    expect(calculateExactFeeTotal('23', '1')).toBe('23');
  });

  it('容忍输入中的小数尾部零', () => {
    expect(calculateExactFeeTotal('211.04500', '1')).toBe('211.05');
    expect(normalizeDecimalInput(' 211.04500 ')).toBe('211.045');
    expect(normalizeDecimalInput('1.0000')).toBe('1');
    expect(normalizeDecimalInput('100')).toBe('100');
  });

  it('不对超出录入精度的值做静默舍入', () => {
    expect(calculateExactFeeTotal('1', '1.00001')).toBeUndefined();
  });

  it('接受八位小数汇率且拒绝第九位', () => {
    expect(isPositiveExactDecimal('0.12345678', exchangeRatePattern)).toBe(
      true,
    );
    expect(isPositiveExactDecimal('0.123456789', exchangeRatePattern)).toBe(
      false,
    );
  });
});
