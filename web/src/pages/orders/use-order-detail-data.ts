import { App } from 'antd';
import { useCallback, useEffect, useState } from 'react';
import { orderCargoItemServiceListCargoItems } from '@/services/roncin/orderCargoItemService';
import { orderContainerServiceListContainers } from '@/services/roncin/orderContainerService';
import { orderMilestoneServiceListMilestones } from '@/services/roncin/orderMilestoneService';
import { orderPersonnelServiceListPersonnel } from '@/services/roncin/orderPersonnelService';
import {
  orderServiceGetOrder,
  orderServiceListPersonnelOptions,
} from '@/services/roncin/orderService';
import { orderShippingDocumentServiceListShippingDocuments } from '@/services/roncin/orderShippingDocumentService';
import {
  fetchOrderMasterData,
  isMasterDataKind,
  MASTER_DATA_KINDS,
  type OrderKindConfig,
  requireSeaServiceTypeOptions,
} from './common';
import type { SelectOption } from './templates';

/** 订单详情页的订单档案与主数据候选项加载。 */
export function useOrderDetailData(
  orderId: string | undefined,
  config: OrderKindConfig,
) {
  const { message } = App.useApp();
  const { businessType, category } = config;
  const [loading, setLoading] = useState(true);
  const [order, setOrder] = useState<API.Order>();
  const [shippingDocs, setShippingDocs] = useState<API.OrderShippingDocument[]>(
    [],
  );
  const [_containers, setContainers] = useState<API.OrderContainer[]>([]);
  const [_cargoItems, setCargoItems] = useState<API.OrderCargoItem[]>([]);
  const [_milestones, setMilestones] = useState<API.OrderMilestone[]>([]);
  const [personnel, setPersonnel] = useState<API.OrderPersonnel[]>([]);

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

  const loadData = useCallback(async () => {
    if (!orderId) return;
    setLoading(true);
    try {
      const [
        masterData,
        personnelOptRes,
        orderRes,
        docsRes,
        cntrsRes,
        cargoRes,
        milestonesRes,
        personnelRes,
      ] = await Promise.all([
        fetchOrderMasterData(),
        category === 'sea'
          ? orderServiceListPersonnelOptions({
              businessType,
              page: 1,
              pageSize: 200,
            })
          : Promise.resolve({ data: [] }),
        orderServiceGetOrder({ id: orderId }),
        orderShippingDocumentServiceListShippingDocuments({ orderId }),
        orderContainerServiceListContainers({ orderId }),
        orderCargoItemServiceListCargoItems({ orderId }),
        orderMilestoneServiceListMilestones({ orderId }),
        orderPersonnelServiceListPersonnel({ orderId }),
      ]);

      const nextServiceTypeOptions =
        category === 'sea'
          ? requireSeaServiceTypeOptions(masterData.serviceTypeOptions)
          : masterData.serviceTypeOptions;

      setServiceTypeOptions(nextServiceTypeOptions);
      setCargoCategoryOptions(masterData.cargoCategoryOptions);
      setLocationOptions(
        category === 'sea'
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
      setPersonnelOptions(personnelOptRes.data ?? []);

      setOrder(orderRes.data);
      setShippingDocs(docsRes.data ?? []);
      setContainers(cntrsRes.data ?? []);
      setCargoItems(cargoRes.data ?? []);
      setMilestones(milestonesRes.data ?? []);
      setPersonnel(personnelRes.data ?? []);
    } catch (error: any) {
      message.error(error.message || '加载订单数据失败');
    } finally {
      setLoading(false);
    }
  }, [businessType, category, message, orderId]);

  useEffect(() => {
    void loadData();
  }, [loadData]);

  return {
    loading,
    order,
    shippingDocs,
    personnel,
    serviceTypeOptions,
    cargoCategoryOptions,
    locationOptions,
    currencyOptions,
    containerSpecOptions,
    personnelOptions,
    loadData,
  };
}
