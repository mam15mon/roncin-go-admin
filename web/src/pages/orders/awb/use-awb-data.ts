import { useQuery } from '@tanstack/react-query';
import { useEffect, useMemo, useRef, useState } from 'react';
import { usePageForeground } from '@/hooks/usePageForeground';
import { airExportServiceGetAirMasterWaybill } from '@/services/roncin/airExportService';
import { orderServiceGetOrder } from '@/services/roncin/orderService';
import { generateUUID } from '@/utils/uuid';
import {
  type AwbBundle,
  type AwbIdentity,
  awbFingerprint,
  requireAwbBundle,
} from './model';

export async function fetchAwbBundle(
  identity: AwbIdentity,
  signal?: AbortSignal,
): Promise<AwbBundle> {
  if (identity.orderId) {
    const response = await orderServiceGetOrder(
      { id: identity.orderId },
      { signal, skipErrorHandler: true },
    );
    if (response?.success === false || !response?.data?.airExport?.master)
      throw new Error('航空运单读取失败。');
    const order = response.data;
    if (identity.target === 'MAWB') {
      const master = await fetchAwbBundle(
        {
          ...identity,
          orderId: undefined,
          masterId: response.data.airExport.master.id,
        },
        signal,
      );
      if (master.master.version !== response.data.airExport.master.version)
        throw new Error('主单读取期间发生变动，请重试。');
      return requireAwbBundle({ ...master, order }, identity);
    }
    return requireAwbBundle(
      { order, master: response.data.airExport.master, blockedReasons: [] },
      identity,
    );
  }
  if (!identity.masterId) throw new Error('缺少航空运单身份。');
  const response = await airExportServiceGetAirMasterWaybill(
    { id: identity.masterId, page: 1, pageSize: 1 },
    { signal, skipErrorHandler: true },
  );
  if (response?.success === false || !response?.data)
    throw new Error('航空主单读取失败。');
  return requireAwbBundle(
    { master: response.data, blockedReasons: response.blockedReasons ?? [] },
    identity,
  );
}

/** 每个挂载实例与显式刷新代际独立；探测永远不能成为编辑提交基线。 */
export function useAwbChangeNotice(
  baseline: AwbBundle,
  identity: AwbIdentity,
  session: number,
  enabled: boolean,
) {
  const owner = useMemo(() => generateUUID(), []);
  const base = awbFingerprint(baseline);
  const marker = `${owner}:${session}:${base}`;
  const [changedMarker, setChangedMarker] = useState('');
  const resume = useRef<() => void>(() => undefined);
  const foreground = usePageForeground(() => resume.current());
  const query = useQuery({
    queryKey: ['air-orders', 'awb-check', { owner, identity, session }],
    initialData: baseline,
    staleTime: Infinity,
    enabled: enabled && foreground,
    refetchInterval: enabled && foreground ? 15_000 : false,
    refetchIntervalInBackground: false,
    queryFn: ({ signal }) => fetchAwbBundle(identity, signal),
    meta: { silent: true },
  });
  resume.current = () => {
    if (enabled) void query.refetch({ cancelRefetch: false });
  };
  useEffect(() => {
    if (query.isSuccess && awbFingerprint(query.data) !== base)
      setChangedMarker(marker);
  }, [query.isSuccess, query.data, base, marker]);
  return {
    changed:
      changedMarker === marker ||
      (query.isSuccess && awbFingerprint(query.data) !== base),
    checkError: query.isError,
    async verifyCurrent() {
      const result = await query.refetch();
      if (!result.isSuccess || !result.data)
        throw new Error('航空运单资料检查失败，请重试。');
      if (awbFingerprint(result.data) !== base) {
        setChangedMarker(marker);
        return false;
      }
      return true;
    },
  };
}
