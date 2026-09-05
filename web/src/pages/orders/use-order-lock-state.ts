import { useCallback, useEffect, useRef, useState } from 'react';
import { businessTypeMeta } from '@/constants/statusMeta';
import { orderLockServiceGetOrderLockState } from '@/services/roncin/orderLockService';

type OrderLockSnapshot = {
  orderId?: string;
  state: API.OrderLockStateData | null;
  loading: boolean;
  error: Error | null;
};

export type OrderBusinessWritePolicy = {
  disabled: boolean;
  reason?: string;
};

function normalizeError(error: unknown): Error {
  if (error instanceof Error) return error;
  return new Error('加载订单锁定状态失败');
}

export function getOrderBusinessTypeLabel(businessType?: number): string {
  if (businessType === undefined) return '订单';
  return businessTypeMeta[businessType]?.text ?? '订单';
}

/**
 * 订单业务写入口的统一失败关闭策略。
 *
 * 锁状态加载、失败、缺失及同步刷新期间都不能开放写入口；服务端仍是最终门禁。
 */
export function getOrderBusinessWritePolicy({
  state,
  loading,
  error,
}: Pick<
  OrderLockSnapshot,
  'state' | 'loading' | 'error'
>): OrderBusinessWritePolicy {
  if (loading) {
    return { disabled: true, reason: '正在同步订单锁定状态，请稍候' };
  }
  if (error) {
    return { disabled: true, reason: '订单锁定状态加载失败，请重试' };
  }
  if (!state) {
    return { disabled: true, reason: '订单锁定状态尚未加载，请重试' };
  }
  if (state.isLocked) {
    return {
      disabled: true,
      reason: `${getOrderBusinessTypeLabel(state.businessType)}订单已锁定，如需修改请先解锁`,
    };
  }
  return { disabled: false };
}

/**
 * 按订单 ID 加载锁状态。快照始终绑定到请求时的订单 ID，并用请求序号丢弃迟到响应。
 */
export function useOrderLockState(orderId?: string) {
  const requestSequenceRef = useRef(0);
  const [snapshot, setSnapshot] = useState<OrderLockSnapshot>({
    state: null,
    loading: Boolean(orderId),
    error: null,
  });

  const load = useCallback(async (targetOrderId?: string) => {
    const requestSequence = ++requestSequenceRef.current;
    if (!targetOrderId) {
      setSnapshot({ state: null, loading: false, error: null });
      return null;
    }

    setSnapshot({
      orderId: targetOrderId,
      state: null,
      loading: true,
      error: null,
    });
    try {
      const response = await orderLockServiceGetOrderLockState({
        orderId: targetOrderId,
      });
      if (requestSequence !== requestSequenceRef.current) return null;
      const state = response?.data ?? null;
      if (!state) {
        const error = new Error('订单锁定状态响应为空');
        setSnapshot({
          orderId: targetOrderId,
          state: null,
          loading: false,
          error,
        });
        return null;
      }
      setSnapshot({
        orderId: targetOrderId,
        state,
        loading: false,
        error: null,
      });
      return state;
    } catch (error: unknown) {
      if (requestSequence !== requestSequenceRef.current) return null;
      setSnapshot({
        orderId: targetOrderId,
        state: null,
        loading: false,
        error: normalizeError(error),
      });
      return null;
    }
  }, []);

  useEffect(() => {
    void load(orderId);
    return () => {
      requestSequenceRef.current += 1;
    };
  }, [load, orderId]);

  const refresh = useCallback(() => load(orderId), [load, orderId]);
  const belongsToCurrentOrder = snapshot.orderId === orderId;

  return {
    state: belongsToCurrentOrder ? snapshot.state : null,
    loading: Boolean(orderId) && (!belongsToCurrentOrder || snapshot.loading),
    error: belongsToCurrentOrder ? snapshot.error : null,
    refresh,
  };
}
