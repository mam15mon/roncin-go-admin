import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { SelectOption } from './options';

const listCurrencies = vi.hoisted(() => vi.fn());
const listShippingLines = vi.hoisted(() => vi.fn());
const listPartners = vi.hoisted(() => vi.fn());

vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceListCurrencies: listCurrencies,
  masterDataServiceListShippingLines: listShippingLines,
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
        { id: 'p1', code: 'CUS001', legalName: '示例客户', isCasual: true },
        { id: 'p2', code: 'CUS002', isCasual: false },
      ],
    });
    const { searchPartnerOptions } = await import('./options');

    const result = await searchPartnerOptions('示例', {
      role: 1,
      enabled: true,
    });

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
        isCasual: true,
      },
      {
        label: 'CUS002',
        value: 'p2',
        code: 'CUS002',
        name: undefined,
        isCasual: false,
      },
    ]);
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
    const { searchShippingLineOptions } = await import('./options');

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
    expect(listCurrencies).toHaveBeenCalledWith({ enabledOnly: true });
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

  it('直接干预模式下禁用超额客户候选，仅提醒模式原样返回', async () => {
    const { disableCreditExceededOptions } = await import('./options');

    const options: SelectOption[] = [
      { label: '正常客户 (CUS001)', value: 'p1' },
      { label: '超额客户 (CUS002)', value: 'p2', creditExceeded: true },
      { label: '散客 (CUS003)', value: 'p3', isCasual: true },
    ];

    // 仅提醒模式（默认）：不做任何禁用。
    expect(disableCreditExceededOptions(options, false)).toEqual(options);

    // 直接干预模式：仅超额候选被置灰，其余不受影响；label 保持纯净。
    const disabled = disableCreditExceededOptions(options, true);
    expect(disabled[0].disabled).toBeUndefined();
    expect(disabled[1].disabled).toBe(true);
    expect(disabled[1].label).toBe('超额客户 (CUS002)');
    expect(disabled[2].disabled).toBeUndefined();
  });
});
