import { useQuery } from '@tanstack/react-query';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { orderServiceGetOrder } from '@/services/roncin/orderService';
import { generateUUID } from '@/utils/uuid';

/** 只比较编辑身份与并发标识，不能把查询结果回填成新的编辑基线。 */
function orderChangeFingerprint(order: API.Order): string {
  const master = order.seaMasterBill;
  return JSON.stringify([
    order.id,
    order.version,
    master?.masterBillId,
    master?.version,
    master?.transportExecutionId,
    master?.transportExecutionVersion,
    order.seaDocumentLinkVersion ?? order.seaDocumentSummary?.linkVersion,
    order.seaDocumentStructure ?? order.seaDocumentSummary?.documentStructure,
  ]);
}

interface Options {
  identity?: string;
  scope?: string;
  orderId?: string;
  initialOrder?: API.Order;
  enabled: boolean;
  paused: boolean;
  readEditingOrder: () =>
    | { order: API.Order; masterContentVersion?: string }
    | undefined;
}

interface Observation {
  owner: string;
  generation: number;
  snapshot?: API.Order;
  changed: boolean;
}

export function useOrderChangeNotice(options: Options) {
  const { identity, scope, orderId, initialOrder, enabled, paused } = options;
  // A→B→A 仍创建独立消费范围，旧请求与旧提醒不能回到新实例。
  const owner = useMemo(() => generateUUID(), [identity, scope]);
  const ownerRef = useRef(owner);
  ownerRef.current = owner;
  const [observation, setObservation] = useState<Observation>();
  const current = observation?.owner === owner ? observation : undefined;
  const generation = current?.generation ?? 0;
  const seed = current?.snapshot ?? initialOrder;
  const ready = Boolean(enabled && identity && scope && seed?.id === orderId);
  const canCheck = useRef(ready && !paused);
  canCheck.current = ready && !paused;
  const latest = useRef(options);
  latest.current = options;
  const [foreground, setForeground] = useState(
    () => document.visibilityState === 'visible' && document.hasFocus(),
  );
  const query = useQuery({
    queryKey: [
      'orders',
      'change-notice',
      { identity, scope, orderId, owner, generation, ready, paused },
    ],
    enabled: ready && foreground && !paused,
    initialData: seed,
    // 初次已由完整详情加载，周期和前台恢复才主动重读。
    staleTime: Infinity,
    refetchInterval: ready && foreground && !paused ? 15_000 : false,
    refetchIntervalInBackground: false,
    queryFn: async ({ signal }) => {
      if (!orderId) throw new Error('缺少订单检查身份');
      const response = await orderServiceGetOrder(
        { id: orderId },
        { signal, skipErrorHandler: true },
      );
      if (response?.data?.id !== orderId || response.success === false)
        throw new Error('订单资料检查失败');
      return response.data;
    },
    meta: { silent: true },
  });
  const refetch = useRef(query.refetch);
  refetch.current = query.refetch;

  useEffect(() => {
    let focused = document.hasFocus();
    let active = document.visibilityState === 'visible' && focused;
    setForeground(active);
    const synchronize = () => {
      const next = document.visibilityState === 'visible' && focused;
      if (next === active) return;
      active = next;
      setForeground(next);
      if (next && canCheck.current)
        void refetch.current({ cancelRefetch: false });
    };
    const onFocus = () => {
      focused = true;
      synchronize();
    };
    const onBlur = () => {
      focused = false;
      synchronize();
    };
    const onVisibility = () => {
      focused = document.hasFocus();
      synchronize();
    };
    window.addEventListener('focus', onFocus);
    window.addEventListener('blur', onBlur);
    document.addEventListener('visibilitychange', onVisibility);
    return () => {
      window.removeEventListener('focus', onFocus);
      window.removeEventListener('blur', onBlur);
      document.removeEventListener('visibilitychange', onVisibility);
    };
  }, []);

  useEffect(() => {
    if (!ready || paused || !query.isSuccess || !query.data) return;
    const editing = latest.current.readEditingOrder();
    if (!editing || editing.order.id !== orderId) return;
    if (
      orderChangeFingerprint(editing.order) ===
        orderChangeFingerprint(query.data) &&
      editing.masterContentVersion === query.data.seaMasterBill?.version
    )
      return;
    setObservation((previous) => {
      const state = previous?.owner === owner ? previous : undefined;
      if ((state?.generation ?? 0) !== generation || state?.changed)
        return previous;
      return { owner, generation, snapshot: seed, changed: true };
    });
  }, [
    ready,
    paused,
    query.isSuccess,
    query.data,
    orderId,
    owner,
    generation,
    seed,
  ]);

  const acceptSnapshot = useCallback(
    (snapshot: API.Order) => {
      if (ownerRef.current !== owner) return;
      setObservation((previous) => ({
        owner,
        generation: (previous?.owner === owner ? previous.generation : 0) + 1,
        snapshot,
        changed: false,
      }));
    },
    [owner],
  );
  return { changed: current?.changed === true, acceptSnapshot };
}
