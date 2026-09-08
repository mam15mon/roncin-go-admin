import { useModel } from '@umijs/max';
import { App } from 'antd';
import { useCallback, useEffect, useRef, useState } from 'react';
import { getFormDraftScope } from '@/components/layout/formDraft';
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

type DetailErrorState = {
  organizationId: string;
  orderId: string;
  category: OrderKindConfig['category'];
  businessType: number;
  error: Error;
};

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
  const activeCategoryRef = useRef(category);
  activeCategoryRef.current = category;
  const activeBusinessTypeRef = useRef(businessType);
  activeBusinessTypeRef.current = businessType;
  const [loading, setLoading] = useState(Boolean(config && orderId));
  const [order, setOrder] = useState<API.Order>();
  const [loadedOrderId, setLoadedOrderId] = useState<string | undefined>();
  const [loadedOrganizationId, setLoadedOrganizationId] = useState<
    string | undefined
  >();
  const [loadedCategory, setLoadedCategory] = useState<typeof category>();
  const [loadedBusinessType, setLoadedBusinessType] = useState<number>();
  const [errorState, setErrorState] = useState<DetailErrorState | null>(null);
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
      requestIdRef.current += 1;
      setOrder(undefined);
      setLoadedOrderId(undefined);
      setLoadedOrganizationId(undefined);
      setLoadedCategory(undefined);
      setLoadedBusinessType(undefined);
      setErrorState(null);
      setShippingDocs([]);
      setContainers([]);
      setCargoItems([]);
      setMilestones([]);
      setPersonnel([]);
      setLoading(false);
      return;
    }

    if (isUserLoaded && !organizationId) {
      requestIdRef.current += 1;
      setOrder(undefined);
      setLoadedOrderId(undefined);
      setLoadedOrganizationId(undefined);
      setLoadedCategory(undefined);
      setLoadedBusinessType(undefined);
      setErrorState(null);
      setShippingDocs([]);
      setContainers([]);
      setCargoItems([]);
      setMilestones([]);
      setPersonnel([]);
      setLoading(false);
      return;
    }

    if (!organizationId) {
      requestIdRef.current += 1;
      setLoading(true);
      setErrorState(null);
      return;
    }

    const currentRequestId = ++requestIdRef.current;
    const currentOrderId = orderId;
    const currentOrgId = organizationId;
    const currentCategory = config.category;
    const currentBusinessType = businessType;
    setLoading(true);
    setErrorState(null);
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
        currentOrgId !== activeOrgIdRef.current ||
        currentCategory !== activeCategoryRef.current ||
        currentBusinessType !== activeBusinessTypeRef.current
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
      setLoadedCategory(currentCategory);
      setLoadedBusinessType(currentBusinessType);
      setErrorState(null);
      setShippingDocs(unwrapList(docsRes));
      setContainers(unwrapList(cntrsRes));
      setCargoItems(unwrapList(cargoRes));
      setMilestones(unwrapList(milestonesRes));
      setPersonnel(unwrapList(personnelRes));
    } catch (err: any) {
      if (
        currentRequestId === requestIdRef.current &&
        currentOrderId === activeOrderIdRef.current &&
        currentOrgId === activeOrgIdRef.current &&
        currentCategory === activeCategoryRef.current &&
        currentBusinessType === activeBusinessTypeRef.current
      ) {
        setOrder(undefined);
        setLoadedOrderId(undefined);
        setLoadedOrganizationId(undefined);
        setLoadedCategory(undefined);
        setLoadedBusinessType(undefined);
        setErrorState({
          organizationId: currentOrgId,
          orderId: currentOrderId,
          category: currentCategory,
          businessType: currentBusinessType,
          error: err instanceof Error ? err : new Error(String(err)),
        });
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
        currentOrgId === activeOrgIdRef.current &&
        currentCategory === activeCategoryRef.current &&
        currentBusinessType === activeBusinessTypeRef.current
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

  const isOrderMatched = Boolean(
    orderId &&
      organizationId &&
      loadedOrderId === orderId &&
      loadedOrganizationId === organizationId &&
      loadedCategory === category &&
      loadedBusinessType === businessType,
  );
  const effectiveOrder = isOrderMatched ? order : undefined;
  const effectiveShippingDocs = isOrderMatched ? shippingDocs : [];
  const effectivePersonnel = isOrderMatched ? personnel : [];
  const effectiveLocationOptions = isOrderMatched ? locationOptions : [];

  const searchLocations = useCallback(
    async (keyword?: string) => {
      const requestOrgId = organizationId;
      const requestCategory = category;
      const requestBusinessType = businessType;
      if (!requestOrgId || !requestCategory) {
        return [];
      }

      if (!keyword?.trim()) {
        return activeOrgIdRef.current === requestOrgId &&
          activeCategoryRef.current === requestCategory &&
          activeBusinessTypeRef.current === requestBusinessType &&
          activeOrderIdRef.current === orderId &&
          loadedOrderId === orderId &&
          loadedOrganizationId === requestOrgId &&
          loadedCategory === requestCategory &&
          loadedBusinessType === requestBusinessType
          ? locationOptions
          : [];
      }

      const options = await searchOrderLocations(requestCategory, keyword);
      if (
        activeOrgIdRef.current !== requestOrgId ||
        activeCategoryRef.current !== requestCategory ||
        activeBusinessTypeRef.current !== requestBusinessType ||
        activeOrderIdRef.current !== orderId
      ) {
        return [];
      }
      return options;
    },
    [
      category,
      businessType,
      loadedCategory,
      loadedBusinessType,
      loadedOrderId,
      loadedOrganizationId,
      locationOptions,
      orderId,
      organizationId,
    ],
  );

  const missingOrgError =
    isUserLoaded && !organizationId
      ? new Error('缺少当前组织，无法加载订单详情')
      : null;
  const effectiveError =
    missingOrgError ||
    (organizationId &&
    orderId &&
    errorState?.organizationId === organizationId &&
    errorState.orderId === orderId &&
    errorState.category === category &&
    errorState.businessType === businessType
      ? errorState.error
      : null);
  const isPending =
    Boolean(config && orderId && organizationId) &&
    !isOrderMatched &&
    !effectiveError;

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
    locationOptions: effectiveLocationOptions,
    searchLocations,
    currencyOptions: isOrderMatched ? currencyOptions : [],
    containerSpecOptions: isOrderMatched ? containerSpecOptions : [],
    personnelOptions: isOrderMatched ? personnelOptions : [],
    draftScope: getFormDraftScope(
      initialState?.currentUser?.id,
      organizationId,
    ),
    loadData,
  };
}
