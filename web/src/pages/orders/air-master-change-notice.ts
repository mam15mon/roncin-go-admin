import { useQuery } from '@tanstack/react-query';
import { useEffect, useRef, useState } from 'react';
import { usePageForeground } from '@/hooks/usePageForeground';
import { airExportServiceGetAirMasterWaybill } from '@/services/roncin/airExportService';

function markerOf(response?: API.GetAirMasterWaybillResponse) {
  return JSON.stringify([
    response?.data?.version,
    [...(response?.blockedReasons ?? [])].sort(),
  ]);
}

/** 仅检查主单版本与共享修改门禁；探测永远不替换表单与提交版本。 */
export function useAirMasterChangeNotice(options: {
  baseline?: API.GetAirMasterWaybillResponse;
  session: number;
  organizationId?: string;
  enabled: boolean;
}) {
  const { baseline, session, organizationId, enabled } = options;
  const masterId = baseline?.data?.id;
  const [changedSession, setChangedSession] = useState<number>();
  const resumeRef = useRef<() => void>(() => undefined);
  const foreground = usePageForeground(() => resumeRef.current());
  const active = enabled && Boolean(masterId && organizationId);
  const query = useQuery({
    queryKey: [
      'air-export',
      'master-change-notice',
      { masterId, organizationId, session },
    ],
    enabled: active && foreground,
    initialData: baseline,
    staleTime: Infinity,
    refetchInterval: active && foreground ? 15_000 : false,
    refetchIntervalInBackground: false,
    meta: { silent: true },
    queryFn: async ({ signal }) => {
      if (!masterId) throw new Error('缺少主单检查身份');
      const response = await airExportServiceGetAirMasterWaybill(
        { id: masterId, page: 1, pageSize: 1 },
        { signal, skipErrorHandler: true },
      );
      if (
        response?.success === false ||
        response?.data?.id !== masterId ||
        !response.data.version
      )
        throw new Error('主单变动检查失败');
      return response;
    },
  });
  resumeRef.current = () => {
    if (active) void query.refetch({ cancelRefetch: false });
  };
  const base = markerOf(baseline);
  const observed = markerOf(query.data);
  useEffect(() => {
    if (active && query.isSuccess && observed !== base)
      setChangedSession(session);
  }, [active, query.isSuccess, observed, base, session]);
  return active && changedSession === session;
}
