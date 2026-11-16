import { describe, expect, it } from 'vitest';
import { matchesSearch } from './search';

describe('名称搜索', () => {
  it.each(['海运', 'haiyunfei', 'HYF', 'hai yun fei', 'yunfei'])(
    '%s 命中费用名称',
    (keyword) => {
      expect(matchesSearch(keyword, '海运费')).toBe(true);
    },
  );
  it('原文、英文代码与空搜索保持原有语义，非文本值不会伪装成名称', () => {
    expect(matchesSearch('ocean', 'Ocean Freight')).toBe(true);
    expect(matchesSearch('usd', 'USD')).toBe(true);
    expect(matchesSearch('0', 0)).toBe(true);
    expect(matchesSearch(' ', undefined)).toBe(true);
    expect(
      matchesSearch('dcf', '海运费', null, undefined, { name: '订舱费' }),
    ).toBe(false);
  });
  it('中文标点和英文代码不会打断中文名称的全拼与首字母', () => {
    expect(matchesSearch('dailifei', '代理费 (AGENCY)')).toBe(true);
    expect(matchesSearch('dlf', '代理费 (AGENCY)')).toBe(true);
    expect(matchesSearch('chongqing', '重庆港')).toBe(true);
    expect(matchesSearch('cqg', '重庆港')).toBe(true);
    expect(matchesSearch('minhang', '闵行新区')).toBe(true);
    expect(matchesSearch('mhxq', '闵行新区')).toBe(true);
    expect(matchesSearch('huanxiangfei', '还箱费')).toBe(true);
    expect(matchesSearch('hgf', '还柜费')).toBe(true);
  });
});
