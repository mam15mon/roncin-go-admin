import { act, renderHook, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { masterDataServiceListPorts } from '@/services/roncin/masterDataService';
import { orderServiceListPersonnelOptions } from '@/services/roncin/orderService';
import { searchPartnerOptions } from '@/utils/options';
import {
  getCachedAirports,
  getCachedPorts,
  getMasterDataOptions,
} from '@/utils/order-options-cache';
import {
  type OrderKindConfig,
  parseOrderKind,
  searchOrderLocations,
} from './common';
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

vi.mock('./common', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./common')>();
  return {
    ...actual,
    searchOrderLocations: vi.fn(),
  };
});

const mockGetMasterData = vi.mocked(getMasterDataOptions);
const mockGetPorts = vi.mocked(getCachedPorts);
const mockGetAirports = vi.mocked(getCachedAirports);
const mockSearchPartners = vi.mocked(searchPartnerOptions);
const mockSearchPorts = vi.mocked(masterDataServiceListPorts);
const mockSearchLocations = vi.mocked(searchOrderLocations);
const mockSearchPersonnel = vi.mocked(orderServiceListPersonnelOptions);

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
      {
        id: 'port-1',
        nameZh: '洋山',
        nameEn: 'Yangshan',
        unLocode: 'CNYAS',
        enabled: true,
      } as any,
    ]);
    mockGetAirports.mockResolvedValue([
      {
        id: 'air-1',
        nameZh: '浦东',
        nameEn: 'Pudong',
        iataCode: 'PVG',
        enabled: true,
      } as any,
    ]);
    mockSearchPartners.mockResolvedValue([
      { label: '阿里巴巴', value: 'cust-1' },
    ]);
    mockSearchPorts.mockResolvedValue({ data: [] });
    mockSearchLocations.mockResolvedValue([]);
    mockSearchPersonnel.mockResolvedValue({ data: [] });
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

    await expect(result.current.searchCustomers('客户')).resolves.toEqual([]);
    await expect(result.current.searchOrderPorts('港口')).resolves.toEqual([]);
    await expect(result.current.searchLocations('地点')).resolves.toEqual([]);
    await expect(result.current.searchOrderCarriers('承运人')).resolves.toEqual(
      [],
    );
    await expect(result.current.searchOrderPersonnel('人员')).resolves.toEqual(
      [],
    );
    expect(mockSearchPartners).not.toHaveBeenCalled();
    expect(mockSearchPorts).not.toHaveBeenCalled();
    expect(mockSearchLocations).not.toHaveBeenCalled();
    expect(mockSearchPersonnel).not.toHaveBeenCalled();
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

  it('组织切换后，五类迟到搜索均向调用方返回空数组且不得写入新组织状态', async () => {
    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-A', name: '组织A' },
    };
    const { result, rerender } = renderHook(
      () => useOrderListResources(seaConfig),
      { wrapper },
    );

    await waitFor(() => expect(result.current.masterOptions).toHaveLength(1));

    const customerSearch = deferred<{ label: string; value: string }[]>();
    const carrierSearch = deferred<{ label: string; value: string }[]>();
    const portSearch = deferred<any>();
    const locationSearch = deferred<{ label: string; value: string }[]>();
    const personnelSearch = deferred<any>();
    mockSearchPartners
      .mockImplementationOnce(() => customerSearch.promise)
      .mockImplementationOnce(() => carrierSearch.promise);
    mockSearchPorts.mockImplementationOnce(() => portSearch.promise);
    mockSearchLocations.mockImplementationOnce(() => locationSearch.promise);
    mockSearchPersonnel.mockImplementationOnce(() => personnelSearch.promise);

    const searchPromises = [
      result.current.searchCustomers('慢速客户'),
      result.current.searchOrderPorts('慢速港口'),
      result.current.searchLocations('慢速地点'),
      result.current.searchOrderCarriers('慢速承运人'),
      result.current.searchOrderPersonnel('慢速人员'),
    ];

    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-B', name: '组织B' },
    };
    rerender();

    let searchResults: unknown[] = [];
    await act(async () => {
      customerSearch.resolve([{ label: '旧组织客户', value: 'old-customer' }]);
      carrierSearch.resolve([{ label: '旧组织承运人', value: 'old-carrier' }]);
      portSearch.resolve({
        data: [
          {
            id: 'old-port',
            nameZh: '旧港口',
            nameEn: 'Old Port',
            unLocode: 'CNOLD',
          },
        ],
      });
      locationSearch.resolve([{ label: '旧组织地点', value: 'old-location' }]);
      personnelSearch.resolve({
        data: [
          {
            userId: 'old-user',
            displayName: '旧组织人员',
            organizationId: 'org-A',
            organizationName: '组织A',
          },
        ],
      });
      searchResults = await Promise.all(searchPromises);
    });

    expect(searchResults).toEqual([[], [], [], [], []]);
    expect(result.current.customerMap['old-customer']).toBeUndefined();
    expect(result.current.ports).not.toEqual(
      expect.arrayContaining([expect.objectContaining({ id: 'old-port' })]),
    );
  });

  it('同组织切换业务配置时，地点和人员搜索拒绝旧类别的迟到结果', async () => {
    let currentConfig = seaConfig;
    const { result, rerender } = renderHook(
      () => useOrderListResources(currentConfig),
      { wrapper },
    );
    await waitFor(() => expect(result.current.masterOptions).toHaveLength(1));

    const locationSearch = deferred<{ label: string; value: string }[]>();
    const personnelSearch = deferred<any>();
    mockSearchLocations.mockImplementationOnce(() => locationSearch.promise);
    mockSearchPersonnel.mockImplementationOnce(() => personnelSearch.promise);
    const locationPromise = result.current.searchLocations('旧海运地点');
    const personnelPromise = result.current.searchOrderPersonnel('旧海运人员');
    expect(mockSearchLocations).toHaveBeenLastCalledWith('sea', '旧海运地点');
    expect(mockSearchPersonnel).toHaveBeenLastCalledWith(
      expect.objectContaining({ businessType: seaConfig.businessType }),
    );

    currentConfig = {
      ...seaConfig,
      category: 'air',
      businessType: seaConfig.businessType + 1,
    };
    rerender();

    locationSearch.resolve([
      { label: '旧海运地点', value: 'old-sea-location' },
    ]);
    personnelSearch.resolve({
      data: [
        {
          userId: 'old-sea-user',
          displayName: '旧海运人员',
          organizationId: 'org-1',
          organizationName: '测试组织1',
        },
      ],
    });

    await expect(locationPromise).resolves.toEqual([]);
    await expect(personnelPromise).resolves.toEqual([]);
  });
});
