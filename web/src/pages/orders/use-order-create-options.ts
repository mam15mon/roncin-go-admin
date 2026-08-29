import { App } from 'antd';
import { useEffect, useState } from 'react';
import { orderServiceListPersonnelOptions } from '@/services/roncin/orderService';
import {
  fetchOrderMasterData,
  isMasterDataKind,
  MASTER_DATA_KINDS,
  type OrderKindConfig,
  requireSeaServiceTypeOptions,
} from './common';
import type { SelectOption } from './templates';

/** 新建订单页的主数据与人员候选项加载。 */
export function useOrderCreateOptions(config?: OrderKindConfig) {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(true);
  const [serviceTypeOptions, setServiceTypeOptions] = useState<SelectOption[]>(
    [],
  );
  const [cargoCategoryOptions, setCargoCategoryOptions] = useState<
    SelectOption[]
  >([]);
  const [locationOptions, setLocationOptions] = useState<SelectOption[]>([]);
  const [currencyOptions, setCurrencyOptions] = useState<SelectOption[]>([]);
  const [containerSpecOptions, setContainerSpecOptions] = useState<
    SelectOption[]
  >([]);
  const [personnelOptions, setPersonnelOptions] = useState<
    API.OrderPersonnelOption[]
  >([]);

  useEffect(() => {
    if (!config) {
      setLoading(false);
      return;
    }

    setLoading(true);
    Promise.all([
      fetchOrderMasterData(),
      config.category === 'sea'
        ? orderServiceListPersonnelOptions({
            businessType: config.businessType,
            page: 1,
            pageSize: 200,
          })
        : Promise.resolve({ data: [] }),
    ])
      .then(([masterData, personnelResponse]) => {
        const nextServiceTypeOptions =
          config.category === 'sea'
            ? requireSeaServiceTypeOptions(masterData.serviceTypeOptions)
            : masterData.serviceTypeOptions;

        setServiceTypeOptions(nextServiceTypeOptions);
        setCargoCategoryOptions(masterData.cargoCategoryOptions);
        setLocationOptions(
          config.category === 'sea'
            ? masterData.seaLocationOptions
            : masterData.airLocationOptions,
        );
        setCurrencyOptions(masterData.currencyOptions);
        setContainerSpecOptions(
          masterData.masterOptions
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
        );
        setPersonnelOptions(personnelResponse.data ?? []);
      })
      .catch((error: Error) => {
        message.error(error.message || '加载订单主数据失败');
      })
      .finally(() => {
        setLoading(false);
      });
  }, [config, message]);

  return {
    loading,
    serviceTypeOptions,
    cargoCategoryOptions,
    locationOptions,
    currencyOptions,
    containerSpecOptions,
    personnelOptions,
  };
}
