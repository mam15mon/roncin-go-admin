import { act, renderHook, waitFor } from '@testing-library/react';
import React from 'react';
import { App } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  getCachedAirports,
  getCachedPorts,
  getMasterDataOptions,
} from '@/utils/order-options-cache';
import { searchPartnerOptions } from '@/utils/options';
import { type OrderKindConfig, parseOrderKind } from './common';
import { useOrderListResources } from './list-resources';

let mockCurrentUser: any = {
  id: 'user-1',
  currentOrganization: { id: 'org-1', name: '测试组织1' },
};

vi.mock('@umijs/max', () => ({
  useModel: (model: string) => {
    if (model === '@@initialState') {
      return {
        initialState: { currentUser: mockCurrentUser },
      };
    }
    return {};
  },
}));

vi.mock('@/utils/order-options-cache', () => ({
  getMasterDataOptions: vi.fn(),
  getCachedPorts: vi.fn(),
  getCachedAirports: vi.fn(),
}));

vi.mock('@/utils/options', () => ({
  searchPartnerOptions: vi.fn().mockResolvedValue([]),
}));

vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceListPorts: vi.fn().mockResolvedValue({ data: [] }),
}));

vi.mock('@/services/roncin/orderService', () => ({
  orderServiceListPersonnelOptions: vi.fn().mockResolvedValue({ data: [] }),
}));

const mockGetMasterData = vi.mocked(getMasterDataOptions);
const mockGetPorts = vi.mocked(getCachedPorts);
const mockGetAirports = vi.mocked(getCachedAirports);
const mockSearchPartners = vi.mocked(searchPartnerOptions);

const seaConfig = parseOrderKind('sea-export') as OrderKindConfig;

function wrapper({ children }: { children: React.ReactNode }) {
  return React.createElement(App, null, children);
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: any) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

