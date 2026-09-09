import { useModel } from '@umijs/max';
import { App } from 'antd';
import { useCallback, useEffect, useRef, useState } from 'react';
import { getFormDraftScope } from '@/components/layout/formDraft';
import { OrderBusinessType } from '@/enums.generated';
import { orderPersonnelServiceListPersonnel } from '@/services/roncin/orderPersonnelService';
import { orderServiceGetOrder } from '@/services/roncin/orderService';
import { orderShippingDocumentServiceListShippingDocuments } from '@/services/roncin/orderShippingDocumentService';
import { unwrapList } from '@/utils/api';
import { getOrderPersonnelOptions } from '@/utils/order-options-cache';
import {
  fetchOrderMasterData,
  isMasterDataKind,
  MASTER_DATA_KINDS,
  requireSeaServiceTypeOptions,
  resolveOrderLocationOptions,
  searchOrderLocations,
} from './common';
import type { OrderKindDefinition } from './order-kinds/types';
import type { SelectOption } from './templates';

type DetailErrorState = {
  organizationId: string;
  orderId: string;
  transportMode: OrderKindDefinition['transportMode'];
  businessType: number;
  error: Error;
};

/** 订单详情页的订单档案与主数据候选项加载。 */
export function useOrderDetailData(
  orderId: string | undefined,
  definition?: OrderKindDefinition,
) {
  const { message } = App.useApp();
  const { initialState } = useModel('@@initialState');
  const organizationId = initialState?.currentUser?.currentOrganization?.id;
  const isUserLoaded = Boolean(initialState?.currentUser);
  const activeOrgIdRef = useRef(organizationId);
  activeOrgIdRef.current = organizationId;
  const businessType =
    definition?.businessType ?? OrderBusinessType.BUSINESS_TYPE_UNSPECIFIED;
  const transportMode = definition?.transportMode;
  const activeTransportModeRef = useRef(transportMode);
  activeTransportModeRef.current = transportMode;
  const activeBusinessTypeRef = useRef(businessType);
  activeBusinessTypeRef.current = businessType;
  const [loading, setLoading] = useState(Boolean(definition && orderId));
  const [order, setOrder] = useState<API.Order>();
  const [loadedOrderId, setLoadedOrderId] = useState<string | undefined>();
  const [loadedOrganizationId, setLoadedOrganizationId] = useState<
    string | undefined
  >();
  const [loadedTransportMode, setLoadedTransportMode] =
    useState<typeof transportMode>();
  const [loadedBusinessType, setLoadedBusinessType] = useState<number>();
  const [errorState, setErrorState] = useState<DetailErrorState | null>(null);
  const activeOrderIdRef = useRef(orderId);
  activeOrderIdRef.current = orderId;
  const requestIdRef = useRef(0);
  const [shippingDocs, setShippingDocs] = useState<API.OrderShippingDocument[]>(
    [],
  );
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
    if (!orderId || !definition) {
      requestIdRef.current += 1;
      setOrder(undefined);
      setLoadedOrderId(undefined);
      setLoadedOrganizationId(undefined);
      setLoadedTransportMode(undefined);
      setLoadedBusinessType(undefined);
      setErrorState(null);
      setShippingDocs([]);
      setPersonnel([]);
      setLoading(false);
      return;
    }

    if (isUserLoaded && !organizationId) {
      requestIdRef.current += 1;
      setOrder(undefined);
      setLoadedOrderId(undefined);
      setLoadedOrganizationId(undefined);
      setLoadedTransportMode(undefined);
      setLoadedBusinessType(undefined);
      setErrorState(null);
      setShippingDocs([]);
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
    const currentTransportMode = definition.transportMode;
    const currentBusinessType = businessType;
    setLoading(true);
    setErrorState(null);
    try {
      const [masterData, personnelOptRes, orderRes, docsRes, personnelRes] =
        await Promise.all([
          fetchOrderMasterData(organizationId, definition.transportMode),
          transportMode === 'sea'
            ? getOrderPersonnelOptions(organizationId, businessType)
            : Promise.resolve([]),
          orderServiceGetOrder({ id: orderId }),
          orderShippingDocumentServiceListShippingDocuments({ orderId }),
          orderPersonnelServiceListPersonnel({ orderId }),
        ]);

      if (
        currentRequestId !== requestIdRef.current ||
        currentOrderId !== activeOrderIdRef.current ||
        currentOrgId !== activeOrgIdRef.current ||
        currentTransportMode !== activeTransportModeRef.current ||
        currentBusinessType !== activeBusinessTypeRef.current
      ) {
        return;
      }

      const nextServiceTypeOptions =
        transportMode === 'sea'
          ? requireSeaServiceTypeOptions(masterData.serviceTypeOptions)
          : masterData.serviceTypeOptions;

      setServiceTypeOptions(nextServiceTypeOptions);
      setCargoCategoryOptions(masterData.cargoCategoryOptions);
      setLocationOptions(
        resolveOrderLocationOptions(definition.transportMode, masterData),
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
      setLoadedTransportMode(currentTransportMode);
      setLoadedBusinessType(currentBusinessType);
      setErrorState(null);
      setShippingDocs(unwrapList(docsRes));
      setPersonnel(unwrapList(personnelRes));
    } catch (err: any) {
      if (
        currentRequestId === requestIdRef.current &&
        currentOrderId === activeOrderIdRef.current &&
        currentOrgId === activeOrgIdRef.current &&
        currentTransportMode === activeTransportModeRef.current &&
        currentBusinessType === activeBusinessTypeRef.current
      ) {
        setOrder(undefined);
        setLoadedOrderId(undefined);
        setLoadedOrganizationId(undefined);
        setLoadedTransportMode(undefined);
        setLoadedBusinessType(undefined);
        setErrorState({
          organizationId: currentOrgId,
          orderId: currentOrderId,
          transportMode: currentTransportMode,
          businessType: currentBusinessType,
          error: err instanceof Error ? err : new Error(String(err)),
        });
        setShippingDocs([]);
        setPersonnel([]);
        message.error(err.message || '加载订单数据失败');
      }
    } finally {
      if (
        currentRequestId === requestIdRef.current &&
        currentOrderId === activeOrderIdRef.current &&
        currentOrgId === activeOrgIdRef.current &&
        currentTransportMode === activeTransportModeRef.current &&
        currentBusinessType === activeBusinessTypeRef.current
      ) {
        setLoading(false);
      }
    }
  }, [
    businessType,
    transportMode,
    definition,
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
      loadedTransportMode === transportMode &&
      loadedBusinessType === businessType,
  );
  const effectiveOrder = isOrderMatched ? order : undefined;
  const effectiveShippingDocs = isOrderMatched ? shippingDocs : [];
  const effectivePersonnel = isOrderMatched ? personnel : [];
  const effectiveLocationOptions = isOrderMatched ? locationOptions : [];

  const searchLocations = useCallback(
    async (keyword?: string) => {
      const requestOrgId = organizationId;
      const requestTransportMode = transportMode;
      const requestBusinessType = businessType;
      if (!requestOrgId || !requestTransportMode) {
        return [];
      }

      if (!keyword?.trim()) {
        return activeOrgIdRef.current === requestOrgId &&
          activeTransportModeRef.current === requestTransportMode &&
          activeBusinessTypeRef.current === requestBusinessType &&
          activeOrderIdRef.current === orderId &&
          loadedOrderId === orderId &&
          loadedOrganizationId === requestOrgId &&
          loadedTransportMode === requestTransportMode &&
          loadedBusinessType === requestBusinessType
          ? locationOptions
          : [];
      }

      const options = await searchOrderLocations(requestTransportMode, keyword);
      if (
        activeOrgIdRef.current !== requestOrgId ||
        activeTransportModeRef.current !== requestTransportMode ||
        activeBusinessTypeRef.current !== requestBusinessType ||
        activeOrderIdRef.current !== orderId
      ) {
        return [];
      }
      return options;
    },
    [
      transportMode,
      businessType,
      loadedTransportMode,
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
    errorState.transportMode === transportMode &&
    errorState.businessType === businessType
      ? errorState.error
      : null);
  const isPending =
    Boolean(definition && orderId && organizationId) &&
    !isOrderMatched &&
    !effectiveError;

  const effectiveLoading = !definition
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
