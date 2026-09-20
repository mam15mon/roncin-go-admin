import { useQuery } from '@tanstack/react-query';
import { useCallback } from 'react';
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
  canOperate,
}: Pick<OrderLockSnapshot, 'state' | 'loading' | 'error'> & {
  canOperate: boolean;
}): OrderBusinessWritePolicy {
  if (!canOperate) {
    return { disabled: true, reason: '请切换至订单所属分公司工作台办理' };
  }
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
 * 按订单 ID 加载锁状态。
 *
 * 订单身份进入 queryKey：切换订单即切换到新身份的查询，旧订单的迟到响应
 * 只会写入自己的缓存，天然丢弃；竞态令牌由 React Query 取代。
 */
export function useOrderLockState(orderId?: string) {
  const lockStateQuery = useQuery({
    queryKey: ['orders', 'lock-state', { orderId }],
    enabled: Boolean(orderId),
    // 旧实现对失败静默处理（错误只进入写策略 reason，不弹全局提示）。
    meta: { silent: true },
    queryFn: async (): Promise<API.OrderLockStateData> => {
      if (!orderId) {
        // enabled 已保证订单身份齐备；此处仅为类型收窄兜底。
        throw new Error('缺少订单锁定状态加载参数');
      }
      try {
        const response = await orderLockServiceGetOrderLockState({ orderId });
        const state = response?.data ?? null;
        if (!state) {
          throw new Error('订单锁定状态响应为空');
        }
        return state;
      } catch (error) {
        throw error instanceof Error
          ? error
          : new Error('加载订单锁定状态失败');
      }
    },
  });

  const { data, error, isPending, isFetching, refetch } = lockStateQuery;

  // 失败即不暴露旧数据（与旧实现「刷新失败清空快照」语义一致），写入口
  // 由 getOrderBusinessWritePolicy 依据 loading/error 失败关闭。
  const state = error ? null : (data ?? null);
  const loading = Boolean(orderId) && (isPending || isFetching);

  const refresh = useCallback(() => {
    if (!orderId) {
      return Promise.resolve(null);
    }
    // refetch 的 Promise 只以结果对象 resolve、从不 reject，与旧实现
    // 「refresh 吸收错误并正常返回」的语义一致。
    return refetch().then((result) => result.data ?? null);
  }, [orderId, refetch]);

  return {
    state,
    loading,
    error: error ?? null,
    refresh,
  };
}
