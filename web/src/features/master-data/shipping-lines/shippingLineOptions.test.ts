import { beforeEach, describe, expect, it, vi } from 'vitest';
import { searchShippingLineOptions } from './shippingLineOptions';

const listShippingLines = vi.hoisted(() => vi.fn());

vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceListShippingLines: listShippingLines,
}));

describe('船公司候选搜索', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('按关键字搜索启用船公司并输出中英文名与 SCAC 标签', async () => {
    listShippingLines.mockResolvedValue({
      data: [
        {
          id: 'line-1',
          scacCode: 'COSU',
          nameZh: '中远海运',
          nameEn: 'COSCO SHIPPING',
        },
      ],
    });

    const result = await searchShippingLineOptions('zhongyuan');

    expect(listShippingLines).toHaveBeenCalledWith({
      page: 1,
      pageSize: 50,
      keyword: 'zhongyuan',
      enabled: true,
    });
    expect(result).toEqual([
      {
        label: '中远海运 / COSCO SHIPPING (COSU)',
        value: 'line-1',
        code: 'COSU',
        name: '中远海运',
      },
    ]);
  });
});
