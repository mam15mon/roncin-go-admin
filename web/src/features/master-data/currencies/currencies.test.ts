import { beforeEach, describe, expect, it, vi } from 'vitest';

const listCurrencies = vi.hoisted(() => vi.fn());

vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceListCurrencies: listCurrencies,
}));

describe('币种候选缓存', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.resetModules();
  });

  it('并发读取币种时复用同一个请求并排除停用项', async () => {
    listCurrencies.mockResolvedValue({
      data: [
        { code: 'CNY', name: '人民币', enabled: true },
        { code: 'USD', name: '美元', enabled: false },
      ],
    });
    const { getCurrencyOptions } = await import('./currencies');

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
    const { getCurrencies } = await import('./currencies');

    await expect(getCurrencies()).rejects.toThrow('network');
    await expect(getCurrencies()).resolves.toEqual([]);

    expect(listCurrencies).toHaveBeenCalledTimes(2);
  });

  it('强制刷新替换缓存并返回最新数据（币种启停后的刷新入口）', async () => {
    listCurrencies
      .mockResolvedValueOnce({
        data: [{ code: 'CNY', name: '人民币', enabled: true }],
      })
      .mockResolvedValueOnce({
        data: [
          { code: 'CNY', name: '人民币', enabled: true },
          { code: 'EUR', name: '欧元', enabled: true },
        ],
      });
    const { getCurrencies } = await import('./currencies');

    await getCurrencies();
    const refreshed = await getCurrencies(true);

    expect(refreshed.map((currency) => currency.code)).toEqual(['CNY', 'EUR']);
    expect(listCurrencies).toHaveBeenCalledTimes(2);

    // 刷新后的缓存继续命中，不再发起请求。
    expect(await getCurrencies()).toBe(refreshed);
    expect(listCurrencies).toHaveBeenCalledTimes(2);
  });

  it('强制刷新后旧请求迟到失败，不得清掉新缓存引用', async () => {
    let rejectStaleRequest!: (error: Error) => void;
    listCurrencies.mockImplementationOnce(
      () =>
        new Promise<any>((_, reject) => {
          rejectStaleRequest = reject;
        }),
    );
    const { getCurrencies } = await import('./currencies');

    const stalePromise = getCurrencies();

    // 旧请求仍在途时强制刷新（模拟启停币种后的 getCurrencies(true)）。
    listCurrencies.mockResolvedValueOnce({
      data: [{ code: 'USD', name: '美元', enabled: true }],
    });
    const refreshedPromise = getCurrencies(true);

    rejectStaleRequest(new Error('旧请求网络失败'));
    await expect(stalePromise).rejects.toThrow('旧请求网络失败');

    // 新缓存不被旧失败清掉：强刷结果正常返回，后续调用继续命中且无第 3 次请求。
    expect(await refreshedPromise).toEqual([
      { code: 'USD', name: '美元', enabled: true },
    ]);
    expect(await getCurrencies()).toEqual([
      { code: 'USD', name: '美元', enabled: true },
    ]);
    expect(listCurrencies).toHaveBeenCalledTimes(2);
  });
});
