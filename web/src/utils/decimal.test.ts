import { describe, expect, it } from 'vitest';
import {
  calculateExactFeeTotal,
  exchangeRatePattern,
  isPositiveExactDecimal,
  trimExactDecimal,
} from './decimal';

describe('订单费用十进制计算', () => {
  it('不经过 Number 精确计算常见浮点陷阱', () => {
    expect(calculateExactFeeTotal('0.1', '0.2')).toBe('0.02000000');
  });

  it('保留四位数量乘四位单价的完整八位结果', () => {
    expect(calculateExactFeeTotal('1.2345', '6.7891')).toBe('8.38114395');
  });

  it('不对超出录入精度的值做静默舍入', () => {
    expect(calculateExactFeeTotal('1', '1.00001')).toBeUndefined();
  });

  it('仅移除展示用的末尾零', () => {
    expect(trimExactDecimal('120.34000000')).toBe('120.34');
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
