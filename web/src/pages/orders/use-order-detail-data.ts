import { useQuery } from '@tanstack/react-query';
import { useCallback, useRef } from 'react';
import { useInitialState } from '@/app/AppProvider';
import { getFormDraftScope } from '@/components/layout/formDraft';
import { OrderBusinessType } from '@/enums.generated';
import { getOrderPersonnelOptions } from '@/features/orders/options';
import { orderPersonnelServiceListPersonnel } from '@/services/roncin/orderPersonnelService';
import { orderServiceGetOrder } from '@/services/roncin/orderService';
import { orderShippingDocumentServiceListShippingDocuments } from '@/services/roncin/orderShippingDocumentService';
import { ensureListResponse, unwrapList } from '@/utils/api';
import { getErrorMessage } from '@/utils/errorMessage';
import {
  fetchOrderMasterData,
  isMasterDataKind,
  MASTER_DATA_KINDS,
  requireSeaServiceTypeOptions,
  resolveOrderLocationOptions,
  searchOrderLocations,
} from './common';
import type { OrderKindDefinition } from './order-kinds/types';
import type { SelectOption } from './templates';

/**
 * 订单详情聚合载荷：queryFn 内并行拉取全部接口并完成候选映射，
 * 消费方拿到的即「同一订单身份下同时到达」的完整数据。
 */
interface OrderDetailBundle {
  order: API.Order | undefined;
  shippingDocs: API.OrderShippingDocument[];
  personnel: API.OrderPersonnel[];
  serviceTypeOptions: SelectOption[];
  cargoCategoryOptions: SelectOption[];
  locationOptions: SelectOption[];
  currencyOptions: SelectOption[];
  containerSpecOptions: SelectOption[];
  personnelOptions: API.OrderPersonnelOption[];
}

/** 订单详情域前缀：跨组件失效按该前缀 invalidate。 */
const ORDER_DETAIL_QUERY_PREFIX = 'order-detail';

