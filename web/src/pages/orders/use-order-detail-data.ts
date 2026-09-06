import { useModel } from '@umijs/max';
import { App } from 'antd';
import { useCallback, useEffect, useRef, useState } from 'react';
import { OrderBusinessType } from '@/enums.generated';
import { orderCargoItemServiceListCargoItems } from '@/services/roncin/orderCargoItemService';
import { orderContainerServiceListContainers } from '@/services/roncin/orderContainerService';
import { orderMilestoneServiceListMilestones } from '@/services/roncin/orderMilestoneService';
import { orderPersonnelServiceListPersonnel } from '@/services/roncin/orderPersonnelService';
import { orderServiceGetOrder } from '@/services/roncin/orderService';
import { orderShippingDocumentServiceListShippingDocuments } from '@/services/roncin/orderShippingDocumentService';
import { unwrapList } from '@/utils/api';
import { getOrderPersonnelOptions } from '@/utils/order-options-cache';
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
  const { initialState } = useModel('@@initialState');
  const organizationId = initialState?.currentUser?.currentOrganization?.id;
  const isUserLoaded = Boolean(initialState?.currentUser);
  const activeOrgIdRef = useRef(organizationId);
  activeOrgIdRef.current = organizationId;
  const businessType =
    config?.businessType ?? OrderBusinessType.BUSINESS_TYPE_UNSPECIFIED;
  const category = config?.category;
  const [loading, setLoading] = useState(Boolean(config && orderId));
  const [order, setOrder] = useState<API.Order>();
  const [loadedOrderId, setLoadedOrderId] = useState<string | undefined>();
  const [loadedOrganizationId, setLoadedOrganizationId] = useState<
    string | undefined
  >();
  const [failedOrderId, setFailedOrderId] = useState<string | undefined>();
  const [error, setError] = useState<Error | null>(null);
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
      setOrder(undefined);
      setLoadedOrderId(undefined);
      setLoadedOrganizationId(undefined);
      setFailedOrderId(undefined);
      setError(null);
      setShippingDocs([]);
      setContainers([]);
      setCargoItems([]);
      setMilestones([]);
      setPersonnel([]);
      setLoading(false);
      return;
    }

    if (isUserLoaded && !organizationId) {
      setOrder(undefined);
      setLoadedOrderId(undefined);
      setLoadedOrganizationId(undefined);
      setFailedOrderId(undefined);
      setError(new Error('缺少当前组织，无法加载订单详情'));
      setShippingDocs([]);
      setContainers([]);
      setCargoItems([]);
      setMilestones([]);
      setPersonnel([]);
      setLoading(false);
      return;
    }

    if (!organizationId) {
      setLoading(true);
      return;
    }

    const currentRequestId = ++requestIdRef.current;
    const currentOrderId = orderId;
    const currentOrgId = organizationId;
    setLoading(true);
    setFailedOrderId(undefined);
    setError(null);
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
        fetchOrderMasterData(organizationId, category),
        category === 'sea'
          ? getOrderPersonnelOptions(organizationId, businessType)
          : Promise.resolve([]),
        orderServiceGetOrder({ id: orderId }),
        orderShippingDocumentServiceListShippingDocuments({ orderId }),
        orderContainerServiceListContainers({ orderId }),
        orderCargoItemServiceListCargoItems({ orderId }),
        orderMilestoneServiceListMilestones({ orderId }),
        orderPersonnelServiceListPersonnel({ orderId }),
      ]);

      if (
        currentRequestId !== requestIdRef.current ||
        currentOrderId !== activeOrderIdRef.current ||
        currentOrgId !== activeOrgIdRef.current
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
      setPersonnelOptions(personnelOptRes);

      setOrder(orderRes.data);
      setLoadedOrderId(currentOrderId);
      setLoadedOrganizationId(currentOrgId);
      setError(null);
      setShippingDocs(unwrapList(docsRes));
      setContainers(unwrapList(cntrsRes));
      setCargoItems(unwrapList(cargoRes));
      setMilestones(unwrapList(milestonesRes));
      setPersonnel(unwrapList(personnelRes));
    } catch (err: any) {
      if (
        currentRequestId === requestIdRef.current &&
        currentOrderId === activeOrderIdRef.current &&
        currentOrgId === activeOrgIdRef.current
      ) {
        setOrder(undefined);
        setLoadedOrderId(undefined);
        setLoadedOrganizationId(undefined);
        setFailedOrderId(currentOrderId);
        setError(err instanceof Error ? err : new Error(String(err)));
        setShippingDocs([]);
        setContainers([]);
        setCargoItems([]);
        setMilestones([]);
        setPersonnel([]);
        message.error(err.message || '加载订单数据失败');
      }
    } finally {
      if (
        currentRequestId === requestIdRef.current &&
        currentOrderId === activeOrderIdRef.current &&
        currentOrgId === activeOrgIdRef.current
      ) {
        setLoading(false);
      }
    }
  }, [
    businessType,
    category,
    config,
    isUserLoaded,
    message,
    orderId,
    organizationId,
  ]);

  useEffect(() => {
    void loadData();
  }, [loadData]);

  const searchLocations = useCallback(
    (keyword?: string) =>
      category ? searchOrderLocations(category, keyword) : Promise.resolve([]),
    [category],
  );

  const isOrderMatched = Boolean(
    orderId &&
    organizationId &&
    loadedOrderId === orderId &&
    loadedOrganizationId === organizationId,
  );
  const effectiveOrder = isOrderMatched ? order : undefined;
  const effectiveShippingDocs = isOrderMatched ? shippingDocs : [];
  const effectivePersonnel = isOrderMatched ? personnel : [];
  const isPending =
    Boolean(config && orderId && organizationId) &&
    !isOrderMatched &&
    failedOrderId !== orderId;

  const missingOrgError =
    isUserLoaded && !organizationId
      ? new Error('缺少当前组织，无法加载订单详情')
      : null;
  const effectiveError =
    missingOrgError || (failedOrderId === orderId ? error : null);

  const effectiveLoading = !config
    ? false
    : isUserLoaded && !organizationId
      ? false
      : effectiveError
        ? false
        : loading || isPending;

  return {
    loading: effectiveLoading,
    error: effectiveError,
    order: effectiveOrder,
    loadedOrderId,
    loadedOrganizationId,
    shippingDocs: effectiveShippingDocs,
    personnel: effectivePersonnel,
    serviceTypeOptions: isOrderMatched ? serviceTypeOptions : [],
    cargoCategoryOptions: isOrderMatched ? cargoCategoryOptions : [],
    locationOptions: isOrderMatched ? locationOptions : [],
    searchLocations,
    currencyOptions: isOrderMatched ? currencyOptions : [],
    containerSpecOptions: isOrderMatched ? containerSpecOptions : [],
    personnelOptions: isOrderMatched ? personnelOptions : [],
    loadData,
  };
}
