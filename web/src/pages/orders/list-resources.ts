import { useModel } from '@umijs/max';
import { App } from 'antd';
import { useEffect, useRef, useState } from 'react';
import { PartnerRoleType } from '@/enums.generated';
import { masterDataServiceListPorts } from '@/services/roncin/masterDataService';
import { orderServiceListPersonnelOptions } from '@/services/roncin/orderService';
import { unwrapList } from '@/utils/api';
import {
  searchPartnerOptions,
  searchShippingLineOptions,
} from '@/utils/options';
import {
  getCachedAirports,
  getCachedPorts,
  getMasterDataOptions,
} from '@/utils/order-options-cache';
import {
  isMasterDataKind,
  isUnimplementedTransportMode,
  MASTER_DATA_KINDS,
  searchOrderLocations,
} from './common';
import type { OrderKindDefinition } from './order-kinds/types';

/** 订单列表页共用的主数据加载、候选项派生与联想搜索逻辑。 */
export function useOrderListResources(definition?: OrderKindDefinition) {
  const { message } = App.useApp();
  const { initialState } = useModel('@@initialState');
  const organizationId = initialState?.currentUser?.currentOrganization?.id;
  const activeOrgIdRef = useRef(organizationId);
  activeOrgIdRef.current = organizationId;
  const locationTransportMode = definition?.transportMode;
  const activeTransportModeRef = useRef(locationTransportMode);
  activeTransportModeRef.current = locationTransportMode;
  const personnelBusinessType = definition?.businessType;
  const activeBusinessTypeRef = useRef(personnelBusinessType);
  activeBusinessTypeRef.current = personnelBusinessType;
  const requestIdRef = useRef(0);

  const [masterOptions, setMasterOptions] = useState<API.MasterDataItem[]>([]);
  const [ports, setPorts] = useState<API.Port[]>([]);
  const [airports, setAirports] = useState<API.Airport[]>([]);
  const [customerMap, setCustomerMap] = useState<Record<string, string>>({});
  // 已加载资源携带完整身份（组织 + 运输方式）：同组织 sea → air 切换时，
  // 即使旧 state 尚未清空，身份不匹配也会让旧港口立即从对外结果中隐藏。
  const [loadedResourceIdentity, setLoadedResourceIdentity] = useState<
    string | undefined
  >(undefined);

  useEffect(() => {
    // 未注册订单类型与未开放运输方式 fail-closed：不发任何主数据请求，
    // 清空已加载资源；后续真实接入 land/rail 时在此接入其专属主数据装载。
    if (
      !organizationId ||
      !definition ||
      isUnimplementedTransportMode(definition.transportMode)
    ) {
      requestIdRef.current += 1;
      setLoadedResourceIdentity(undefined);
      setMasterOptions([]);
      setPorts([]);
      setAirports([]);
      setCustomerMap({});
      return;
    }

    const currentRequestId = ++requestIdRef.current;
    const currentOrgId = organizationId;
    const currentResourceIdentity = `${organizationId}:${definition.transportMode}`;
    const shouldLoadPorts = definition.transportMode === 'sea';
    const shouldLoadAirports = definition.transportMode === 'air';

    void Promise.all([
      getMasterDataOptions(organizationId),
      shouldLoadPorts ? getCachedPorts(organizationId) : Promise.resolve([]),
      shouldLoadAirports
        ? getCachedAirports(organizationId)
        : Promise.resolve([]),
      searchPartnerOptions(undefined, {
        role: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
        enabled: true,
      }),
    ])
      .then(([options, portsList, airportsList, partnerOptions]) => {
        if (
          currentRequestId !== requestIdRef.current ||
          currentOrgId !== activeOrgIdRef.current
        ) {
          return;
        }
        setLoadedResourceIdentity(currentResourceIdentity);
        setMasterOptions(options);
        setPorts(portsList);
        setAirports(airportsList);
        const nextCustomerMap: Record<string, string> = {};
        for (const option of partnerOptions) {
          nextCustomerMap[option.value] = option.label;
        }
        setCustomerMap(nextCustomerMap);
      })
      .catch((error: Error) => {
        if (
          currentRequestId !== requestIdRef.current ||
          currentOrgId !== activeOrgIdRef.current
        ) {
          return;
        }
        setLoadedResourceIdentity(undefined);
        message.error(error.message || '订单主数据加载失败');
      });
  }, [definition, message, organizationId]);

  const resourceIdentity =
    organizationId && definition
      ? `${organizationId}:${definition.transportMode}`
      : undefined;
  const resourcesMatched =
    Boolean(resourceIdentity) && loadedResourceIdentity === resourceIdentity;
  const effectiveMasterOptions = resourcesMatched ? masterOptions : [];
  const effectivePorts = resourcesMatched ? ports : [];
  const effectiveAirports = resourcesMatched ? airports : [];
  const effectiveCustomerMap = resourcesMatched ? customerMap : {};

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

  const serviceTypeOptions = effectiveMasterOptions
    .filter(
      (item) =>
        isMasterDataKind(item.kind, MASTER_DATA_KINDS.SERVICE_TYPE) &&
        item.enabled !== false,
    )
    .map((item) => ({
      label: item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      value: item.id ?? '',
    }));

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

  // 未注册类型与未开放运输方式的对外联想入口统一关闭：
  // 既不请求，也不抛「尚未开放」，避免部分加载、部分报错的混合语义。
  const resourcesEnabled =
    definition?.transportMode !== undefined &&
    !isUnimplementedTransportMode(definition.transportMode) &&
    Boolean(organizationId);

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
    const options = await searchPartnerOptions(keyword, {
      role: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
      enabled: true,
    });
    if (
      activeOrgIdRef.current !== requestOrgId ||
      activeTransportModeRef.current !== requestTransportMode
    ) {
      return [];
    }
    setCustomerMap((prev) => {
      const next = { ...prev };
      for (const option of options) {
        next[option.value] = option.label;
      }
      return next;
    });
    return options;
  };

  const searchOrderPorts = async (keyword?: string) => {
    const requestOrgId = organizationId;
    const requestTransportMode = locationTransportMode;
    // 港口联想只属于海运运输方式，其余类型直接关闭。
    if (!requestOrgId || !resourcesEnabled || requestTransportMode !== 'sea') {
      return [];
    }
    const response = await masterDataServiceListPorts({
      page: 1,
      pageSize: 50,
      keyword,
      enabled: true,
    });
    const result = unwrapList(response);
    // 组织或运输方式已切换的迟到港口响应不得写入当前资源。
    if (
      activeOrgIdRef.current !== requestOrgId ||
      activeTransportModeRef.current !== requestTransportMode
    ) {
      return [];
    }
    setPorts((current) => {
      const merged = new Map(
        current.filter((item) => item.id).map((item) => [item.id, item]),
      );
      for (const item of result) {
        if (item.id) merged.set(item.id, item);
      }
      return [...merged.values()];
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
    const options = await searchOrderLocations(requestTransportMode, keyword);
    return activeOrgIdRef.current === requestOrgId &&
      activeTransportModeRef.current === requestTransportMode
      ? options
      : [];
  };

  const searchOrderCarriers = async (keyword?: string) => {
    const requestOrgId = organizationId;
    const requestTransportMode = locationTransportMode;
    if (!requestOrgId || !resourcesEnabled) {
      return [];
    }
    const options = await searchShippingLineOptions(keyword);
    return activeOrgIdRef.current === requestOrgId &&
      activeTransportModeRef.current === requestTransportMode
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
    const response = await orderServiceListPersonnelOptions({
      businessType: requestBusinessType,
      keyword,
      page: 1,
      pageSize: 50,
    });
    if (
      activeOrgIdRef.current !== requestOrgId ||
      activeTransportModeRef.current !== requestTransportMode ||
      activeBusinessTypeRef.current !== requestBusinessType
    ) {
      return [];
    }
    return unwrapList(response)
      .filter((item) => item.userId && item.displayName)
      .map((item) => ({
        userId: item.userId as string,
        displayName: item.displayName as string,
      }));
  };

  return {
    masterOptions: effectiveMasterOptions,
    ports: effectivePorts,
    airports: effectiveAirports,
    customerMap: effectiveCustomerMap,
    containerSpecOptions,
    containerSpecMap,
    serviceTypeOptions,
    cargoCategoryOptions,
    locationOptions,
    searchLocations,
    searchCustomers,
    searchOrderPorts,
    searchOrderCarriers,
    searchOrderPersonnel,
  };
}
