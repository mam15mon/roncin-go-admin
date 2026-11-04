import { useQuery } from '@tanstack/react-query';
import { useEffect, useRef, useState } from 'react';
import { orderServiceGetOrder } from '@/services/roncin/orderService';
import { seaDocumentServiceGetSeaOrderDocuments } from '@/services/roncin/seaDocumentService';
import { type HblBundle, hblBundleFingerprint } from './model';

export async function fetchHblBundle(
  orderId: string,
  signal?: AbortSignal,
): Promise<HblBundle> {
  const [order, documents] = await Promise.all([
    orderServiceGetOrder({ id: orderId }, { signal, skipErrorHandler: true }),
    seaDocumentServiceGetSeaOrderDocuments(
      { orderId },
      { signal, skipErrorHandler: true },
    ),
  ]);
  if (
    order?.success === false ||
    documents?.success === false ||
    order?.data?.id !== orderId ||
    documents?.data?.orderId !== orderId
  )
    throw new Error('HBL 制作资料读取失败，请重试。');
  return { order: order.data, documents: documents.data };
}

/** 轮询只比较版本标识，查询结果不能回填为用户编辑的新基线。 */
export function useHblChangeNotice(
  baseline: HblBundle,
  scope: string,
  paused: boolean,
) {
  const base = hblBundleFingerprint(baseline);
  const [changedBase, setChangedBase] = useState<string>();
  const [foreground, setForeground] = useState(
    () => document.visibilityState === 'visible' && document.hasFocus(),
  );
  const query = useQuery({
    queryKey: ['orders', 'hbl-change-notice', { scope, base }],
    queryFn: ({ signal }) =>
      fetchHblBundle(baseline.order.id as string, signal),
    initialData: baseline,
    staleTime: Infinity,
    enabled: foreground && !paused,
    refetchInterval: foreground && !paused ? 15_000 : false,
    refetchIntervalInBackground: false,
    meta: { silent: true },
  });
  const refetch = useRef(query.refetch);
  refetch.current = query.refetch;
  const pausedRef = useRef(paused);
  pausedRef.current = paused;
  useEffect(() => {
    let focused = document.hasFocus();
    let active = focused && document.visibilityState === 'visible';
    const synchronize = () => {
      const next = focused && document.visibilityState === 'visible';
      setForeground(next);
      if (next && !active && !pausedRef.current) void refetch.current();
      active = next;
    };
    const focus = () => {
      focused = true;
      synchronize();
    };
    const blur = () => {
      focused = false;
      synchronize();
    };
    const visibility = () => {
      focused = document.hasFocus();
      synchronize();
    };
    window.addEventListener('focus', focus);
    window.addEventListener('blur', blur);
    document.addEventListener('visibilitychange', visibility);
    return () => {
      window.removeEventListener('focus', focus);
      window.removeEventListener('blur', blur);
      document.removeEventListener('visibilitychange', visibility);
    };
  }, []);
  useEffect(() => {
    if (query.data && hblBundleFingerprint(query.data) !== base)
      setChangedBase(base);
  }, [query.data, base]);
  return {
    changed: changedBase === base,
    async verifyCurrent() {
      const result = await query.refetch();
      if (!result.data || result.error)
        throw new Error('HBL 资料检查失败，请重试。');
      if (hblBundleFingerprint(result.data) !== base) {
        setChangedBase(base);
        return false;
      }
      return true;
    },
  };
}
