import { useModel } from '@umijs/max';
import { App } from 'antd';
import { useCallback, useEffect, useRef, useState } from 'react';
import {
  clearOrderMasterDataCache,
  getOrderPersonnelOptions,
} from '@/utils/order-options-cache';
import {
  fetchOrderMasterData,
  isMasterDataKind,
  MASTER_DATA_KINDS,
  requireSeaServiceTypeOptions,
  searchOrderLocations,
} from './common';
import type { OrderKindDefinition } from './order-kinds/types';
import type { SelectOption } from './templates';

type CreateOptionsRequestIdentity = {
  organizationId: string;
  transportMode: OrderKindDefinition['transportMode'];
  businessType: OrderKindDefinition['businessType'];
};

type CreateOptionsErrorState = CreateOptionsRequestIdentity & {
  error: Error;
};

/** 新建订单页的主数据与人员候选项加载。 */
export function useOrderCreateOptions(definition?: OrderKindDefinition) {
  const { message } = App.useApp();
  const { initialState } = useModel('@@initialState');
  const organizationId = initialState?.currentUser?.currentOrganization?.id;
  const isUserLoaded = Boolean(initialState?.currentUser);

  const [loading, setLoading] = useState(true);
  const [errorState, setErrorState] = useState<CreateOptionsErrorState | null>(
    null,
  );
  const [reloadKey, setReloadKey] = useState(0);
  const [loadedIdentity, setLoadedIdentity] =
    useState<CreateOptionsRequestIdentity | null>(null);

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

  const activeOrgIdRef = useRef(organizationId);
  activeOrgIdRef.current = organizationId;
  const activeTransportModeRef = useRef(definition?.transportMode);
  activeTransportModeRef.current = definition?.transportMode;
  const activeBusinessTypeRef = useRef(definition?.businessType);
  activeBusinessTypeRef.current = definition?.businessType;
  const requestIdRef = useRef(0);

  const retry = useCallback(() => {
    if (organizationId) {
      clearOrderMasterDataCache(organizationId);
    }
    setLoadedIdentity(null);
    setErrorState(null);
    setReloadKey((k) => k + 1);
  }, [organizationId]);

  useEffect(() => {
    if (!definition) {
      requestIdRef.current += 1;
      setLoadedIdentity(null);
      setLoading(false);
      setErrorState(null);
      return;
    }

    if (isUserLoaded && !organizationId) {
      requestIdRef.current += 1;
      setLoadedIdentity(null);
      setLoading(false);
      setErrorState(null);
      return;
    }

    if (!organizationId) {
      requestIdRef.current += 1;
      setLoadedIdentity(null);
      setLoading(true);
      setErrorState(null);
      return;
    }

    const currentRequestId = ++requestIdRef.current;
    const currentOrgId = organizationId;
    const currentTransportMode = definition.transportMode;
    const currentBusinessType = definition.businessType;
    setLoading(true);
    setErrorState(null);

    Promise.all([
      fetchOrderMasterData(organizationId, definition.transportMode),
      definition.transportMode === 'sea'
        ? getOrderPersonnelOptions(organizationId, definition.businessType)
        : Promise.resolve([]),
    ])
      .then(([masterData, personnelResponse]) => {
        if (
          currentRequestId !== requestIdRef.current ||
          currentOrgId !== activeOrgIdRef.current ||
          currentTransportMode !== activeTransportModeRef.current ||
          currentBusinessType !== activeBusinessTypeRef.current
        ) {
          return;
        }

        const nextServiceTypeOptions =
          definition.transportMode === 'sea'
            ? requireSeaServiceTypeOptions(masterData.serviceTypeOptions)
            : masterData.serviceTypeOptions;

        setServiceTypeOptions(nextServiceTypeOptions);
        setCargoCategoryOptions(masterData.cargoCategoryOptions);
        setLocationOptions(
          definition.transportMode === 'sea'
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
        setPersonnelOptions(personnelResponse);
        setLoadedIdentity({
          organizationId: currentOrgId,
          transportMode: currentTransportMode,
          businessType: currentBusinessType,
        });
        setErrorState(null);
      })
      .catch((err: Error) => {
        if (
          currentRequestId !== requestIdRef.current ||
          currentOrgId !== activeOrgIdRef.current ||
          currentTransportMode !== activeTransportModeRef.current ||
          currentBusinessType !== activeBusinessTypeRef.current
        ) {
          return;
        }
        setLoadedIdentity(null);
        setErrorState({
          organizationId: currentOrgId,
          transportMode: currentTransportMode,
          businessType: currentBusinessType,
          error: err,
        });
        message.error(err.message || '加载订单主数据失败');
      })
      .finally(() => {
        if (
          currentRequestId === requestIdRef.current &&
          currentOrgId === activeOrgIdRef.current &&
          currentTransportMode === activeTransportModeRef.current &&
          currentBusinessType === activeBusinessTypeRef.current
        ) {
          setLoading(false);
        }
      });
  }, [definition, isUserLoaded, message, organizationId, reloadKey]);

  const isIdentityMatched = Boolean(
    organizationId &&
      definition &&
      loadedIdentity?.organizationId === organizationId &&
      loadedIdentity.transportMode === definition.transportMode &&
      loadedIdentity.businessType === definition.businessType,
  );
  const effectiveLocationOptions = isIdentityMatched ? locationOptions : [];

  const searchLocations = useCallback(
    async (keyword?: string) => {
      const requestOrgId = organizationId;
      const requestTransportMode = definition?.transportMode;
      const requestBusinessType = definition?.businessType;
      if (
        !requestOrgId ||
        !requestTransportMode ||
        requestBusinessType == null
      ) {
        return [];
      }

      if (!keyword?.trim()) {
        return activeOrgIdRef.current === requestOrgId &&
          activeTransportModeRef.current === requestTransportMode &&
          activeBusinessTypeRef.current === requestBusinessType &&
          loadedIdentity?.organizationId === requestOrgId &&
          loadedIdentity.transportMode === requestTransportMode &&
          loadedIdentity.businessType === requestBusinessType
          ? locationOptions
          : [];
      }

      const options = await searchOrderLocations(requestTransportMode, keyword);
      if (
        activeOrgIdRef.current !== requestOrgId ||
        activeTransportModeRef.current !== requestTransportMode ||
        activeBusinessTypeRef.current !== requestBusinessType
      ) {
        return [];
      }
      return options;
    },
    [
      definition?.businessType,
      definition?.transportMode,
      loadedIdentity,
      locationOptions,
      organizationId,
    ],
  );

  const missingOrgError =
    isUserLoaded && !organizationId
      ? new Error('缺少当前组织，无法加载订单主数据')
      : null;
  const effectiveError =
    missingOrgError ||
    (organizationId &&
    definition &&
    errorState?.organizationId === organizationId &&
    errorState.transportMode === definition.transportMode &&
    errorState.businessType === definition.businessType
      ? errorState.error
      : null);
  const effectiveLoading = !definition
    ? false
    : isUserLoaded && !organizationId
      ? false
      : effectiveError
        ? false
        : !isIdentityMatched || loading;

  return {
    loading: effectiveLoading,
    error: effectiveError,
    retry,
    serviceTypeOptions: isIdentityMatched ? serviceTypeOptions : [],
    cargoCategoryOptions: isIdentityMatched ? cargoCategoryOptions : [],
    locationOptions: effectiveLocationOptions,
    searchLocations,
    currencyOptions: isIdentityMatched ? currencyOptions : [],
    containerSpecOptions: isIdentityMatched ? containerSpecOptions : [],
    personnelOptions: isIdentityMatched ? personnelOptions : [],
  };
}
