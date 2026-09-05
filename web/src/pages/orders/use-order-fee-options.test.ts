import { renderHook, waitFor } from '@testing-library/react';
import React from 'react';
import { App } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useOrderFeeOptions } from './use-order-fee-options';
import { orderServiceGetOrder } from '@/services/roncin/orderService';
import { orderFeeServiceListFeeOptions } from '@/services/roncin/orderFeeService';

vi.mock('@/services/roncin/orderService', () => ({
  orderServiceGetOrder: vi.fn(),
}));

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceListFeeOptions: vi.fn(),
}));

const mockGetOrder = vi.mocked(orderServiceGetOrder);
const mockListFeeOptions = vi.mocked(orderFeeServiceListFeeOptions);

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

describe('useOrderFeeOptions', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockListFeeOptions.mockResolvedValue({
      currencies: [{ label: 'CNY', value: 'CNY' }],
      settlementParties: [{ label: '客户A', value: 'part-1' }],
      customerName: '某客户',
      feeSettings: [],
      billingUnits: [],
      financeLocked: false,
    } as any);
  });

  it('成功加载指定订单的费用选项', async () => {
    mockGetOrder.mockResolvedValue({
      data: { id: 'ord-1', orderNo: 'SE001', version: '1' },
    } as any);

    const { result } = renderHook(() => useOrderFeeOptions('ord-1'), { wrapper });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.order?.id).toBe('ord-1');
    expect(result.current.loadedOrderId).toBe('ord-1');
    expect(result.current.customerName).toBe('某客户');
  });

  it('从订单 A 快速切换到订单 B 时，立即进入 B 加载态且不得渲染 A 的旧数据', async () => {
    const deferA = deferred<any>();
    const deferB = deferred<any>();

    mockGetOrder
      .mockImplementationOnce(() => deferA.promise)
      .mockImplementationOnce(() => deferB.promise);

    let currentId = 'ord-A';
    const { result, rerender } = renderHook(() => useOrderFeeOptions(currentId), { wrapper });

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

    // 在 B 返回前，order 必须立即为空，loading 必须为 true，严禁显示 A
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
      .mockRejectedValueOnce(new Error('订单 B 加载失败'));

    let currentId = 'ord-A';
    const { result, rerender } = renderHook(() => useOrderFeeOptions(currentId), { wrapper });

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
    const { result, rerender } = renderHook(() => useOrderFeeOptions(currentId), { wrapper });

    // 切换至订单 B
    currentId = 'ord-B';
    rerender();

    // B 先返回
    deferB.resolve({
      data: { id: 'ord-B', orderNo: 'ORDER-B', version: '1' },
    });
    await waitFor(() => expect(result.current.order?.id).toBe('ord-B'));

    // 迟到的 A 响应到达
    deferA.resolve({
      data: { id: 'ord-A', orderNo: 'ORDER-A', version: '1' },
    });

    // 依然保持为 B，A 响应被正确丢弃
    await waitFor(() => expect(result.current.order?.id).toBe('ord-B'));
    expect(result.current.order?.orderNo).toBe('ORDER-B');
  });
});
