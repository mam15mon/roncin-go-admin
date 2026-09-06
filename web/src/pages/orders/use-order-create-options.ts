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
  type OrderKindConfig,
  requireSeaServiceTypeOptions,
  searchOrderLocations,
} from './common';
import type { SelectOption } from './templates';

type CreateOptionsRequestIdentity = {
  organizationId: string;
  category: OrderKindConfig['category'];
  businessType: OrderKindConfig['businessType'];
};

type CreateOptionsErrorState = CreateOptionsRequestIdentity & {
  error: Error;
};

/** 新建订单页的主数据与人员候选项加载。 */
export function useOrderCreateOptions(config?: OrderKindConfig) {
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
  const activeCategoryRef = useRef(config?.category);
  activeCategoryRef.current = config?.category;
  const activeBusinessTypeRef = useRef(config?.businessType);
  activeBusinessTypeRef.current = config?.businessType;
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
    if (!config) {
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
    const currentCategory = config.category;
    const currentBusinessType = config.businessType;
    setLoading(true);
    setErrorState(null);

    Promise.all([
      fetchOrderMasterData(organizationId, config.category),
      config.category === 'sea'
        ? getOrderPersonnelOptions(organizationId, config.businessType)
        : Promise.resolve([]),
    ])
      .then(([masterData, personnelResponse]) => {
        if (
          currentRequestId !== requestIdRef.current ||
          currentOrgId !== activeOrgIdRef.current ||
          currentCategory !== activeCategoryRef.current ||
          currentBusinessType !== activeBusinessTypeRef.current
        ) {
          return;
        }

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
        setPersonnelOptions(personnelResponse);
        setLoadedIdentity({
          organizationId: currentOrgId,
          category: currentCategory,
          businessType: currentBusinessType,
        });
        setErrorState(null);
      })
      .catch((err: Error) => {
        if (
          currentRequestId !== requestIdRef.current ||
          currentOrgId !== activeOrgIdRef.current ||
          currentCategory !== activeCategoryRef.current ||
          currentBusinessType !== activeBusinessTypeRef.current
        ) {
          return;
        }
        setLoadedIdentity(null);
        setErrorState({
          organizationId: currentOrgId,
          category: currentCategory,
          businessType: currentBusinessType,
          error: err,
        });
        message.error(err.message || '加载订单主数据失败');
      })
      .finally(() => {
        if (
          currentRequestId === requestIdRef.current &&
          currentOrgId === activeOrgIdRef.current &&
          currentCategory === activeCategoryRef.current &&
          currentBusinessType === activeBusinessTypeRef.current
        ) {
          setLoading(false);
        }
      });
  }, [config, isUserLoaded, message, organizationId, reloadKey]);

  const isIdentityMatched = Boolean(
    organizationId &&
      config &&
      loadedIdentity?.organizationId === organizationId &&
      loadedIdentity.category === config.category &&
      loadedIdentity.businessType === config.businessType,
  );
  const effectiveLocationOptions = isIdentityMatched ? locationOptions : [];

  const searchLocations = useCallback(
    async (keyword?: string) => {
      const requestOrgId = organizationId;
      const requestCategory = config?.category;
      const requestBusinessType = config?.businessType;
      if (!requestOrgId || !requestCategory || requestBusinessType == null) {
        return [];
      }

      if (!keyword?.trim()) {
        return activeOrgIdRef.current === requestOrgId &&
          activeCategoryRef.current === requestCategory &&
          activeBusinessTypeRef.current === requestBusinessType &&
          loadedIdentity?.organizationId === requestOrgId &&
          loadedIdentity.category === requestCategory &&
          loadedIdentity.businessType === requestBusinessType
          ? locationOptions
          : [];
      }

      const options = await searchOrderLocations(requestCategory, keyword);
      if (
        activeOrgIdRef.current !== requestOrgId ||
        activeCategoryRef.current !== requestCategory ||
        activeBusinessTypeRef.current !== requestBusinessType
      ) {
        return [];
      }
      return options;
    },
    [
      config?.businessType,
      config?.category,
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
    config &&
    errorState?.organizationId === organizationId &&
    errorState.category === config.category &&
    errorState.businessType === config.businessType
      ? errorState.error
      : null);
  const effectiveLoading = !config
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
