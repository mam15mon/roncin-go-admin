import { act, renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { OrderBusinessType } from '@/enums.generated';
import { orderLockServiceGetOrderLockState } from '@/services/roncin/orderLockService';
import {
  getOrderBusinessWritePolicy,
  useOrderLockState,
} from './use-order-lock-state';

vi.mock('@/services/roncin/orderLockService', () => ({
  orderLockServiceGetOrderLockState: vi.fn(),
}));

const getLockState = vi.mocked(orderLockServiceGetOrderLockState);

function response(state: API.OrderLockStateData) {
  return { data: state } as Awaited<ReturnType<typeof getLockState>>;
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((promiseResolve) => {
    resolve = promiseResolve;
  });
  return { promise, resolve };
}

describe('useOrderLockState', () => {
  beforeEach(() => {
    getLockState.mockReset();
  });

  it('成功加载并返回服务端锁状态', async () => {
    getLockState.mockResolvedValue(
      response({
        orderId: 'order-a',
        businessType: OrderBusinessType.BUSINESS_TYPE_SE,
        isLocked: false,
        orderVersion: '3',
      }),
    );

    const { result } = renderHook(() => useOrderLockState('order-a'));

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.error).toBeNull();
    expect(result.current.state?.orderId).toBe('order-a');
    expect(result.current.state?.orderVersion).toBe('3');
  });

  it('失败时清空已有状态，并可通过 refresh 重试', async () => {
    getLockState
      .mockResolvedValueOnce(
        response({ orderId: 'order-a', isLocked: false, orderVersion: '1' }),
      )
      .mockRejectedValueOnce(new Error('网络错误'))
      .mockResolvedValueOnce(
        response({ orderId: 'order-a', isLocked: true, orderVersion: '2' }),
      );

    const { result } = renderHook(() => useOrderLockState('order-a'));
    await waitFor(() => expect(result.current.state?.orderVersion).toBe('1'));

    await act(async () => {
      await result.current.refresh();
    });
    expect(result.current.state).toBeNull();
    expect(result.current.error?.message).toBe('网络错误');

    await act(async () => {
      await result.current.refresh();
    });
    expect(result.current.error).toBeNull();
    expect(result.current.state?.isLocked).toBe(true);
  });

  it('切换订单后立即清空旧状态并丢弃迟到响应', async () => {
    const first = deferred<Awaited<ReturnType<typeof getLockState>>>();
    const second = deferred<Awaited<ReturnType<typeof getLockState>>>();
    getLockState
      .mockReturnValueOnce(first.promise)
      .mockReturnValueOnce(second.promise);

    const { result, rerender } = renderHook(
      ({ orderId }) => useOrderLockState(orderId),
      { initialProps: { orderId: 'order-a' } },
    );
    rerender({ orderId: 'order-b' });
    expect(result.current.state).toBeNull();
    expect(result.current.loading).toBe(true);

    await act(async () => {
      second.resolve(
        response({ orderId: 'order-b', isLocked: false, orderVersion: '8' }),
      );
      await second.promise;
    });
    expect(result.current.state?.orderId).toBe('order-b');

    await act(async () => {
      first.resolve(
        response({ orderId: 'order-a', isLocked: true, orderVersion: '2' }),
      );
      await first.promise;
    });
    expect(result.current.state?.orderId).toBe('order-b');
    expect(result.current.state?.orderVersion).toBe('8');
  });
});

describe('getOrderBusinessWritePolicy', () => {
  it('加载、错误、空状态和已锁定时都失败关闭', () => {
    expect(
      getOrderBusinessWritePolicy({ state: null, loading: true, error: null })
        .disabled,
    ).toBe(true);
    expect(
      getOrderBusinessWritePolicy({
        state: null,
        loading: false,
        error: new Error('failed'),
      }).disabled,
    ).toBe(true);
    expect(
      getOrderBusinessWritePolicy({ state: null, loading: false, error: null })
        .disabled,
    ).toBe(true);
    expect(
      getOrderBusinessWritePolicy({
        state: {
          isLocked: true,
          businessType: OrderBusinessType.BUSINESS_TYPE_AI,
        },
        loading: false,
        error: null,
      }),
    ).toEqual({
      disabled: true,
      reason: '空运进口订单已锁定，如需修改请先解锁',
    });
  });

  it('只有成功加载且未锁定时开放业务写', () => {
    expect(
      getOrderBusinessWritePolicy({
        state: { isLocked: false },
        loading: false,
        error: null,
      }),
    ).toEqual({ disabled: false });
  });
});
