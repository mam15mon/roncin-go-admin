import { afterEach, describe, expect, it, vi } from 'vitest';
import { csvAmount, csvDecimal, downloadCsv, serializeCsv } from './csv';

afterEach(() => {
  vi.restoreAllMocks();
  vi.clearAllMocks();
});

describe('CSV 序列化', () => {
  it('保留中文 BOM、CRLF、列顺序及引号逗号换行，文本公式包含前导空白也被保护', () => {
    const values = [
      '中文,"备注"\n第二行',
      '=1+1',
      '  +SUM(A1)',
      '\t-1',
      '\r@A1',
      '\u00a0=1',
      '正常',
    ];
    const content = serializeCsv(values, [
      { header: '名称,备注', value: (value) => value },
    ]);
    expect(content).toBe(
      '\uFEFF"名称,备注"\r\n"中文,""备注""\n第二行"\r\n\'=1+1\r\n\'  +SUM(A1)\r\n\'\t-1\r\n"\'\r@A1"\r\n\'\u00a0=1\r\n正常',
    );
  });

  it('受控负金额和小数保持数值，非法数值公式仍然保护，空值不变成零', () => {
    expect(
      serializeCsv(
        ['-15362.50', '6.7183', '-1+2', ' \t=1'],
        [{ header: '金额', kind: 'number', value: (value) => value }],
      ),
    ).toBe("\uFEFF金额\r\n-15362.50\r\n6.7183\r\n'-1+2\r\n' \t=1");
    expect(csvAmount('15362.50000000')).toBe('15362.50');
    expect(csvAmount('-3.50000000')).toBe('-3.50');
    expect(csvAmount(undefined)).toBe('');
    expect(csvAmount('bad')).toBe('');
    expect(csvDecimal('6.71830000')).toBe('6.7183');
    expect(csvDecimal('3.50000000')).toBe('3.5');
    expect(csvDecimal(null)).toBe('');
    expect(serializeCsv([], [{ header: '名称', value: () => '' }])).toBeNull();
  });
});

describe('CSV 下载资源', () => {
  it.each([false, true])('点击下载后回收链接和 URL，点击异常=%s', (throws) => {
    const create = vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:csv');
    const revoke = vi
      .spyOn(URL, 'revokeObjectURL')
      .mockImplementation(() => {});
    const click = vi
      .spyOn(HTMLAnchorElement.prototype, 'click')
      .mockImplementation(function (this: HTMLAnchorElement) {
        expect(this.download).toBe('导出.csv');
        expect(this.isConnected).toBe(true);
        if (throws) throw new Error('下载失败');
      });
    if (throws)
      expect(() => downloadCsv('\uFEFF金额\r\n-3.50', '导出.csv')).toThrow(
        '下载失败',
      );
    else downloadCsv('\uFEFF金额\r\n-3.50', '导出.csv');
    expect(create).toHaveBeenCalledOnce();
    expect(click).toHaveBeenCalledOnce();
    expect(revoke).toHaveBeenCalledWith('blob:csv');
    expect(document.querySelector('a[download]')).toBeNull();
  });
});
