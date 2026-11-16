import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  CURRENCY_QUERY_PREFIX,
  currencyOptionsQueryOptions,
  currencyQueryOptions,
  fetchCurrencyOptions,
  getCurrencies,
  getCurrencyOptions,
} from './currencies';

const listCurrencies = vi.hoisted(() => vi.fn());
vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceListCurrencies: listCurrencies,
}));

beforeEach(() => vi.resetAllMocks());

describe('币种工作区查询', () => {
  it('远程字段的显式取数入口传递取消信号，原生字段入口不接收框架参数', async () => {
    listCurrencies.mockResolvedValue({
      success: true,
      data: [{ code: 'USD', enabled: true }],
    });
    const options = {
      signal: new AbortController().signal,
      skipErrorHandler: true,
    };
    expect(await fetchCurrencyOptions(options)).toEqual([
      { label: 'USD', value: 'USD', code: 'USD', name: undefined },
    ]);
    expect(listCurrencies).toHaveBeenCalledWith({ enabledOnly: true }, options);
  });
  it('纯请求函数不保留跨客户端缓存，并保留启用候选映射', async () => {
    listCurrencies.mockResolvedValue({
      data: [
        { code: 'CNY', name: '人民币', enabled: true },
        { code: 'USD', name: '美元', enabled: false },
      ],
    });
    const [first, second] = await Promise.all([
      getCurrencyOptions(),
      getCurrencyOptions(),
    ]);
    expect(listCurrencies).toHaveBeenCalledTimes(2);
    expect(first).toEqual([
      { label: 'CNY - 人民币', value: 'CNY', code: 'CNY', name: '人民币' },
    ]);
    expect(second).toEqual(first);
  });

  it('同客户端共享在途请求，统一失效后取最新币种', async () => {
    listCurrencies
      .mockResolvedValueOnce({ data: [{ code: 'CNY', enabled: true }] })
      .mockResolvedValueOnce({ data: [{ code: 'EUR', enabled: true }] });
    const client = createTestQueryClient();
    const options = { ...currencyQueryOptions(), staleTime: Infinity };
    const [first, second] = await Promise.all([
      client.fetchQuery(options),
      client.fetchQuery(options),
    ]);
    expect(first).toEqual([{ code: 'CNY', enabled: true }]);
    expect(second).toBe(first);
    expect(currencyOptionsQueryOptions().queryKey).toEqual(options.queryKey);
    expect(listCurrencies).toHaveBeenCalledOnce();
    await client.invalidateQueries({ queryKey: CURRENCY_QUERY_PREFIX });
    expect(await client.fetchQuery(options)).toEqual([
      { code: 'EUR', enabled: true },
    ]);
    expect(listCurrencies).toHaveBeenCalledTimes(2);
    client.clear();
  });

  it('不同工作区客户端不共享缓存，clear后的旧请求不回写新实例', async () => {
    let resolveOld!: (value: API.ListCurrenciesResponse) => void;
    listCurrencies
      .mockImplementationOnce(
        () =>
          new Promise<API.ListCurrenciesResponse>((resolve) => {
            resolveOld = resolve;
          }),
      )
      .mockResolvedValueOnce({ data: [{ code: 'USD', enabled: true }] });
    const oldClient = createTestQueryClient();
    const currentClient = createTestQueryClient();
    const oldRequest = oldClient
      .fetchQuery(currencyQueryOptions())
      .catch(() => undefined);
    oldClient.clear();
    expect(await currentClient.fetchQuery(currencyQueryOptions())).toEqual([
      { code: 'USD', enabled: true },
    ]);
    resolveOld({ data: [{ code: 'CNY', enabled: true }] });
    await oldRequest;
    expect(
      oldClient.getQueryData(currencyQueryOptions().queryKey),
    ).toBeUndefined();
    expect(currentClient.getQueryData(currencyQueryOptions().queryKey)).toEqual(
      [{ code: 'USD', enabled: true }],
    );
    currentClient.clear();
  });

  it.each([undefined, { success: false }, { success: true, data: null }])(
    '响应 %j 不能当作空候选',
    async (response) => {
      listCurrencies.mockResolvedValueOnce(response);
      await expect(getCurrencies()).rejects.toThrow('币种数据加载失败');
    },
  );

  it('成功省略 repeated 是合法空列表，失败后可以重试', async () => {
    listCurrencies
      .mockRejectedValueOnce(new Error('网络失败'))
      .mockResolvedValueOnce({ success: true });
    await expect(getCurrencies()).rejects.toThrow('网络失败');
    await expect(getCurrencies()).resolves.toEqual([]);
  });
});
