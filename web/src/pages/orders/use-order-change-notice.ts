import { useQuery } from '@tanstack/react-query';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  OrderBusinessType,
  OrderTerminationStatus,
  SeaDocumentStructure,
} from '@/enums.generated';
import { usePageForeground } from '@/hooks/usePageForeground';
import { orderServiceGetOrder } from '@/services/roncin/orderService';
import { generateUUID } from '@/utils/uuid';

/** 只比较编辑身份与并发标识，不能把查询结果回填成新的编辑基线。 */
function orderChangeFingerprint(order: API.Order, terminated = false): string {
  // 终态已无活动关系；表单中的保留单证不再代表当前运输执行身份。
  if (terminated) return JSON.stringify([order.id, order.version]);
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

/** 仅用恢复前已经持有的保留单证辨识本次原关联恢复；原航程身份由服务端锁内验证。 */
function matchesRestoredSeaRelation(
  previous: API.Order,
  restored: API.Order,
  retainedDocuments?: API.SeaOrderDocuments,
): boolean {
  const master = retainedDocuments?.masterBill;
  const linkVersion =
    restored.seaDocumentLinkVersion ?? restored.seaDocumentSummary?.linkVersion;
  const structure =
    restored.seaDocumentStructure ??
    restored.seaDocumentSummary?.documentStructure;
  if (
    previous.businessType !== OrderBusinessType.BUSINESS_TYPE_SE ||
    previous.terminationStatus !==
      OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED ||
    restored.terminationStatus !==
      OrderTerminationStatus.ORDER_TERMINATION_STATUS_ACTIVE ||
    !master?.id ||
    !master.version ||
    restored.seaMasterBill?.masterBillId !== master.id ||
    restored.seaMasterBill.version !== master.version ||
    !restored.seaMasterBill.transportExecutionId ||
    !restored.seaMasterBill.transportExecutionVersion ||
    linkVersion !== '1' ||
    (retainedDocuments?.documentStructure !==
      SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT &&
      retainedDocuments?.documentStructure !==
        SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE) ||
    structure !== retainedDocuments.documentStructure
  )
    return false;
  return (
    [
      'shippingLineId',
      'originLocationId',
      'dischargeLocationId',
      'transitLocationId',
      'vesselVoyage',
      'etd',
      'eta',
    ] as const
  ).every((key) => (previous[key] ?? '') === (restored[key] ?? ''));
}

interface Options {
  identity?: string;
  scope?: string;
  orderId?: string;
  initialOrder?: API.Order;
  /** 当前编辑身份已加载的保留单证，不使用恢复后查询替换它来识别本人操作。 */
  retainedDocuments?: API.SeaOrderDocuments;
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
  transitionSnapshot?: API.Order;
  changedSnapshot?: API.Order;
  hasOtherChange?: boolean;
}

export function useOrderChangeNotice(options: Options) {
  const {
    identity,
    scope,
    orderId,
    initialOrder,
    retainedDocuments,
    enabled,
    paused,
  } = options;
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
  const resumeRef = useRef<() => void>(() => undefined);
  const foreground = usePageForeground(() => resumeRef.current());
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
  resumeRef.current = () => {
    if (canCheck.current) void query.refetch({ cancelRefetch: false });
  };

  useEffect(() => {
    if (!ready || paused || !query.isSuccess || !query.data) return;
    const editing = current?.transitionSnapshot
      ? {
          order: current.transitionSnapshot,
          masterContentVersion:
            current.transitionSnapshot.seaMasterBill?.version,
        }
      : latest.current.readEditingOrder();
    if (!editing || editing.order.id !== orderId) return;
    const terminated =
      query.data.terminationStatus ===
      OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED;
    if (
      orderChangeFingerprint(editing.order, terminated) ===
        orderChangeFingerprint(query.data, terminated) &&
      (terminated ||
        editing.masterContentVersion === query.data.seaMasterBill?.version)
    )
      return;
    setObservation((previous) => {
      const state = previous?.owner === owner ? previous : undefined;
      if ((state?.generation ?? 0) !== generation) return previous;
      if (state?.changed) {
        if (
          state.hasOtherChange ||
          !state.changedSnapshot ||
          orderChangeFingerprint(state.changedSnapshot) ===
            orderChangeFingerprint(query.data)
        )
          return previous;
        return { ...state, hasOtherChange: true };
      }
      return {
        ...state,
        owner,
        generation,
        snapshot: seed,
        changed: true,
        changedSnapshot: query.data,
      };
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
    current?.transitionSnapshot,
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
  const acceptTransition = useCallback(
    (
      previousOrder: API.Order,
      snapshot: API.Order,
      refreshedOrder: API.Order = snapshot,
    ) => {
      if (ownerRef.current !== owner || snapshot.id !== previousOrder.id)
        return;
      const validVersion =
        previousOrder.version != null &&
        snapshot.version != null &&
        /^\d+$/.test(previousOrder.version) &&
        /^\d+$/.test(snapshot.version) &&
        BigInt(snapshot.version) === BigInt(previousOrder.version) + 1n;
      const completedTermination =
        previousOrder.terminationStatus ===
          OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATING &&
        snapshot.terminationStatus ===
          OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED &&
        !snapshot.seaMasterBill &&
        snapshot.seaDocumentLinkVersion == null &&
        snapshot.seaDocumentSummary == null &&
        snapshot.seaDocumentStructure == null;
      const restoringSeaOrder =
        previousOrder.businessType === OrderBusinessType.BUSINESS_TYPE_SE &&
        previousOrder.terminationStatus ===
          OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED &&
        snapshot.terminationStatus ===
          OrderTerminationStatus.ORDER_TERMINATION_STATUS_ACTIVE;
      const validTransition =
        validVersion &&
        (completedTermination ||
          (restoringSeaOrder
            ? matchesRestoredSeaRelation(
                previousOrder,
                snapshot,
                retainedDocuments,
              )
            : orderChangeFingerprint({
                ...snapshot,
                version: previousOrder.version,
              }) === orderChangeFingerprint(previousOrder)));
      setObservation((previous) => {
        if (ownerRef.current !== owner || refreshedOrder.id !== snapshot.id)
          return previous;
        const state = previous?.owner === owner ? previous : undefined;
        // 重读已包含额外变动时立即保留提醒，不把更晚查询当作本人写入。
        if (
          !validTransition ||
          orderChangeFingerprint(refreshedOrder) !==
            orderChangeFingerprint(snapshot)
        ) {
          return {
            ...state,
            owner,
            generation: state?.generation ?? 0,
            snapshot: state?.snapshot ?? previousOrder,
            changed: true,
            changedSnapshot: state?.changedSnapshot ?? refreshedOrder,
            hasOtherChange: true,
          };
        }
        // 仅清除实际观察均等于本人写响应的误报，真正的外部差异持续保留。
        if (
          state?.changed &&
          (state.hasOtherChange ||
            !state.changedSnapshot ||
            orderChangeFingerprint(state.changedSnapshot) !==
              orderChangeFingerprint(snapshot))
        )
          return previous;
        const editing = state?.transitionSnapshot
          ? {
              order: state.transitionSnapshot,
              masterContentVersion:
                state.transitionSnapshot.seaMasterBill?.version,
            }
          : latest.current.readEditingOrder();
        const previouslyTerminated =
          previousOrder.terminationStatus ===
          OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED;
        if (
          !editing ||
          orderChangeFingerprint(editing.order, previouslyTerminated) !==
            orderChangeFingerprint(previousOrder, previouslyTerminated) ||
          (!previouslyTerminated &&
            editing.masterContentVersion !==
              previousOrder.seaMasterBill?.version)
        )
          return previous;
        // 仅推进检查参考点，不回填输入或表单提交版本；后续查询仍须与确切写响应比较。
        return {
          owner,
          generation: (state?.generation ?? 0) + 1,
          snapshot,
          transitionSnapshot: snapshot,
          changed: false,
        };
      });
    },
    [owner, retainedDocuments],
  );
  return {
    changed: current?.changed === true,
    acceptSnapshot,
    acceptTransition,
  };
}
