import { useQuery } from '@tanstack/react-query';
import { useEffect, useMemo, useRef, useState } from 'react';
import { usePageForeground } from '@/hooks/usePageForeground';
import { airExportServiceGetAirMasterWaybill } from '@/services/roncin/airExportService';
import { generateUUID } from '@/utils/uuid';
import { requireAirMasterIdentity } from './order-kinds/air-export/identity';

function markerOf(response?: API.GetAirMasterWaybillResponse) {
  return JSON.stringify([
    response?.data?.version,
    [...(response?.blockedReasons ?? [])].sort(),
  ]);
}

/**
 * 仅检查主单版本与共享修改门禁；探测永远不替换表单与提交版本。
 *
 * 本人写入在途（ownWritePending，含写后重读）时暂停探测且不记录新差异：此时读到的是
 * 本人写后的版本，不能当作外部变动；写入前已记录的提醒保持不变。写入成功后编辑弹窗
 * 关闭（会话结束），调用方保持 pending 直至关闭；失败时结束 pending 恢复检查。
 */
export function useAirMasterChangeNotice(options: {
  baseline?: API.GetAirMasterWaybillResponse;
  session: number;
  organizationId?: string;
  businessType?: number;
  enabled: boolean;
  ownWritePending?: boolean;
}) {
  const {
    baseline,
    session,
    organizationId,
    businessType,
    enabled,
    ownWritePending = false,
  } = options;
  const masterId = baseline?.data?.id;
  const owner = useMemo(
    () => generateUUID(),
    [organizationId, businessType, masterId],
  );
  const identity = `${owner}:${session}`;
  const [changedIdentity, setChangedIdentity] = useState<string>();
  const resumeRef = useRef<() => void>(() => undefined);
  const foreground = usePageForeground(() => resumeRef.current());
  const active = enabled && Boolean(masterId && organizationId);
  // 暂停时观察者失活：轮询停止，写后 invalidateQueries(['air-orders']) 也不会重查。
  const polling = active && foreground && !ownWritePending;
  const query = useQuery({
    queryKey: [
      'air-orders',
      'master-change-notice',
      { masterId, organizationId, businessType, owner, session },
    ],
    enabled: polling,
    initialData: baseline,
    staleTime: Infinity,
    refetchInterval: polling ? 15_000 : false,
    refetchIntervalInBackground: false,
    meta: { silent: true },
    queryFn: async ({ signal }) => {
      if (!masterId) throw new Error('缺少主单检查身份');
      const response = await airExportServiceGetAirMasterWaybill(
        { id: masterId, page: 1, pageSize: 1 },
        { signal, skipErrorHandler: true },
      );
      if (response?.success === false || !response?.data)
        throw new Error('主单变动检查失败');
      requireAirMasterIdentity(
        response.data,
        masterId,
        organizationId,
        businessType,
      );
      return response;
    },
  });
  resumeRef.current = () => {
    // 回前台回调早于 foreground 状态渲染，不能依赖 polling。
    if (active && !ownWritePending)
      void query.refetch({ cancelRefetch: false });
  };
  const base = markerOf(baseline);
  const observed = markerOf(query.data);
  useEffect(() => {
    if (active && !ownWritePending && query.isSuccess && observed !== base)
      setChangedIdentity(identity);
  }, [active, ownWritePending, query.isSuccess, observed, base, identity]);
  return active && changedIdentity === identity;
}
