import { renderHook, waitFor } from '@testing-library/react';
import React from 'react';
import { App } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { parseOrderKind } from './common';
import { useOrderDetailData } from './use-order-detail-data';
import { orderServiceGetOrder } from '@/services/roncin/orderService';

let mockCurrentUser: any = {
  id: 'user-1',
  currentOrganization: { id: 'org-1', name: '测试组织' },
};

vi.mock('@umijs/max', () => ({
  useModel: (model: string) => {
    if (model === '@@initialState') {
      return {
        initialState: {
          currentUser: mockCurrentUser,
        },
      };
    }
    return {};
  },
}));

vi.mock('@/utils/order-options-cache', () => ({
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

vi.mock('@/services/roncin/orderContainerService', () => ({
  orderContainerServiceListContainers: vi.fn().mockResolvedValue({ data: [] }),
}));

vi.mock('@/services/roncin/orderCargoItemService', () => ({
  orderCargoItemServiceListCargoItems: vi.fn().mockResolvedValue({ data: [] }),
}));

vi.mock('@/services/roncin/orderMilestoneService', () => ({
  orderMilestoneServiceListMilestones: vi.fn().mockResolvedValue({ data: [] }),
}));

vi.mock('@/services/roncin/orderPersonnelService', () => ({
  orderPersonnelServiceListPersonnel: vi.fn().mockResolvedValue({ data: [] }),
}));

vi.mock('./common', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./common')>();
  return {
    ...actual,
    fetchOrderMasterData: vi.fn().mockResolvedValue({
      serviceTypeOptions: actual.seaServiceTypes.map(({ code, name }) => ({
        code,
        label: name,
        value: code,
      })),
      cargoCategoryOptions: [],
      seaLocationOptions: [],
      airLocationOptions: [],
      currencyOptions: [],
      masterOptions: [],
    }),
  };
});

const mockGetOrder = vi.mocked(orderServiceGetOrder);
const config = parseOrderKind('sea-export');

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

describe('useOrderDetailData', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-1', name: '测试组织' },
    };
  });

  it('成功加载指定订单的数据', async () => {
    mockGetOrder.mockResolvedValue({
      data: { id: 'ord-1', orderNo: 'SE001', version: '1' },
    } as any);

    const { result } = renderHook(
      () => useOrderDetailData('ord-1', config),
      { wrapper },
    );

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

    // 3. 切换至订单 B
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
  });

  it('A 与 B 响应逆序返回时，迟到的 A 响应不得覆盖订单 B 的数据', async () => {
    const deferA = deferred<any>();
    const deferB = deferred<any>();

    mockGetOrder
      .mockImplementationOnce(() => deferA.promise)
      .mockImplementationOnce(() => deferB.promise);

    let currentId = 'ord-A';
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

    // 随后迟到的 A 响应到达
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
    const { result, rerender } = renderHook(
      () => useOrderDetailData('ord-1', config),
      { wrapper },
    );

    expect(result.current.loading).toBe(true);

    // 切换到组织 B
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

    const { result } = renderHook(
      () => useOrderDetailData('ord-1', config),
      { wrapper },
    );

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
});
