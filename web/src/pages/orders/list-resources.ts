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
  MASTER_DATA_KINDS,
  type OrderKindConfig,
  isMasterDataKind,
  searchOrderLocations,
} from './common';

/** 订单列表页共用的主数据加载、候选项派生与联想搜索逻辑。 */
export function useOrderListResources(config?: OrderKindConfig) {
  const { message } = App.useApp();
  const { initialState } = useModel('@@initialState');
  const organizationId = initialState?.currentUser?.currentOrganization?.id;
  const activeOrgIdRef = useRef(organizationId);
  activeOrgIdRef.current = organizationId;
  const requestIdRef = useRef(0);

  const [masterOptions, setMasterOptions] = useState<API.MasterDataItem[]>([]);
  const [ports, setPorts] = useState<API.Port[]>([]);
  const [airports, setAirports] = useState<API.Airport[]>([]);
  const [customerMap, setCustomerMap] = useState<Record<string, string>>({});

  useEffect(() => {
    if (!organizationId) {
      setMasterOptions([]);
      setPorts([]);
      setAirports([]);
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
      .then(
        ([
          options,
          portsList,
          airportsList,
          partnerOptions,
        ]) => {
          if (
            currentRequestId !== requestIdRef.current ||
            currentOrgId !== activeOrgIdRef.current
          ) {
            return;
          }
          setMasterOptions(options);
          setPorts(portsList);
          setAirports(airportsList);
          setCustomerMap((prev) => {
            const next = { ...prev };
            for (const option of partnerOptions) {
              next[option.value] = option.label;
            }
            return next;
          });
        },
      )
      .catch((error: Error) => {
        if (
          currentRequestId !== requestIdRef.current ||
          currentOrgId !== activeOrgIdRef.current
        ) {
          return;
        }
        message.error(error.message || '订单主数据加载失败');
      });
  }, [config?.category, message, organizationId]);

  const containerSpecOptions = masterOptions
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
    masterOptions
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

  const serviceTypeOptions = masterOptions
    .filter(
      (item) =>
        isMasterDataKind(item.kind, MASTER_DATA_KINDS.SERVICE_TYPE) &&
        item.enabled !== false,
    )
    .map((item) => ({
      label: item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      value: item.id ?? '',
    }));

  const cargoCategoryOptions = masterOptions
    .filter(
      (item) =>
        isMasterDataKind(item.kind, MASTER_DATA_KINDS.CARGO_CATEGORY) &&
        item.enabled !== false,
    )
    .map((item) => ({
      label: item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      value: item.id ?? '',
    }));

  const regionLocationOptions = masterOptions
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
    ...ports
      .filter((item) => item.enabled !== false)
      .map((item) => ({
        label: `${item.nameZh ? `${item.nameZh} / ` : ''}${item.nameEn} (${item.unLocode})`,
        value: item.id ?? '',
      })),
    ...airports
      .filter((item) => item.enabled !== false)
      .map((item) => ({
        label: `${item.nameZh ? `${item.nameZh} / ` : ''}${item.nameEn} (${item.iataCode})`,
        value: item.id ?? '',
      })),
  ];

  const searchCustomers = async (keyword?: string) => {
    const options = await searchPartnerOptions(keyword, {
      role: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
      enabled: true,
    });
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
    const response = await masterDataServiceListPorts({
      page: 1,
      pageSize: 50,
      keyword,
      enabled: true,
    });
    const result = unwrapList(response);
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

  const searchLocations = (keyword?: string) =>
    searchOrderLocations(config?.category === 'air' ? 'air' : 'sea', keyword);

  const searchOrderCarriers = async (keyword?: string) => {
    return searchPartnerOptions(keyword, {
      role: PartnerRoleType.PARTNER_ROLE_TYPE_CARRIER,
      enabled: true,
    });
  };

  const searchOrderPersonnel = async (keyword?: string) => {
    const response = await orderServiceListPersonnelOptions({
      businessType: config?.businessType ?? OrderBusinessType.BUSINESS_TYPE_SE,
      keyword,
      page: 1,
      pageSize: 50,
    });
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
    masterOptions,
    ports,
    airports,
    customerMap,
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
