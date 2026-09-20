import { useQuery } from '@tanstack/react-query';
import { useCallback, useMemo, useRef, useState } from 'react';
import { orderFeeServiceListFeeOptions } from '@/services/roncin/orderFeeService';
import { orderServiceGetOrder } from '@/services/roncin/orderService';
import { getErrorMessage } from '@/utils/errorMessage';

/** 订单费用工作台聚合载荷：queryFn 内并行拉取订单档案与候选项。 */
interface OrderFeeOptionsBundle {
  order: API.Order | undefined;
  currencies: API.OrderFeeCurrencyOption[];
  settlementParties: API.OrderFeeSettlementPartyOption[];
  feeSettings: API.OrderFeeSettingOption[];
  billingUnits: API.OrderFeeBillingUnitOption[];
  financeLocked: boolean;
  financeLockReason: string;
  financeLockCommissionNos: string[];
  customerName: string;
}

/**
 * 本地新增候选项覆盖层：快捷新建费目/往来单位后即时可用。绑定到发起时的
 * 订单身份防止跨订单串数据；只保存相对服务端基线的差集，重查返回后已
 * 落库的项自动并入基线，未落库的项保持可见。
 */
interface OrderBoundExtras<T> {
  orderId: string | undefined;
  items: T[];
}

const INITIAL_EXTRAS: OrderBoundExtras<never> = {
  orderId: undefined,
  items: [],
};

/**
 * setter 更新器共享算法：基于「未落库差集 + 服务端基线」拼出当前全量，
 * 应用入参或更新器后再收敛回差集；订单身份任一维度变化时旧差集整体丢弃。
 */
function applyNextList<T extends { id?: string }>(
  prev: OrderBoundExtras<T>,
  base: T[],
  currentOrderId: string | undefined,
  next: T[] | ((prev: T[]) => T[]),
): OrderBoundExtras<T> {
  const baseIds = new Set(base.map((item) => item.id));
  const prevExtras =
    prev.orderId === currentOrderId
      ? prev.items.filter((item) => item.id && !baseIds.has(item.id))
      : [];
  const current = [...prevExtras, ...base];
  const value = typeof next === 'function' ? next(current) : next;
  return {
    orderId: currentOrderId,
    items: value.filter((item) => item.id && !baseIds.has(item.id)),
  };
}

