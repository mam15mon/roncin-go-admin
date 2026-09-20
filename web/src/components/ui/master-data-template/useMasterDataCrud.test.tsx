import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { act, renderHook, waitFor } from '@testing-library/react';
import React from 'react';
import { describe, expect, it, vi } from 'vitest';
import { useMasterDataCrud } from './useMasterDataCrud';

type Item = {
  id: string;
  code: string;
  name: string;
  enabled: boolean;
};

const item1: Item = {
  id: 'item-1',
  code: 'CNSHG',
  name: '上海港',
  enabled: true,
};
const item2: Item = {
  id: 'item-2',
  code: 'CNNGB',
  name: '宁波港',
  enabled: true,
};

/**
 * hook 测试包装：每个用例独立 QueryClient，防止缓存串味（与
 * renderWithClient 同策略，以 wrapper 形式供 renderHook 使用）。
 */
function createHookWrapper() {
  const queryClient = createTestQueryClient();
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  return { queryClient, wrapper };
}

describe('useMasterDataCrud', () => {
  it('按服务端分页加载数据并读取启停统计', async () => {
    const fetchList = vi.fn(
      async (query: { page: number; pageSize: number; enabled?: boolean }) => {
        if (query.enabled === true) return { data: [], total: 180 };
        if (query.enabled === false) return { data: [], total: 70 };
        const item: Item = {
          id: `item-${query.page}`,
          code: query.page === 1 ? 'CNSHG' : 'CNNGB',
          name: query.page === 1 ? '上海港' : '宁波港',
          enabled: true,
        };
        return { data: [item], total: 250 };
      },
    );
    const { wrapper } = createHookWrapper();
    const { result } = renderHook(
      () =>
        useMasterDataCrud<Item, Item>({
          entityName: '港口',
          fetchList,
          mapItem: (item) => item,
          createItem: vi.fn(),
          updateItem: vi.fn(),
        }),
      { wrapper },
    );

    await waitFor(() => expect(result.current.data[0]?.code).toBe('CNSHG'));
    expect(result.current.total).toBe(250);
    expect(result.current.activeTotal).toBe(180);
    expect(result.current.disabledTotal).toBe(70);

    act(() => {
      result.current.setQuery({ page: 2, pageSize: 10 });
    });

    await waitFor(() => expect(result.current.data[0]?.code).toBe('CNNGB'));
    expect(fetchList).toHaveBeenCalledWith({ page: 2, pageSize: 10 });
  });

  it('外部回调引用不稳定不再触发重取（原 ref 稳定方案的无限重取回归等价断言）', async () => {
    const fetchList = vi.fn(async () => ({ data: [item1], total: 1 }));
    const { wrapper } = createHookWrapper();
    let renderCount = 0;
    const { result, rerender } = renderHook(
      () => {
        renderCount += 1;
        return useMasterDataCrud<Item, Item>({
          entityName: '港口',
          // 每次渲染传入全新的内联闭包：原实现靠 ref 稳定引用防止
          // 「useEffect 依赖变化 → 重取 → setState → 再渲染」死循环；
          // queryKey 模式下回调不参与缓存键，应保持零重取。
          fetchList: () => fetchList(),
          mapItem: (item) => item,
          createItem: async () => ({}),
          updateItem: async () => ({}),
        });
      },
      { wrapper },
    );

    await waitFor(() => expect(result.current.data).toHaveLength(1));
    const callsAfterMount = fetchList.mock.calls.length;
    expect(callsAfterMount).toBeGreaterThan(0);

    rerender();
    rerender();
    await act(async () => {});

    expect(renderCount).toBeGreaterThanOrEqual(3);
    expect(fetchList).toHaveBeenCalledTimes(callsAfterMount);
    expect(result.current.data).toEqual([item1]);
  });

  it('翻页请求期间保留旧页数据（placeholderData 等价原「新页返回前不清空」）', async () => {
    let resolvePage2: (value: { data: Item[]; total: number }) => void =
      () => {};
    const fetchList = vi.fn(async (query: { page: number }) => {
      if (query.page === 2) {
        return new Promise<{ data: Item[]; total: number }>((resolve) => {
          resolvePage2 = resolve;
        });
      }
      return { data: [item1], total: 2 };
    });
    const { wrapper } = createHookWrapper();
    const { result } = renderHook(
      () =>
        useMasterDataCrud<Item, Item>({
          entityName: '港口',
          fetchList,
          mapItem: (item) => item,
          createItem: vi.fn(),
          updateItem: vi.fn(),
        }),
      { wrapper },
    );

    await waitFor(() => expect(result.current.data[0]?.code).toBe('CNSHG'));

    act(() => {
      result.current.setQuery({ page: 2, pageSize: 10 });
    });
    await act(async () => {});

    // 新页未返回前旧页内容与总数保持可见，且处于加载态（原 loading 置位）
    expect(result.current.data[0]?.code).toBe('CNSHG');
    expect(result.current.total).toBe(2);
    expect(result.current.loading).toBe(true);

    act(() => {
      resolvePage2({ data: [item2], total: 2 });
    });

    await waitFor(() => expect(result.current.data[0]?.code).toBe('CNNGB'));
    expect(result.current.loading).toBe(false);
  });

  it('列表加载失败时保留服务端错误明细且统计独立不受影响', async () => {
    const fetchList = vi.fn(
      async (query: { page: number; enabled?: boolean }) => {
        if (query.enabled === true) return { data: [], total: 5 };
        if (query.enabled === false) return { data: [], total: 2 };
        throw new Error('网关超时');
      },
    );
    const { queryClient, wrapper } = createHookWrapper();
    const { result } = renderHook(
      () =>
        useMasterDataCrud<Item, Item>({
          entityName: '港口',
          fetchList,
          mapItem: (item) => item,
          createItem: vi.fn(),
          updateItem: vi.fn(),
        }),
      { wrapper },
    );

    // 动态文案包装惯例：服务端明细透出到 query 状态（全局 onError 消费）
    await waitFor(() => {
      const hasListError = queryClient
        .getQueryCache()
        .findAll({ queryKey: ['master-data', '港口'] })
        .some((query) => query.state.error?.message === '网关超时');
      expect(hasListError).toBe(true);
    });
    // 失败后列表保持空数据、loading 归零（原 catch 后不再 setData）
    expect(result.current.data).toEqual([]);
    expect(result.current.loading).toBe(false);
    // 统计查询独立成功，不受列表失败影响
    expect(result.current.activeTotal).toBe(5);
    expect(result.current.disabledTotal).toBe(2);
  });

  it('启停统计加载失败时以「XX统计加载失败」兜底且列表不受影响', async () => {
    const fetchList = vi.fn(async (query: { enabled?: boolean }) => {
      if (query.enabled !== undefined) {
        // 模拟非 Error 拒绝：兜底文案必须命中 hook 拼接的模板字符串
        return Promise.reject({ status: 500 });
      }
      return { data: [item1], total: 3 };
    });
    const { queryClient, wrapper } = createHookWrapper();
    const { result } = renderHook(
      () =>
        useMasterDataCrud<Item, Item>({
          entityName: '港口',
          fetchList,
          mapItem: (item) => item,
          createItem: vi.fn(),
          updateItem: vi.fn(),
        }),
      { wrapper },
    );

    await waitFor(() => expect(result.current.data).toHaveLength(1));
    expect(result.current.total).toBe(3);
    await waitFor(() => {
      const hasStatsError = queryClient
        .getQueryCache()
        .findAll({ queryKey: ['master-data', '港口', 'stats'] })
        .some((query) => query.state.error?.message === '港口统计加载失败');
      expect(hasStatsError).toBe(true);
    });
    expect(result.current.activeTotal).toBe(0);
    expect(result.current.disabledTotal).toBe(0);
  });
});