describe('useOrderListResources', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-1', name: '测试组织1' },
    };
    mockGetMasterData.mockResolvedValue([
      { id: 'item-1', name: '上海', code: 'SHA', kind: 1, enabled: true },
    ] as any);
    mockGetPorts.mockResolvedValue([
      { id: 'port-1', nameZh: '洋山', nameEn: 'Yangshan', unLocode: 'CNYAS', enabled: true } as any,
    ]);
    mockGetAirports.mockResolvedValue([
      { id: 'air-1', nameZh: '浦东', nameEn: 'Pudong', iataCode: 'PVG', enabled: true } as any,
    ]);
    mockSearchPartners.mockResolvedValue([
      { label: '阿里巴巴', value: 'cust-1' },
    ]);
  });

  it('缺少当前组织时，不触发主数据请求，候选项置空', async () => {
    mockCurrentUser = { id: 'user-1', currentOrganization: null };

    const { result } = renderHook(() => useOrderListResources(seaConfig), {
      wrapper,
    });

    expect(result.current.masterOptions).toEqual([]);
    expect(result.current.ports).toEqual([]);
    expect(result.current.airports).toEqual([]);
    expect(mockGetMasterData).not.toHaveBeenCalled();
  });

  it('sea 模式按需拉取：仅拉取港口，不拉取机场', async () => {
    const { result } = renderHook(() => useOrderListResources(seaConfig), {
      wrapper,
    });

    await waitFor(() => expect(result.current.ports).toHaveLength(1));
    expect(mockGetMasterData).toHaveBeenCalledWith('org-1');
    expect(mockGetPorts).toHaveBeenCalledWith('org-1');
    expect(mockGetAirports).not.toHaveBeenCalled();
    expect(result.current.airports).toEqual([]);
    expect(result.current.customerMap).toEqual({ 'cust-1': '阿里巴巴' });
  });

  it('air 模式按需拉取：仅拉取机场，不拉取港口', async () => {
    const airConfig: OrderKindConfig = {
      ...seaConfig,
      category: 'air',
    };

    const { result } = renderHook(() => useOrderListResources(airConfig), {
      wrapper,
    });

    await waitFor(() => expect(result.current.airports).toHaveLength(1));
    expect(mockGetMasterData).toHaveBeenCalledWith('org-1');
    expect(mockGetAirports).toHaveBeenCalledWith('org-1');
    expect(mockGetPorts).not.toHaveBeenCalled();
    expect(result.current.ports).toEqual([]);
  });

  it('未指定 category 时同时拉取港口与机场', async () => {
    const { result } = renderHook(() => useOrderListResources(undefined), {
      wrapper,
    });

    await waitFor(() => {
      expect(result.current.ports).toHaveLength(1);
      expect(result.current.airports).toHaveLength(1);
    });
    expect(mockGetPorts).toHaveBeenCalledWith('org-1');
    expect(mockGetAirports).toHaveBeenCalledWith('org-1');
  });

  it('组织切换竞态：组织 A 的迟到响应不得覆盖组织 B 的最新数据', async () => {
    const deferOrgA = deferred<any>();
    const deferOrgB = deferred<any>();

    mockGetMasterData
      .mockImplementationOnce(() => deferOrgA.promise)
      .mockImplementationOnce(() => deferOrgB.promise);

    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-A', name: '组织A' },
    };

    const { result, rerender } = renderHook(
      () => useOrderListResources(seaConfig),
      { wrapper },
    );

    // 切换到组织 B
    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-B', name: '组织B' },
    };
    rerender();

    // 组织 B 先返回
    deferOrgB.resolve([
      { id: 'b-item', name: 'B组织数据', kind: 1, enabled: true },
    ]);
    await waitFor(() =>
      expect(result.current.masterOptions).toEqual([
        expect.objectContaining({ id: 'b-item' }),
      ]),
    );

    // 随后组织 A 迟到的响应到达
    deferOrgA.resolve([
      { id: 'a-item', name: 'A组织数据', kind: 1, enabled: true },
    ]);

    // 确认依然保持 B，A 被成功丢弃
    await waitFor(() =>
      expect(result.current.masterOptions).toEqual([
        expect.objectContaining({ id: 'b-item' }),
      ]),
    );
  });

  it('组织切换时，已落入 React state 的旧组织数据立即清空，绝不在新组织中暴露', async () => {
    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-A', name: '组织A' },
    };
    mockGetMasterData.mockResolvedValueOnce([
      { id: 'a-item', name: 'A选项', kind: 1, enabled: true },
    ] as any);
    mockGetPorts.mockResolvedValueOnce([
      { id: 'port-a', nameEn: 'Port A', enabled: true } as any,
    ]);
    mockSearchPartners.mockResolvedValueOnce([
      { label: '客户A', value: 'cust-a' },
    ]);

    const { result, rerender } = renderHook(
      () => useOrderListResources(seaConfig),
      { wrapper },
    );

    await waitFor(() =>
      expect(result.current.masterOptions).toEqual([
        expect.objectContaining({ id: 'a-item' }),
      ]),
    );
    expect(result.current.ports).toHaveLength(1);
    expect(result.current.customerMap).toEqual({ 'cust-a': '客户A' });

    // 组织 A 已经就绪，此时切换至组织 B（组织 B 尚未完成加载）
    const deferOrgB = deferred<any>();
    mockGetMasterData.mockImplementationOnce(() => deferOrgB.promise);

    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-B', name: '组织B' },
    };
    rerender();

    // 在组织 B 响应前，必须立即置空所有数据，严禁暴露组织 A
    expect(result.current.masterOptions).toEqual([]);
    expect(result.current.ports).toEqual([]);
    expect(result.current.customerMap).toEqual({});

    // 组织 B 响应后正常渲染组织 B 数据
    deferOrgB.resolve([
      { id: 'b-item', name: 'B选项', kind: 1, enabled: true },
    ]);
    await waitFor(() =>
      expect(result.current.masterOptions).toEqual([
        expect.objectContaining({ id: 'b-item' }),
      ]),
    );
  });

  it('在旧组织下触发的搜索若迟到返回，不得写入新组织的 customerMap 或 ports', async () => {
    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-A', name: '组织A' },
    };
    const { result, rerender } = renderHook(
      () => useOrderListResources(seaConfig),
      { wrapper },
    );

    await waitFor(() => expect(result.current.masterOptions).toHaveLength(1));

    // 在组织 A 下触发 searchCustomers
    let resolveCustomer!: (val: any) => void;
    mockSearchPartners.mockImplementationOnce(
      () =>
        new Promise((res) => {
          resolveCustomer = res;
        }),
    );
    const searchPromise = result.current.searchCustomers('慢速客户');

    // 切换到组织 B
    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-B', name: '组织B' },
    };
    rerender();

    // 组织 A 的搜索完成返回
    await act(async () => {
      resolveCustomer([{ label: '旧组织客户', value: 'old-cust' }]);
      await searchPromise;
    });

    // customerMap 不得写入 old-cust
    expect(result.current.customerMap['old-cust']).toBeUndefined();
  });
});