/** 加载订单档案与费用录入候选项、财务锁定状态。 */
export function useOrderFeeOptions(orderId?: string) {
  const feeOptionsQuery = useQuery({
    queryKey: ['orders', 'fee-options', { orderId }],
    enabled: Boolean(orderId),
    queryFn: async (): Promise<OrderFeeOptionsBundle> => {
      if (!orderId) {
        // enabled 已保证订单身份齐备；此处仅为类型收窄兜底。
        throw new Error('缺少订单费用选项加载参数');
      }
      try {
        const [orderRes, optionsRes] = await Promise.all([
          orderServiceGetOrder({ id: orderId }),
          orderFeeServiceListFeeOptions({ orderId }),
        ]);
        return {
          order: orderRes.data,
          currencies: optionsRes.currencies ?? [],
          settlementParties: optionsRes.settlementParties ?? [],
          feeSettings: optionsRes.feeSettings ?? [],
          billingUnits: optionsRes.billingUnits ?? [],
          financeLocked: Boolean(optionsRes.financeLocked),
          financeLockReason: optionsRes.financeLockReason || '',
          financeLockCommissionNos: optionsRes.financeLockCommissionNos || [],
          customerName: optionsRes.customerName || '',
        };
      } catch (error) {
        // 动态错误文案：包装 Error 抛出，由全局 onError 展示（与旧实现
        // message.error(getErrorMessage(error, '加载费用信息失败')) 一致）。
        throw new Error(getErrorMessage(error, '加载费用信息失败'));
      }
    },
  });

  const { data, error, isPending, isFetching, refetch } = feeOptionsQuery;

  // 迟到写入守卫：覆盖层写入按发起时的订单身份绑定，切单后旧覆盖层自然失效。
  const orderIdRef = useRef(orderId);
  orderIdRef.current = orderId;

  const [extraSettlementParties, setExtraSettlementParties] =
    useState<OrderBoundExtras<API.OrderFeeSettlementPartyOption>>(
      INITIAL_EXTRAS,
    );
  const [extraFeeSettings, setExtraFeeSettings] =
    useState<OrderBoundExtras<API.OrderFeeSettingOption>>(INITIAL_EXTRAS);

  const baseSettlementParties = data?.settlementParties ?? [];
  const baseFeeSettings = data?.feeSettings ?? [];
  const settlementPartiesBaseRef = useRef(baseSettlementParties);
  settlementPartiesBaseRef.current = baseSettlementParties;
  const feeSettingsBaseRef = useRef(baseFeeSettings);
  feeSettingsBaseRef.current = baseFeeSettings;

  const mergeExtras = useCallback(
    <T extends { id?: string }>(
      extras: OrderBoundExtras<T>,
      base: T[],
    ): T[] => {
      if (extras.orderId !== orderIdRef.current || extras.items.length === 0) {
        return base;
      }
      const baseIds = new Set(base.map((item) => item.id));
      const pending = extras.items.filter(
        (item) => item.id && !baseIds.has(item.id),
      );
      return pending.length > 0 ? [...pending, ...base] : base;
    },
    [],
  );

  const settlementParties = useMemo(
    () => mergeExtras(extraSettlementParties, baseSettlementParties),
    [extraSettlementParties, baseSettlementParties, mergeExtras],
  );
  const feeSettings = useMemo(
    () => mergeExtras(extraFeeSettings, baseFeeSettings),
    [extraFeeSettings, baseFeeSettings, mergeExtras],
  );

  // setter 语义与旧实现一致：入参或更新器都基于「当前生效列表」计算下一份
  // 全量列表；本地只保留相对服务端基线的差集。
  const setSettlementParties = useCallback(
    (
      next:
        | API.OrderFeeSettlementPartyOption[]
        | ((
            prev: API.OrderFeeSettlementPartyOption[],
          ) => API.OrderFeeSettlementPartyOption[]),
    ) => {
      setExtraSettlementParties((prev) =>
        applyNextList(
          prev,
          settlementPartiesBaseRef.current,
          orderIdRef.current,
          next,
        ),
      );
    },
    [],
  );

  const setFeeSettings = useCallback(
    (
      next:
        | API.OrderFeeSettingOption[]
        | ((prev: API.OrderFeeSettingOption[]) => API.OrderFeeSettingOption[]),
    ) => {
      setExtraFeeSettings((prev) =>
        applyNextList(prev, feeSettingsBaseRef.current, orderIdRef.current, next),
      );
    },
    [],
  );

  // 出错即隐藏全部数据（与旧实现失败清空语义一致）；首次拉取与显式刷新
  // （含错误重试）期间保持加载态。
  const bundle = error ? undefined : data;
  const loading = Boolean(orderId) && (isPending || isFetching);

  const loadData = useCallback(() => {
    if (!orderId) {
      return Promise.resolve();
    }
    // refetch 的 Promise 只以结果对象 resolve、从不 reject，
    // 与旧实现「吸收请求错误并正常 resolve」的语义一致。
    return refetch().then(() => undefined);
  }, [orderId, refetch]);

  return {
    loading,
    order: bundle?.order,
    loadedOrderId: bundle ? orderId : undefined,
    currencies: bundle?.currencies ?? [],
    settlementParties,
    setSettlementParties,
    feeSettings,
    setFeeSettings,
    billingUnits: bundle?.billingUnits ?? [],
    financeLocked: Boolean(bundle?.financeLocked),
    financeLockReason: bundle?.financeLockReason ?? '',
    financeLockCommissionNos: bundle?.financeLockCommissionNos ?? [],
    customerName: bundle?.customerName ?? '',
    loadData,
  };
}
