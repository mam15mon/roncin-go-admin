import { act, renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { useMasterDataCrud } from './useMasterDataCrud';

const messageError = vi.hoisted(() => vi.fn());

vi.mock('antd', () => ({
  App: { useApp: () => ({ message: { error: messageError } }) },
}));

type Item = {
  id: string;
  code: string;
  name: string;
  enabled: boolean;
};

describe('useMasterDataCrud', () => {
  it('按服务端分页加载数据并读取启停统计', async () => {
    const fetchList = vi.fn(async (query: { page: number; pageSize: number; enabled?: boolean }) => {
      if (query.enabled === true) return { data: [], total: 180 };
      if (query.enabled === false) return { data: [], total: 70 };
      const item: Item = {
        id: `item-${query.page}`,
        code: query.page === 1 ? 'CNSHG' : 'CNNGB',
        name: query.page === 1 ? '上海港' : '宁波港',
        enabled: true,
      };
      return { data: [item], total: 250 };
    });
    const { result } = renderHook(() =>
      useMasterDataCrud<Item, Item>({
        entityName: '港口',
        fetchList,
        mapItem: (item) => item,
        createItem: vi.fn(),
        updateItem: vi.fn(),
      }),
    );

    await waitFor(() => expect(result.current.data[0]?.code).toBe('CNSHG'));
    expect(result.current.total).toBe(250);
    expect(result.current.activeTotal).toBe(180);
    expect(result.current.disabledTotal).toBe(70);

    act(() => result.current.setQuery({ page: 2, pageSize: 10 }));

    await waitFor(() => expect(result.current.data[0]?.code).toBe('CNNGB'));
    expect(fetchList).toHaveBeenCalledWith({ page: 2, pageSize: 10 });
  });
});
