import { beforeEach, describe, expect, it, vi } from 'vitest';

const listCurrencies = vi.hoisted(() => vi.fn());
const listPartners = vi.hoisted(() => vi.fn());

vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceListCurrencies: listCurrencies,
}));

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceListPartners: listPartners,
}));

describe('候选项工具', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.resetModules();
  });

  it('按角色和启用状态搜索合作方并统一标签', async () => {
    listPartners.mockResolvedValue({
      data: [
        { id: 'p1', code: 'CUS001', legalName: '示例客户' },
        { id: 'p2', code: 'CUS002' },
      ],
    });
    const { searchPartnerOptions } = await import('./options');

    const result = await searchPartnerOptions('示例', { role: 1, enabled: true });

    expect(listPartners).toHaveBeenCalledWith({
      page: 1,
      pageSize: 50,
      keyword: '示例',
      role: 1,
      enabled: true,
    });
    expect(result).toEqual([
      {
        label: '示例客户 (CUS001)',
        value: 'p1',
        code: 'CUS001',
        name: '示例客户',
      },
      {
        label: 'CUS002',
        value: 'p2',
        code: 'CUS002',
        name: undefined,
      },
    ]);
  });

  it('并发读取币种时复用同一个请求并排除停用项', async () => {
    listCurrencies.mockResolvedValue({
      data: [
        { code: 'CNY', name: '人民币', enabled: true },
        { code: 'USD', name: '美元', enabled: false },
      ],
    });
    const { getCurrencyOptions } = await import('./options');

    const [first, second] = await Promise.all([
      getCurrencyOptions(),
      getCurrencyOptions(),
    ]);

    expect(listCurrencies).toHaveBeenCalledTimes(1);
    expect(first).toEqual([
      {
        label: 'CNY - 人民币',
        value: 'CNY',
        code: 'CNY',
        name: '人民币',
      },
    ]);
    expect(second).toEqual(first);
  });

  it('币种请求失败后允许重新加载', async () => {
    listCurrencies
      .mockRejectedValueOnce(new Error('network'))
      .mockResolvedValueOnce({ data: [] });
    const { getCurrencies } = await import('./options');

    await expect(getCurrencies()).rejects.toThrow('network');
    await expect(getCurrencies()).resolves.toEqual([]);

    expect(listCurrencies).toHaveBeenCalledTimes(2);
  });
});
