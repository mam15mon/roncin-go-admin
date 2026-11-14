import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { usePageForeground } from '@/hooks/usePageForeground';
import { orderLockServiceGetOrderLockState } from '@/services/roncin/orderLockService';

/** 费用外部变动检查参考点：费用内容版本与审核锁状态共同决定费用能否继续按当前输入保存。 */
type FeeChangeMarker = {
  feeContentVersion: string;
  lockGeneration: string;
  isLocked: boolean;
};

function markerOf(state: API.OrderLockStateData): FeeChangeMarker {
  return {
    feeContentVersion: state.feeContentVersion ?? '',
    lockGeneration: state.lockGeneration ?? '',
    // protojson 省略 false：未出现即未锁定。
    isLocked: state.isLocked === true,
  };
}

function sameMarker(a: FeeChangeMarker, b: FeeChangeMarker): boolean {
  return (
    a.feeContentVersion === b.feeContentVersion &&
    a.lockGeneration === b.lockGeneration &&
    a.isLocked === b.isLocked
  );
}

export const FEE_CHANGE_POLL_INTERVAL_MS = 15_000;

/**
 * 费用页外部变动检测：前台每 15 秒读取一次轻量审核锁状态（含费用内容版本），
 * 切回页面立即检查，后台/隐藏不轮询。探测结果只用于判断是否变动，不回填输入，
 * 也不替换提交基线；本人保存、显式刷新后才同步检查参考点。检查失败保留已有提醒。
 */
export function useFeeChangeNotice(options: {
  orderId?: string;
  enabled: boolean;
}) {
  const { orderId, enabled } = options;
  const queryClient = useQueryClient();
  const [baseline, setBaseline] = useState<{
    orderId: string;
    marker: FeeChangeMarker;
  }>();
  const resumeRef = useRef<() => void>(() => undefined);
  const foreground = usePageForeground(() => resumeRef.current());
  const active = enabled && Boolean(orderId);
  const queryKey = useMemo(
    () => ['orders', 'fee-change-probe', { orderId }],
    [orderId],
  );
  const query = useQuery({
    queryKey,
    enabled: active && foreground,
    staleTime: Infinity,
    refetchInterval: active && foreground ? FEE_CHANGE_POLL_INTERVAL_MS : false,
    refetchIntervalInBackground: false,
    meta: { silent: true },
    queryFn: async ({ signal }) => {
      if (!orderId) throw new Error('缺少订单检查身份');
      const response = await orderLockServiceGetOrderLockState(
        { orderId },
        { signal, skipErrorHandler: true },
      );
      if (!response?.data || response.success === false)
        throw new Error('订单费用变动检查失败');
      return markerOf(response.data);
    },
  });
  const { refetch } = query;
  resumeRef.current = () => {
    if (active) void refetch({ cancelRefetch: false });
  };

  const probe = query.data;
  // 首次成功探测建立参考点；订单切换后的旧参考点自然失效。
  useEffect(() => {
    if (!orderId || !probe) return;
    setBaseline((prev) =>
      prev?.orderId === orderId ? prev : { orderId, marker: probe },
    );
  }, [orderId, probe]);

  const current = baseline?.orderId === orderId ? baseline : undefined;
  const changed =
    active &&
    current !== undefined &&
    probe !== undefined &&
    !sameMarker(current.marker, probe);

  /** 显式刷新或合法正式动作后：以最新探测结果重新建立参考点。 */
  const rebaseline = useCallback(async () => {
    if (!orderId) return;
    await queryClient.cancelQueries({ queryKey });
    const result = await refetch();
    if (result.data) {
      setBaseline({ orderId, marker: result.data });
    }
  }, [orderId, queryClient, queryKey, refetch]);

  /** 本人整表保存成功：按成功响应的费用内容版本同步参考点，丢弃更早发出的探测。 */
  const acceptOwnSave = useCallback(
    async (feeContentVersion: string) => {
      if (!orderId) return;
      await queryClient.cancelQueries({ queryKey });
      setBaseline((prev) =>
        prev?.orderId === orderId
          ? { orderId, marker: { ...prev.marker, feeContentVersion } }
          : prev,
      );
      void refetch();
    },
    [orderId, queryClient, queryKey, refetch],
  );

  return { changed, rebaseline, acceptOwnSave };
}
