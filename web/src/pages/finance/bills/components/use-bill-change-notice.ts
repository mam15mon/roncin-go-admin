import { useQuery } from '@tanstack/react-query';
import { useRef } from 'react';
import { usePageForeground } from '@/hooks/usePageForeground';
import { settlementServiceGetBill } from '@/services/roncin/settlementService';

export const BILL_CHANGE_POLL_INTERVAL_MS = 15_000;

/**
 * 账单详情外部变动检测：前台每 15 秒读取一次账单版本，切回页面立即检查，后台/隐藏不轮询。
 * 探测结果只用于判断是否变动，不替换展示快照与提交基线（expectedVersion 仍取当前详情）；
 * 本人保存成功后父级以响应更新详情，版本参考点随之前移，较旧的探测不会误报。
 * 检查失败保留上一次成功结果，已出现的提醒不被清除。
 */
export function useBillChangeNotice(options: {
  billId?: string;
  version?: string;
  enabled: boolean;
}) {
  const { billId, version, enabled } = options;
  const resumeRef = useRef<() => void>(() => undefined);
  const foreground = usePageForeground(() => resumeRef.current());
  const active = enabled && Boolean(billId) && Boolean(version);
  const query = useQuery({
    queryKey: ['finance', 'bill-change-probe', { billId }],
    enabled: active && foreground,
    staleTime: Infinity,
    refetchInterval:
      active && foreground ? BILL_CHANGE_POLL_INTERVAL_MS : false,
    refetchIntervalInBackground: false,
    meta: { silent: true },
    queryFn: async ({ signal }) => {
      if (!billId) throw new Error('缺少账单检查身份');
      const response = await settlementServiceGetBill(
        { id: billId },
        { signal, skipErrorHandler: true },
      );
      if (!response?.data?.version || response.success === false)
        throw new Error('账单变动检查失败');
      return response.data.version;
    },
  });
  const { refetch } = query;
  resumeRef.current = () => {
    if (active) void refetch({ cancelRefetch: false });
  };
  const probe = query.data;
  // uint64 版本以字符串传输；只有探测到更新的版本才视为外部变动。
  const changed =
    active &&
    probe !== undefined &&
    version !== undefined &&
    BigInt(probe) > BigInt(version);
  return { changed };
}
