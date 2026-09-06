import { useModel } from '@umijs/max';
import { App } from 'antd';
import { useEffect, useRef, useState } from 'react';
import { OrderBusinessType, PartnerRoleType } from '@/enums.generated';
import { masterDataServiceListPorts } from '@/services/roncin/masterDataService';
import { orderServiceListPersonnelOptions } from '@/services/roncin/orderService';
import { unwrapList } from '@/utils/api';
import { searchPartnerOptions } from '@/utils/options';
import {
  getCachedAirports,
  getCachedPorts,
  getMasterDataOptions,
} from '@/utils/order-options-cache';
import {
  isMasterDataKind,
  MASTER_DATA_KINDS,
  type OrderKindConfig,
  searchOrderLocations,
} from './common';

/** 订单列表页共用的主数据加载、候选项派生与联想搜索逻辑。 */
export function useOrderListResources(config?: OrderKindConfig) {
  const { message } = App.useApp();
  const { initialState } = useModel('@@initialState');
  const organizationId = initialState?.currentUser?.currentOrganization?.id;
  const activeOrgIdRef = useRef(organizationId);
  activeOrgIdRef.current = organizationId;
  const locationCategory = config?.category === 'air' ? 'air' : 'sea';
  const activeCategoryRef = useRef(locationCategory);
  activeCategoryRef.current = locationCategory;
  const personnelBusinessType =
    config?.businessType ?? OrderBusinessType.BUSINESS_TYPE_SE;
  const activeBusinessTypeRef = useRef(personnelBusinessType);
  activeBusinessTypeRef.current = personnelBusinessType;
  const requestIdRef = useRef(0);

  const [masterOptions, setMasterOptions] = useState<API.MasterDataItem[]>([]);
  const [ports, setPorts] = useState<API.Port[]>([]);
  const [airports, setAirports] = useState<API.Airport[]>([]);
  const [customerMap, setCustomerMap] = useState<Record<string, string>>({});
  const [loadedOrganizationId, setLoadedOrganizationId] = useState<
    string | null
  >(null);

  useEffect(() => {
    if (!organizationId) {
      setLoadedOrganizationId(null);
      setMasterOptions([]);
      setPorts([]);
      setAirports([]);
      setCustomerMap({});
      return;
    }

    const currentRequestId = ++requestIdRef.current;
    const currentOrgId = organizationId;
    const shouldLoadPorts = !config?.category || config.category === 'sea';
    const shouldLoadAirports = !config?.category || config.category === 'air';

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
        setLoadedOrganizationId(currentOrgId);
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
        setLoadedOrganizationId(null);
        message.error(error.message || '订单主数据加载失败');
      });
  }, [config?.category, message, organizationId]);

  const isOrgMatched = Boolean(
    organizationId && loadedOrganizationId === organizationId,
  );
  const effectiveMasterOptions = isOrgMatched ? masterOptions : [];
  const effectivePorts = isOrgMatched ? ports : [];
  const effectiveAirports = isOrgMatched ? airports : [];
  const effectiveCustomerMap = isOrgMatched ? customerMap : {};

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
    if (!requestOrgId) {
      return [];
    }
    const options = await searchPartnerOptions(keyword, {
      role: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
      enabled: true,
    });
    if (activeOrgIdRef.current !== requestOrgId) {
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
    if (!requestOrgId) {
      return [];
    }
    const response = await masterDataServiceListPorts({
      page: 1,
      pageSize: 50,
      keyword,
      enabled: true,
    });
    const result = unwrapList(response);
    if (activeOrgIdRef.current !== requestOrgId) {
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
    const requestCategory = locationCategory;
    if (!requestOrgId) {
      return [];
    }
    const options = await searchOrderLocations(requestCategory, keyword);
    return activeOrgIdRef.current === requestOrgId &&
      activeCategoryRef.current === requestCategory
      ? options
      : [];
  };

  const searchOrderCarriers = async (keyword?: string) => {
    const requestOrgId = organizationId;
    if (!requestOrgId) {
      return [];
    }
    const options = await searchPartnerOptions(keyword, {
      role: PartnerRoleType.PARTNER_ROLE_TYPE_CARRIER,
      enabled: true,
    });
    return activeOrgIdRef.current === requestOrgId ? options : [];
  };

  const searchOrderPersonnel = async (keyword?: string) => {
    const requestOrgId = organizationId;
    const requestBusinessType = personnelBusinessType;
    if (!requestOrgId) {
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
      activeBusinessTypeRef.current !== requestBusinessType
    ) {
      return [];
    }
    return unwrapList(response)
      .filter(
        (item) =>
          item.userId &&
          item.displayName &&
          item.organizationId &&
          item.organizationName,
      )
      .map((item) => ({
        userId: item.userId as string,
        displayName: item.displayName as string,
        organizationId: item.organizationId as string,
        organizationName: item.organizationName as string,
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
