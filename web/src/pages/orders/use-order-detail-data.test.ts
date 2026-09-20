import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { act, renderHook, waitFor } from '@testing-library/react';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { orderServiceGetOrder } from '@/services/roncin/orderService';
import {
  fetchOrderMasterData,
  searchOrderLocations,
  seaServiceTypes,
} from './common';
import { seaExportDefinition } from './order-kinds/sea-export/definition';
import { useOrderDetailData } from './use-order-detail-data';

let mockCurrentUser: any = {
  id: 'user-1',
  currentOrganization: { id: 'org-1', name: '测试组织' },
};

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: mockCurrentUser,
    },
  }),
}));

vi.mock('@/features/orders/options', () => ({
  getOrderPersonnelOptions: vi.fn().mockResolvedValue([]),
}));

vi.mock('@/services/roncin/orderService', () => ({
  orderServiceGetOrder: vi.fn(),
  orderServiceListPersonnelOptions: vi.fn().mockResolvedValue({ data: [] }),
}));

vi.mock('@/services/roncin/orderShippingDocumentService', () => ({
  orderShippingDocumentServiceListShippingDocuments: vi
    .fn()
    .mockResolvedValue({ data: [] }),
}));

vi.mock('@/services/roncin/orderPersonnelService', () => ({
  orderPersonnelServiceListPersonnel: vi.fn().mockResolvedValue({ data: [] }),
}));

vi.mock('./common', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./common')>();
  return {
    ...actual,
    fetchOrderMasterData: vi.fn(),
    searchOrderLocations: vi.fn(),
  };
});

const mockGetOrder = vi.mocked(orderServiceGetOrder);
const mockFetchMasterData = vi.mocked(fetchOrderMasterData);
const mockSearchLocations = vi.mocked(searchOrderLocations);
const config = seaExportDefinition;

const detailMasterData = {
  serviceTypeOptions: seaServiceTypes.map(({ code, name }) => ({
    code,
    label: name,
    value: code,
  })),
  cargoCategoryOptions: [],
  seaLocationOptions: [{ label: '上海港 (CNSHA)', value: 'port-sha' }],
  airLocationOptions: [{ label: '浦东机场 (PVG)', value: 'airport-pvg' }],
  currencyOptions: [],
  masterOptions: [],
  ports: [],
  airports: [],
  currencies: [],
};

/**
 * React Query 迁移后的 hook 测试包装：每个用例独立 QueryClient，
 * 防止缓存串味（与 renderWithClient 同策略，但以 wrapper 形式供 renderHook 使用）。
 */