/** 订单详情页的订单档案与主数据候选项加载。 */
export function useOrderDetailData(
  orderId: string | undefined,
  definition?: OrderKindDefinition,
) {
  const { initialState } = useInitialState();
  const organizationId = initialState?.currentUser?.currentOrganization?.id;
  const isUserLoaded = Boolean(initialState?.currentUser);
  const businessType =
    definition?.businessType ?? OrderBusinessType.BUSINESS_TYPE_UNSPECIFIED;
  const transportMode = definition?.transportMode;

  // 订单、组织、业务身份全部就绪才允许发起请求；
  // 身份参数全部进入 queryKey，组织/订单/配置切换即自然重查且互不串数据。
  const queryEnabled = Boolean(orderId && definition && organizationId);

  const detailQuery = useQuery({
    queryKey: [
      ORDER_DETAIL_QUERY_PREFIX,
      orderId,
      { organizationId, transportMode, businessType },
    ],
    enabled: queryEnabled,
    queryFn: async (): Promise<OrderDetailBundle> => {
      if (!orderId || !definition || !organizationId) {
        // enabled 已保证参数齐备；此处仅为类型收窄兜底。
        throw new Error('缺少订单详情加载参数');
      }
      try {
        const [masterData, personnelOptions, orderRes, docsRes, personnelRes] =
          await Promise.all([
            fetchOrderMasterData(organizationId, definition.transportMode),
            transportMode === 'sea'
              ? getOrderPersonnelOptions(organizationId, businessType)
              : Promise.resolve([]),
            orderServiceGetOrder({ id: orderId }),
            orderShippingDocumentServiceListShippingDocuments({ orderId }),
            orderPersonnelServiceListPersonnel({ orderId }),
          ]);

        // 请求层错误被全局处理器消费后会 resolve undefined：先转成明确
        // 业务错误，避免下方取 .data 抛 TypeError 文案直达用户。
        if (!orderRes) {
          throw new Error('订单详情加载失败，请稍后重试');
        }

        return {
          order: orderRes.data,
          shippingDocs: unwrapList(
            ensureListResponse(docsRes, '订单单证加载失败，请稍后重试'),
          ),
          personnel: unwrapList(
            ensureListResponse(personnelRes, '订单人员加载失败，请稍后重试'),
          ),
          serviceTypeOptions:
            transportMode === 'sea'
              ? requireSeaServiceTypeOptions(masterData.serviceTypeOptions)
              : masterData.serviceTypeOptions,
          cargoCategoryOptions: masterData.cargoCategoryOptions,
          locationOptions: resolveOrderLocationOptions(
            definition.transportMode,
            masterData,
          ),
          currencyOptions: masterData.currencyOptions,
          containerSpecOptions: masterData.masterOptions
            .filter(
              (item) =>
                isMasterDataKind(item.kind, MASTER_DATA_KINDS.CONTAINER_SPEC) &&
                item.enabled !== false,
            )
            .map((item) => ({
              label: item.code
                ? `${item.name ?? item.code} (${item.code})`
                : (item.name ?? ''),
              value: item.id ?? '',
            }))
            .filter((item) => item.value !== ''),
          personnelOptions,
        };
      } catch (err) {
        // 非 Error 抛出统一包装为带兜底文案的 Error，保证全局 cache onError
        // 弹出的文案与旧实现 getErrorMessage(err, '加载订单数据失败') 一致。
        throw err instanceof Error
          ? err
          : new Error(getErrorMessage(err, '加载订单数据失败'));
      }
    },
  });

  const { data, error, isPending, isFetching, refetch } = detailQuery;

  // 迟到搜索守卫：仅供 imperative 异步回调在 await 之后比对「当前身份」。
  // 详情数据本身的竞态已由 React Query 按 queryKey 隔离，不再使用请求序号令牌。
  const latestIdentityRef = useRef({
    organizationId,
    orderId,
    transportMode,
    businessType,
  });
  latestIdentityRef.current = {
    organizationId,
    orderId,
    transportMode,
    businessType,
  };

  const searchLocations = useCallback(
    async (keyword?: string) => {
      if (!organizationId || !transportMode) {
        return [];
      }

      if (!keyword?.trim()) {
        // 空关键字复用当前详情首批候选项；数据未就绪或加载失败时返回空。
        return error ? [] : (data?.locationOptions ?? []);
      }

      const options = await searchOrderLocations(transportMode, keyword);
      const latest = latestIdentityRef.current;
      if (
        latest.organizationId !== organizationId ||
        latest.orderId !== orderId ||
        latest.transportMode !== transportMode ||
        latest.businessType !== businessType
      ) {
        return [];
      }
      return options;
    },
    [organizationId, orderId, transportMode, businessType, data, error],
  );

  const missingOrgError =
    isUserLoaded && !organizationId
      ? new Error('缺少当前组织，无法加载订单详情')
      : null;
  const effectiveError = missingOrgError ?? error ?? null;

  const loadData = useCallback(() => {
    if (!queryEnabled) {
      return Promise.resolve();
    }
    // refetch 的 Promise 只以结果对象 resolve、从不 reject，
    // 与旧实现「吸收请求错误并正常 resolve」的语义一致。
    return refetch().then(() => undefined);
  }, [queryEnabled, refetch]);

  // 出错即隐藏全部数据并暴露错误（与旧实现的 errorState 清空语义一致）；
  // 首次拉取与显式刷新（含错误重试）期间保持加载态，复现旧的
  // 「loadData 发起即进入全页加载、失败回到错误页」行为。
  const loading =
    Boolean(definition) && !missingOrgError && (isPending || isFetching);

  const bundle = effectiveError ? undefined : data;

  return {
    loading,
    error: effectiveError,
    order: bundle?.order,
    loadedOrderId: bundle ? orderId : undefined,
    loadedOrganizationId: bundle ? organizationId : undefined,
    shippingDocs: bundle?.shippingDocs ?? [],
    personnel: bundle?.personnel ?? [],
    serviceTypeOptions: bundle?.serviceTypeOptions ?? [],
    cargoCategoryOptions: bundle?.cargoCategoryOptions ?? [],
    locationOptions: bundle?.locationOptions ?? [],
    searchLocations,
    currencyOptions: bundle?.currencyOptions ?? [],
    containerSpecOptions: bundle?.containerSpecOptions ?? [],
    personnelOptions: bundle?.personnelOptions ?? [],
    draftScope: getFormDraftScope(
      initialState?.currentUser?.id,
      organizationId,
    ),
    loadData,
  };
}
