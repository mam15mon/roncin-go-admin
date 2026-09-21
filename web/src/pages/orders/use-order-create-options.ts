import { useQuery } from '@tanstack/react-query';
import { useCallback, useRef } from 'react';
import { useInitialState } from '@/app/AppProvider';
import {
  clearOrderMasterDataCache,
  getOrderPersonnelOptions,
} from '@/features/orders/options';
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

/** 新建订单页候选聚合载荷：queryFn 内并行拉取主数据与人员选项并完成映射。 */
interface OrderCreateOptionsBundle {
  serviceTypeOptions: SelectOption[];
  cargoCategoryOptions: SelectOption[];
  locationOptions: SelectOption[];
  currencyOptions: SelectOption[];
  containerSpecOptions: SelectOption[];
  personnelOptions: API.OrderPersonnelOption[];
}

/** 新建订单页的主数据与人员候选项加载。无 create 权限时完全静默（不发请求）。 */
export function useOrderCreateOptions(
  definition?: OrderKindDefinition,
  canCreate = true,
) {
  const { initialState } = useInitialState();
  const organizationId = initialState?.currentUser?.currentOrganization?.id;
  const isUserLoaded = Boolean(initialState?.currentUser);
  const transportMode = definition?.transportMode;
  const businessType = definition?.businessType;

  // 组织与业务身份全部就绪且具备 create 权限才允许发起请求；身份参数全部进入
  // queryKey，组织/业务配置切换即自然重查且互不串数据。
  const queryEnabled = Boolean(definition && organizationId) && canCreate;

  const optionsQuery = useQuery({
    queryKey: [
      'orders',
      'create-options',
      { organizationId, transportMode, businessType },
    ],
    enabled: queryEnabled,
    queryFn: async (): Promise<OrderCreateOptionsBundle> => {
      if (!organizationId || !definition) {
        // enabled 已保证身份齐备；此处仅为类型收窄兜底。
        throw new Error('缺少订单主数据加载参数');
      }
      try {
        const [masterData, personnelResponse] = await Promise.all([
          fetchOrderMasterData(organizationId, definition.transportMode),
          definition.transportMode === 'sea'
            ? getOrderPersonnelOptions(organizationId, definition.businessType)
            : Promise.resolve([]),
        ]);

        return {
          serviceTypeOptions:
            definition.transportMode === 'sea'
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
          personnelOptions: personnelResponse,
        };
      } catch (err) {
        // 动态错误文案：包装 Error 抛出，由全局 onError 展示（与旧实现
        // message.error(err.message || '加载订单主数据失败') 一致）。
        throw new Error(
          getErrorMessage(err, '加载订单主数据失败') || '加载订单主数据失败',
        );
      }
    },
  });

  const { data, error, isPending, isFetching, refetch } = optionsQuery;

  // 迟到搜索守卫：仅供 imperative 异步回调在 await 之后比对「当前身份」。
  // 主数据本身的竞态已由 React Query 按 queryKey 隔离，不再使用请求序号令牌。
  const latestIdentityRef = useRef({
    organizationId,
    transportMode,
    businessType,
  });
  latestIdentityRef.current = { organizationId, transportMode, businessType };

  const searchLocations = useCallback(
    async (keyword?: string) => {
      if (!organizationId || !transportMode || businessType == null) {
        return [];
      }

      if (!keyword?.trim()) {
        // 空关键字复用当前身份首批候选项；数据未就绪或加载失败时返回空。
        return error ? [] : (data?.locationOptions ?? []);
      }

      const options = await searchOrderLocations(transportMode, keyword);
      const latest = latestIdentityRef.current;
      if (
        latest.organizationId !== organizationId ||
        latest.transportMode !== transportMode ||
        latest.businessType !== businessType
      ) {
        return [];
      }
      return options;
    },
    [organizationId, transportMode, businessType, data, error],
  );

  const retry = useCallback(() => {
    if (!organizationId) {
      return Promise.resolve();
    }
    // 与旧行为一致：重试前先失效当前组织的主数据缓存。
    clearOrderMasterDataCache(organizationId);
    // refetch 的 Promise 只以结果对象 resolve、从不 reject，
    // 与旧实现「吸收请求错误并正常 resolve」的语义一致。
    return refetch().then(() => undefined);
  }, [organizationId, refetch]);

  const missingOrgError =
    isUserLoaded && !organizationId
      ? new Error('缺少当前组织，无法加载订单主数据')
      : null;
  const effectiveError = missingOrgError ?? error ?? null;
  const bundle = effectiveError ? undefined : data;

  // 首次拉取与重试期间保持加载态；用户信息未加载完成时同样保持加载，
  // 已登录但缺少组织、无 create 权限或加载失败时立即回落到错误/空态。
  const loading = !definition
    ? false
    : missingOrgError
      ? false
      : !canCreate
        ? false
        : isPending || isFetching;

  return {
    loading,
    error: effectiveError,
    retry,
    serviceTypeOptions: bundle?.serviceTypeOptions ?? [],
    cargoCategoryOptions: bundle?.cargoCategoryOptions ?? [],
    locationOptions: bundle?.locationOptions ?? [],
    searchLocations,
    currencyOptions: bundle?.currencyOptions ?? [],
    containerSpecOptions: bundle?.containerSpecOptions ?? [],
    personnelOptions: bundle?.personnelOptions ?? [],
  };
}
