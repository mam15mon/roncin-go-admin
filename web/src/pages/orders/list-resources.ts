import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useRef } from 'react';
import { useInitialState } from '@/app/AppProvider';
import { PartnerRoleType } from '@/enums.generated';
import { searchShippingLineOptions } from '@/features/master-data/shipping-lines';
import {
  getCachedAirports,
  getCachedPorts,
  getMasterDataOptions,
} from '@/features/orders/options';
import { searchPartnerOptions } from '@/features/partners';
import { formatPersonnelLabel } from '@/features/personnel';
import { masterDataServiceListPorts } from '@/services/roncin/masterDataService';
import { orderServiceListPersonnelOptions } from '@/services/roncin/orderService';
import { ensureListResponse, unwrapList } from '@/utils/api';
import {
  isMasterDataKind,
  isUnimplementedTransportMode,
  MASTER_DATA_KINDS,
  searchOrderLocations,
} from './common';
import type { OrderKindDefinition } from './order-kinds/types';

interface OrderListResourceData {
  masterOptions: API.MasterDataItem[];
  ports: API.Port[];
  airports: API.Airport[];
  customerMap: Record<string, string>;
}

/** 订单列表页共用的主数据加载、候选项派生与联想搜索逻辑。 */
export function useOrderListResources(definition?: OrderKindDefinition) {
  const queryClient = useQueryClient();
  const { initialState } = useInitialState();
  const organizationId = initialState?.currentUser?.currentOrganization?.id;
  const activeOrgIdRef = useRef(organizationId);
  activeOrgIdRef.current = organizationId;
  const locationTransportMode = definition?.transportMode;
  const activeTransportModeRef = useRef(locationTransportMode);
  activeTransportModeRef.current = locationTransportMode;
  const personnelBusinessType = definition?.businessType;
  const activeBusinessTypeRef = useRef(personnelBusinessType);
  activeBusinessTypeRef.current = personnelBusinessType;
  const identity = {
    organizationId,
    transportMode: locationTransportMode,
    businessType: personnelBusinessType,
  };
  const resourceKey = ['orders', 'list-resources', identity] as const;
  const resourcesEnabled = Boolean(
    organizationId &&
      definition &&
      !isUnimplementedTransportMode(definition.transportMode),
  );
  const resources = useQuery({
    queryKey: resourceKey,
    enabled: resourcesEnabled,
    queryFn: async ({ signal }): Promise<OrderListResourceData> => {
      signal.throwIfAborted();
      if (!organizationId || !definition)
        throw new Error('缺少订单列表资源身份');
      const [masterOptions, ports, airports, customers] = await Promise.all([
        getMasterDataOptions(organizationId),
        definition.transportMode === 'sea'
          ? getCachedPorts(organizationId)
          : Promise.resolve([]),
        definition.transportMode === 'air'
          ? getCachedAirports(organizationId)
          : Promise.resolve([]),
        searchPartnerOptions(undefined, {
          role: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
          enabled: true,
        }),
      ]);
      signal.throwIfAborted();
      return {
        masterOptions,
        ports,
        airports,
        customerMap: Object.fromEntries(
          customers.map((option) => [option.value, option.label]),
        ),
      };
    },
    meta: { errorMessage: '订单主数据加载失败' },
  });
  const currentResources = resourcesEnabled ? resources.data : undefined;
  const effectiveMasterOptions = currentResources?.masterOptions ?? [];
  const effectivePorts = currentResources?.ports ?? [];
  const effectiveAirports = currentResources?.airports ?? [];
  const effectiveCustomerMap = currentResources?.customerMap ?? {};

  const fetchOptions = <T>(
    resource: string,
    keyword: string | undefined,
    queryFn: () => Promise<T>,
  ) =>
    queryClient.fetchQuery({
      queryKey: [
        'orders',
        resource,
        { ...identity, keyword: keyword?.trim() || undefined },
      ],
      queryFn,
      meta: { silent: true },
    });

  const containerSpecOptions = effectiveMasterOptions
    .filter(
      (item) =>
        isMasterDataKind(item.kind, MASTER_DATA_KINDS.CONTAINER_SPEC) &&
        item.enabled !== false,
    )
    .map((item) => ({
      label: item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      value: item.id ?? '',
    }));

  const containerSpecMap = Object.fromEntries(
    effectiveMasterOptions
      .filter(
        (item) =>
          isMasterDataKind(item.kind, MASTER_DATA_KINDS.CONTAINER_SPEC) &&
          item.id,
      )
      .map((item) => [
        item.id as string,
        item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      ]),
  );

  const cargoCategoryOptions = effectiveMasterOptions
    .filter(
      (item) =>
        isMasterDataKind(item.kind, MASTER_DATA_KINDS.CARGO_CATEGORY) &&
        item.enabled !== false,
    )
    .map((item) => ({
      label: item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      value: item.id ?? '',
    }));

  const regionLocationOptions = effectiveMasterOptions
    .filter(
      (item) =>
        isMasterDataKind(item.kind, MASTER_DATA_KINDS.REGION) &&
        item.enabled !== false,
    )
    .map((item) => ({
      label: item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      value: item.id ?? '',
    }));

  const locationOptions = [
    ...regionLocationOptions,
    ...effectivePorts
      .filter((item) => item.enabled !== false)
      .map((item) => ({
        label: `${item.nameZh ? `${item.nameZh} / ` : ''}${item.nameEn} (${item.unLocode})`,
        value: item.id ?? '',
      })),
    ...effectiveAirports
      .filter((item) => item.enabled !== false)
      .map((item) => ({
        label: `${item.nameZh ? `${item.nameZh} / ` : ''}${item.nameEn} (${item.iataCode})`,
        value: item.id ?? '',
      })),
  ];

  const searchCustomers = async (keyword?: string) => {
    const requestOrgId = organizationId;
    const requestTransportMode = locationTransportMode;
    if (!requestOrgId || !resourcesEnabled) {
      return [];
    }
    const options = await fetchOptions('list-customers', keyword, () =>
      searchPartnerOptions(keyword, {
        role: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
        enabled: true,
      }),
    );
    if (
      activeOrgIdRef.current !== requestOrgId ||
      activeTransportModeRef.current !== requestTransportMode ||
      activeBusinessTypeRef.current !== personnelBusinessType
    ) {
      return [];
    }
    queryClient.setQueryData<OrderListResourceData>(resourceKey, (previous) =>
      previous
        ? {
            ...previous,
            customerMap: {
              ...previous.customerMap,
              ...Object.fromEntries(
                options.map((option) => [option.value, option.label]),
              ),
            },
          }
        : undefined,
    );
    return options;
  };

  const searchOrderPorts = async (keyword?: string) => {
    const requestOrgId = organizationId;
    const requestTransportMode = locationTransportMode;
    // 港口联想只属于海运运输方式，其余类型直接关闭。
    if (!requestOrgId || !resourcesEnabled || requestTransportMode !== 'sea') {
      return [];
    }
    const result = await fetchOptions('list-ports', keyword, async () =>
      unwrapList(
        ensureListResponse(
          await masterDataServiceListPorts({
            page: 1,
            pageSize: 50,
            keyword,
            enabled: true,
          }),
          '港口主数据加载失败，请稍后重试',
        ),
      ),
    );
    // 组织或运输方式已切换的迟到港口响应不得写入当前资源。
    if (
      activeOrgIdRef.current !== requestOrgId ||
      activeTransportModeRef.current !== requestTransportMode ||
      activeBusinessTypeRef.current !== personnelBusinessType
    ) {
      return [];
    }
    queryClient.setQueryData<OrderListResourceData>(resourceKey, (previous) => {
      if (!previous) return undefined;
      const merged = new Map(
        previous.ports.filter((item) => item.id).map((item) => [item.id, item]),
      );
      for (const item of result) {
        if (item.id) merged.set(item.id, item);
      }
      return { ...previous, ports: [...merged.values()] };
    });
    return result.map((item) => ({
      label: `${item.nameZh ? `${item.nameZh} / ` : ''}${item.nameEn} (${item.unLocode})`,
      value: item.id ?? '',
    }));
  };

  const searchLocations = async (keyword?: string) => {
    const requestOrgId = organizationId;
    const requestTransportMode = locationTransportMode;
    if (!requestOrgId || !resourcesEnabled || !requestTransportMode) {
      return [];
    }
    const options = await fetchOptions('list-locations', keyword, () =>
      searchOrderLocations(requestTransportMode, keyword),
    );
    return activeOrgIdRef.current === requestOrgId &&
      activeTransportModeRef.current === requestTransportMode &&
      activeBusinessTypeRef.current === personnelBusinessType
      ? options
      : [];
  };

  const searchOrderCarriers = async (keyword?: string) => {
    const requestOrgId = organizationId;
    const requestTransportMode = locationTransportMode;
    if (!requestOrgId || !resourcesEnabled) {
      return [];
    }
    const options = await fetchOptions('list-carriers', keyword, () =>
      searchShippingLineOptions(keyword),
    );
    return activeOrgIdRef.current === requestOrgId &&
      activeTransportModeRef.current === requestTransportMode &&
      activeBusinessTypeRef.current === personnelBusinessType
      ? options
      : [];
  };

  const searchOrderPersonnel = async (keyword?: string) => {
    const requestOrgId = organizationId;
    const requestTransportMode = locationTransportMode;
    const requestBusinessType = personnelBusinessType;
    if (
      !requestOrgId ||
      !resourcesEnabled ||
      requestBusinessType === undefined
    ) {
      return [];
    }
    const response = await fetchOptions('list-personnel', keyword, () =>
      orderServiceListPersonnelOptions({
        businessType: requestBusinessType,
        keyword,
        page: 1,
        pageSize: 50,
      }),
    );
    if (
      activeOrgIdRef.current !== requestOrgId ||
      activeTransportModeRef.current !== requestTransportMode ||
      activeBusinessTypeRef.current !== requestBusinessType
    ) {
      return [];
    }
    return unwrapList(
      ensureListResponse(response, '订单人员选项加载失败，请稍后重试'),
    )
      .filter((item) => item.userId && item.displayName)
      .map((item) => ({
        userId: item.userId as string,
        displayName: formatPersonnelLabel(item),
      }));
  };

  return {
    masterOptions: effectiveMasterOptions,
    ports: effectivePorts,
    airports: effectiveAirports,
    customerMap: effectiveCustomerMap,
    containerSpecOptions,
    containerSpecMap,
    cargoCategoryOptions,
    locationOptions,
    searchLocations,
    searchCustomers,
    searchOrderPorts,
    searchOrderCarriers,
    searchOrderPersonnel,
  };
}
