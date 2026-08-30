import { App } from 'antd';
import { useEffect, useState } from 'react';
import {
  masterDataServiceListAirports,
  masterDataServiceListOptions,
  masterDataServiceListPorts,
} from '@/services/roncin/masterDataService';
import { orderServiceListPersonnelOptions } from '@/services/roncin/orderService';
import { partnerServiceListPartners } from '@/services/roncin/partnerService';
import { unwrapList } from '@/utils/api';
import {
  MASTER_DATA_KINDS,
  type OrderKindConfig,
  isMasterDataKind,
  searchOrderLocations,
} from './common';

/** 订单列表页共用的主数据加载、候选项派生与联想搜索逻辑。 */
export function useOrderListResources(config?: OrderKindConfig) {
  const { message } = App.useApp();
  const [masterOptions, setMasterOptions] = useState<API.MasterDataItem[]>([]);
  const [ports, setPorts] = useState<API.Port[]>([]);
  const [airports, setAirports] = useState<API.Airport[]>([]);
  const [customerMap, setCustomerMap] = useState<Record<string, string>>({});

  useEffect(() => {
    void Promise.all([
      masterDataServiceListOptions(),
      masterDataServiceListPorts({ page: 1, pageSize: 50, enabled: true }),
      masterDataServiceListAirports({ page: 1, pageSize: 50, enabled: true }),
      partnerServiceListPartners({ role: 1, page: 1, pageSize: 50, enabled: true }),
    ])
      .then(
        ([
          optionsResponse,
          portsResponse,
          airportsResponse,
          partnersResponse,
        ]) => {
          setMasterOptions(unwrapList(optionsResponse));
          setPorts(unwrapList(portsResponse));
          setAirports(unwrapList(airportsResponse));
          const partnerList = unwrapList(partnersResponse);
          setCustomerMap((prev) => {
            const next = { ...prev };
            for (const p of partnerList) {
              if (p.id) {
                next[p.id] = p.legalName
                  ? `${p.legalName} (${p.code})`
                  : p.code || p.id;
              }
            }
            return next;
          });
        },
      )
      .catch((error: Error) =>
        message.error(error.message || '订单主数据加载失败'),
      );
  }, [message]);

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
    const res = await partnerServiceListPartners({
      role: 1,
      enabled: true,
      keyword,
    });
    const partners = unwrapList(res);
    setCustomerMap((prev) => {
      const next = { ...prev };
      for (const p of partners) {
        if (p.id) {
          next[p.id] = p.legalName
            ? `${p.legalName} (${p.code})`
            : p.code || p.id;
        }
      }
      return next;
    });
    return partners.map((p) => ({
      label: p.legalName ? `${p.legalName} (${p.code})` : p.code || p.id || '',
      value: p.id ?? '',
    }));
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
    const response = await partnerServiceListPartners({
      role: 4,
      page: 1,
      pageSize: 50,
      keyword,
      enabled: true,
    });
    return unwrapList(response).map((item) => ({
      label: item.legalName
        ? `${item.legalName} (${item.code})`
        : item.code || item.id || '',
      value: item.id ?? '',
    }));
  };

  const searchOrderPersonnel = async (keyword?: string) => {
    const response = await orderServiceListPersonnelOptions({
      businessType: config?.businessType ?? 1,
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
