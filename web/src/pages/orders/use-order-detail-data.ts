import { App } from 'antd';
import { useCallback, useEffect, useRef, useState } from 'react';
import { OrderBusinessType } from '@/enums.generated';
import { orderCargoItemServiceListCargoItems } from '@/services/roncin/orderCargoItemService';
import { orderContainerServiceListContainers } from '@/services/roncin/orderContainerService';
import { orderMilestoneServiceListMilestones } from '@/services/roncin/orderMilestoneService';
import { orderPersonnelServiceListPersonnel } from '@/services/roncin/orderPersonnelService';
import {
  orderServiceGetOrder,
  orderServiceListPersonnelOptions,
} from '@/services/roncin/orderService';
import { orderShippingDocumentServiceListShippingDocuments } from '@/services/roncin/orderShippingDocumentService';
import { unwrapList } from '@/utils/api';
import {
  fetchOrderMasterData,
  isMasterDataKind,
  MASTER_DATA_KINDS,
  type OrderKindConfig,
  requireSeaServiceTypeOptions,
  searchOrderLocations,
} from './common';
import type { SelectOption } from './templates';

/** 订单详情页的订单档案与主数据候选项加载。 */
export function useOrderDetailData(
  orderId: string | undefined,
  config?: OrderKindConfig,
) {
  const { message } = App.useApp();
  const businessType =
    config?.businessType ?? OrderBusinessType.BUSINESS_TYPE_UNSPECIFIED;
  const category = config?.category;
  const [loading, setLoading] = useState(Boolean(config && orderId));
  const [order, setOrder] = useState<API.Order>();
  const activeOrderIdRef = useRef(orderId);
  activeOrderIdRef.current = orderId;
  const requestIdRef = useRef(0);
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
    if (!orderId || !config) {
      setLoading(false);
      return;
    }
    const currentRequestId = ++requestIdRef.current;
    const currentOrderId = orderId;
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

      if (
        currentRequestId !== requestIdRef.current ||
        currentOrderId !== activeOrderIdRef.current
      ) {
        return;
      }

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
      setPersonnelOptions(unwrapList(personnelOptRes));

      setOrder(orderRes.data);
      setShippingDocs(unwrapList(docsRes));
      setContainers(unwrapList(cntrsRes));
      setCargoItems(unwrapList(cargoRes));
      setMilestones(unwrapList(milestonesRes));
      setPersonnel(unwrapList(personnelRes));
    } catch (error: any) {
      if (
        currentRequestId === requestIdRef.current &&
        currentOrderId === activeOrderIdRef.current
      ) {
        message.error(error.message || '加载订单数据失败');
      }
    } finally {
      if (
        currentRequestId === requestIdRef.current &&
        currentOrderId === activeOrderIdRef.current
      ) {
        setLoading(false);
      }
    }
  }, [businessType, category, message, orderId, config]);

  useEffect(() => {
    void loadData();
  }, [loadData]);

  const searchLocations = useCallback(
    (keyword?: string) =>
      category ? searchOrderLocations(category, keyword) : Promise.resolve([]),
    [category],
  );

  return {
    loading,
    order,
    shippingDocs,
    personnel,
    serviceTypeOptions,
    cargoCategoryOptions,
    locationOptions,
    searchLocations,
    currencyOptions,
    containerSpecOptions,
    personnelOptions,
    loadData,
  };
}