function createHookWrapper() {
  const queryClient = createTestQueryClient();
  const wrapper = ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
  return { queryClient, wrapper };
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

describe('useOrderDetailData', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-1', name: '测试组织' },
    };
    mockFetchMasterData.mockResolvedValue(detailMasterData);
    mockSearchLocations.mockResolvedValue([]);
  });

  it('成功加载指定订单的数据', async () => {
    mockGetOrder.mockResolvedValue({
      data: { id: 'ord-1', orderNo: 'SE001', version: '1' },
    } as any);

    const { wrapper } = createHookWrapper();
    const { result } = renderHook(() => useOrderDetailData('ord-1', config), {
      wrapper,
    });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.order?.id).toBe('ord-1');
    expect(result.current.order?.orderNo).toBe('SE001');
    expect(result.current.loadedOrderId).toBe('ord-1');
  });

  it('从订单 A 快速切换到订单 B 时，立即进入 B 加载态且不得渲染 A 的旧数据', async () => {
    const deferA = deferred<any>();
    const deferB = deferred<any>();

    mockGetOrder
      .mockImplementationOnce(() => deferA.promise)
      .mockImplementationOnce(() => deferB.promise);

    let currentId = 'ord-A';
    const { wrapper } = createHookWrapper();
    const { result, rerender } = renderHook(
      () => useOrderDetailData(currentId, config),
      { wrapper },
    );

    // 1. A 正在加载
    expect(result.current.loading).toBe(true);
    expect(result.current.order).toBeUndefined();

    // 2. A 响应成功
    deferA.resolve({
      data: { id: 'ord-A', orderNo: 'ORDER-A', version: '1' },
    });
    await waitFor(() => expect(result.current.order?.id).toBe('ord-A'));
    expect(result.current.loading).toBe(false);

    // 3. 切换至订单 B：queryKey 变化后新键无缓存，order 立即为空
    currentId = 'ord-B';
    rerender();

    // 在 B 返回之前，order 必须立即为空，loading 必须为 true，严禁显示 A
    expect(result.current.order).toBeUndefined();
    expect(result.current.loading).toBe(true);

    // 4. B 返回成功
    deferB.resolve({
      data: { id: 'ord-B', orderNo: 'ORDER-B', version: '1' },
    });
    await waitFor(() => expect(result.current.order?.id).toBe('ord-B'));
    expect(result.current.loading).toBe(false);
  });

  it('切换到订单 B 且 B 请求失败时，保持为空状态，不得回退显示订单 A 的数据', async () => {
    mockGetOrder
      .mockResolvedValueOnce({
        data: { id: 'ord-A', orderNo: 'ORDER-A', version: '1' },
      } as any)
      .mockRejectedValueOnce(new Error('订单 B 不存在或请求超时'));

    let currentId = 'ord-A';
    const { wrapper } = createHookWrapper();
    const { result, rerender } = renderHook(
      () => useOrderDetailData(currentId, config),
      { wrapper },
    );

    await waitFor(() => expect(result.current.order?.id).toBe('ord-A'));

    // 切换至订单 B
    currentId = 'ord-B';
    rerender();

    // B 失败后：loading 结束，order 仍为 undefined，不得复用 A
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.order).toBeUndefined();
    expect(result.current.loadedOrderId).toBeUndefined();
    expect(result.current.error?.message).toBe('订单 B 不存在或请求超时');
  });

  it('A 与 B 响应逆序返回时，迟到的 A 响应不得覆盖订单 B 的数据', async () => {
    const deferA = deferred<any>();
    const deferB = deferred<any>();

    mockGetOrder
      .mockImplementationOnce(() => deferA.promise)
      .mockImplementationOnce(() => deferB.promise);

    let currentId = 'ord-A';
    const { wrapper } = createHookWrapper();
    const { result, rerender } = renderHook(
      () => useOrderDetailData(currentId, config),
      { wrapper },
    );

    // 快速切换至订单 B
    currentId = 'ord-B';
    rerender();

    // B 先返回
    deferB.resolve({
      data: { id: 'ord-B', orderNo: 'ORDER-B', version: '1' },
    });
    await waitFor(() => expect(result.current.order?.id).toBe('ord-B'));

    // 随后迟到的 A 响应到达：A 的结果写入 A 自己的 queryKey 缓存，不影响当前键
    deferA.resolve({
      data: { id: 'ord-A', orderNo: 'ORDER-A', version: '1' },
    });

    // 依然保持为 B，A 的旧响应被成功丢弃
    await waitFor(() => expect(result.current.order?.id).toBe('ord-B'));
    expect(result.current.order?.orderNo).toBe('ORDER-B');
  });

  it('组织切换时，旧组织的延迟响应不得写入当前详情状态', async () => {
    const deferOrgA = deferred<any>();
    const deferOrgB = deferred<any>();

    mockGetOrder
      .mockImplementationOnce(() => deferOrgA.promise)
      .mockImplementationOnce(() => deferOrgB.promise);

    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-A', name: '组织A' },
    };
    const { wrapper } = createHookWrapper();
    const { result, rerender } = renderHook(
      () => useOrderDetailData('ord-1', config),
      { wrapper },
    );

    expect(result.current.loading).toBe(true);

    // 切换到组织 B：organizationId 变化即 queryKey 变化，缓存天然隔离
    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-B', name: '组织B' },
    };
    rerender();

    // 组织 B 先返回
    deferOrgB.resolve({
      data: { id: 'ord-1', orderNo: 'SE-B', version: '1' },
    });
    await waitFor(() => expect(result.current.order?.orderNo).toBe('SE-B'));

    // 随后组织 A 迟到的响应到达
    deferOrgA.resolve({
      data: { id: 'ord-1', orderNo: 'SE-A', version: '1' },
    });

    // 依然保持 B，A 被成功丢弃
    await waitFor(() => expect(result.current.order?.orderNo).toBe('SE-B'));
  });

  it('用户已登录但缺少当前组织时，结束加载并暴露明确业务错误，绝不死锁在 loading 态', async () => {
    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: null,
    };

    const { wrapper } = createHookWrapper();
    const { result } = renderHook(() => useOrderDetailData('ord-1', config), {
      wrapper,
    });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.order).toBeUndefined();
    expect(result.current.error?.message).toBe(
      '缺少当前组织，无法加载订单详情',
    );
    expect(mockGetOrder).not.toHaveBeenCalled();
  });

  it('组织切换时，已落入 React state 的旧组织订单与候选项立即隐藏，绝不在新组织中暴露', async () => {
    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-A', name: '组织A' },
    };
    mockGetOrder.mockResolvedValueOnce({
      data: { id: 'ord-1', orderNo: 'ORDER-A-001', version: '1' },
    } as any);

    const { wrapper } = createHookWrapper();
    const { result, rerender } = renderHook(
      () => useOrderDetailData('ord-1', config),
      { wrapper },
    );

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.order?.orderNo).toBe('ORDER-A-001');

    // 组织 A 已经就绪，此时切换至组织 B（组织 B 尚未完成加载）
    const deferOrgB = deferred<any>();
    mockGetOrder.mockImplementationOnce(() => deferOrgB.promise);

    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-B', name: '组织B' },
    };
    rerender();

    // 在组织 B 响应前，必须立即进入 loading 态，且 order 与 options 必须隐藏，严防组织 A 数据闪现
    expect(result.current.loading).toBe(true);
    expect(result.current.order).toBeUndefined();
    expect(result.current.serviceTypeOptions).toEqual([]);

    // 组织 B 响应后，正常展示组织 B 数据
    deferOrgB.resolve({
      data: { id: 'ord-1', orderNo: 'ORDER-B-002', version: '1' },
    });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.order?.orderNo).toBe('ORDER-B-002');
  });

  it('地点搜索为空或仅含空白时复用当前详情首批候选项，不发起远程请求', async () => {
    mockGetOrder.mockResolvedValue({
      data: { id: 'ord-1', orderNo: 'SE001', version: '1' },
    } as any);
    const { wrapper } = createHookWrapper();
    const { result } = renderHook(() => useOrderDetailData('ord-1', config), {
      wrapper,
    });

    await waitFor(() => expect(result.current.loading).toBe(false));

    await expect(result.current.searchLocations('')).resolves.toEqual(
      detailMasterData.seaLocationOptions,
    );
    await expect(result.current.searchLocations('  ')).resolves.toEqual(
      detailMasterData.seaLocationOptions,
    );
    expect(mockSearchLocations).not.toHaveBeenCalled();
  });

  it('地点搜索有关键字时继续远程查询，组织切换后的迟到结果返回空数组', async () => {
    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-A', name: '组织A' },
    };
    mockGetOrder.mockResolvedValue({
      data: { id: 'ord-1', orderNo: 'SE001', version: '1' },
    } as any);
    const { wrapper } = createHookWrapper();
    const { result, rerender } = renderHook(
      () => useOrderDetailData('ord-1', config),
      { wrapper },
    );
    await waitFor(() => expect(result.current.loading).toBe(false));

    const delayedSearch = deferred<{ label: string; value: string }[]>();
    mockSearchLocations.mockImplementationOnce(() => delayedSearch.promise);
    const searchPromise = result.current.searchLocations(' shang ');
    expect(mockSearchLocations).toHaveBeenCalledWith('sea', ' shang ');

    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-B', name: '组织B' },
    };
    rerender();
    // 组织切换触发的详情重查在 act 内落地
    await act(async () => {});

    delayedSearch.resolve([{ label: '旧组织港口', value: 'old-port' }]);
    // 迟到搜索结果回调在 act 内收敛，避免用例结束后迟到更新
    await act(async () => {
      await searchPromise;
    });

    await expect(searchPromise).resolves.toEqual([]);
  });

  it('组织 A 详情加载失败后切到 B，B 首次渲染立即隐藏 A 的错误并进入加载态', async () => {
    const orgBRequest = deferred<any>();
    mockGetOrder
      .mockRejectedValueOnce(new Error('组织 A 详情失败'))
      .mockImplementationOnce(() => orgBRequest.promise);
    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-A', name: '组织A' },
    };
    const renderSnapshots: Array<{
      organizationId?: string;
      loading: boolean;
      error: Error | null;
    }> = [];
    const { wrapper } = createHookWrapper();
    const { result, rerender } = renderHook(
      () => {
        const state = useOrderDetailData('ord-1', config);
        renderSnapshots.push({
          organizationId: mockCurrentUser.currentOrganization?.id,
          loading: state.loading,
          error: state.error,
        });
        return state;
      },
      { wrapper },
    );

    await waitFor(() =>
      expect(result.current.error?.message).toBe('组织 A 详情失败'),
    );

    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-B', name: '组织B' },
    };
    rerender();

    const firstOrgBSnapshot = renderSnapshots.find(
      (snapshot) => snapshot.organizationId === 'org-B',
    );
    expect(firstOrgBSnapshot).toEqual(
      expect.objectContaining({ loading: true, error: null }),
    );

    orgBRequest.resolve({
      data: { id: 'ord-1', orderNo: 'SE-B', version: '1' },
    });
    await waitFor(() => expect(result.current.loading).toBe(false));
  });

  it('同组织同订单切换业务配置时，首次渲染立即隐藏旧配置错误并拒绝迟到搜索', async () => {
    mockGetOrder.mockRejectedValueOnce(new Error('旧业务详情失败'));
    let currentConfig = config;
    const { wrapper } = createHookWrapper();
    const { result, rerender } = renderHook(
      () => useOrderDetailData('ord-1', currentConfig),
      { wrapper },
    );
    await waitFor(() =>
      expect(result.current.error?.message).toBe('旧业务详情失败'),
    );

    const delayedSearch = deferred<{ label: string; value: string }[]>();
    mockSearchLocations.mockImplementationOnce(() => delayedSearch.promise);
    const searchPromise = result.current.searchLocations('旧业务地点');

    const nextOrderRequest = deferred<any>();
    mockGetOrder.mockImplementationOnce(() => nextOrderRequest.promise);
    currentConfig = {
      ...config,
      transportMode: 'air',
      businessType: (config.businessType + 1) as typeof config.businessType,
    };
    rerender();

    expect(result.current.error).toBeNull();
    expect(result.current.loading).toBe(true);
    await expect(result.current.searchLocations('')).resolves.toEqual([]);

    delayedSearch.resolve([{ label: '旧业务地点', value: 'old-location' }]);
    await expect(searchPromise).resolves.toEqual([]);

    nextOrderRequest.resolve({
      data: { id: 'ord-1', orderNo: 'AIR-001', version: '1' },
    });
    await waitFor(() => expect(result.current.loading).toBe(false));
  });
});
