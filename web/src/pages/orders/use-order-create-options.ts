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

/** 新建订单页的主数据与人员候选项加载。 */
export function useOrderCreateOptions(config?: OrderKindConfig) {
  const { message } = App.useApp();
  const { initialState } = useModel('@@initialState');
  const organizationId = initialState?.currentUser?.currentOrganization?.id;
  const isUserLoaded = Boolean(initialState?.currentUser);

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  const [reloadKey, setReloadKey] = useState(0);
  const [loadedOrganizationId, setLoadedOrganizationId] = useState<
    string | null
  >(null);

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
  const requestIdRef = useRef(0);

  const retry = useCallback(() => {
    if (organizationId) {
      clearOrderMasterDataCache(organizationId);
    }
    setLoadedOrganizationId(null);
    setReloadKey((k) => k + 1);
  }, [organizationId]);

  useEffect(() => {
    if (!config) {
      setLoadedOrganizationId(null);
      setLoading(false);
      setError(null);
      return;
    }

    if (isUserLoaded && !organizationId) {
      setLoadedOrganizationId(null);
      setLoading(false);
      setError(new Error('缺少当前组织，无法加载订单主数据'));
      return;
    }

    if (!organizationId) {
      setLoadedOrganizationId(null);
      setLoading(true);
      return;
    }

    const currentRequestId = ++requestIdRef.current;
    const currentOrgId = organizationId;
    setLoading(true);
    setError(null);

    Promise.all([
      fetchOrderMasterData(organizationId, config.category),
      config.category === 'sea'
        ? getOrderPersonnelOptions(organizationId, config.businessType)
        : Promise.resolve([]),
    ])
      .then(([masterData, personnelResponse]) => {
        if (
          currentRequestId !== requestIdRef.current ||
          currentOrgId !== activeOrgIdRef.current
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
        setLoadedOrganizationId(currentOrgId);
        setError(null);
      })
      .catch((err: Error) => {
        if (
          currentRequestId !== requestIdRef.current ||
          currentOrgId !== activeOrgIdRef.current
        ) {
          return;
        }
        setLoadedOrganizationId(null);
        setError(err);
        message.error(err.message || '加载订单主数据失败');
      })
      .finally(() => {
        if (
          currentRequestId === requestIdRef.current &&
          currentOrgId === activeOrgIdRef.current
        ) {
          setLoading(false);
        }
      });
  }, [
    config,
    isUserLoaded,
    message,
    organizationId,
    reloadKey,
  ]);

  const searchLocations = useCallback(
    (keyword?: string) =>
      searchOrderLocations(config?.category === 'air' ? 'air' : 'sea', keyword),
    [config?.category],
  );

  const isOrgMatched = Boolean(
    organizationId && loadedOrganizationId === organizationId,
  );
  const effectiveLoading = !config
    ? false
    : isUserLoaded && !organizationId
      ? false
      : error
        ? false
        : !isOrgMatched || loading;

  return {
    loading: effectiveLoading,
    error,
    retry,
    serviceTypeOptions: isOrgMatched ? serviceTypeOptions : [],
    cargoCategoryOptions: isOrgMatched ? cargoCategoryOptions : [],
    locationOptions: isOrgMatched ? locationOptions : [],
    searchLocations,
    currencyOptions: isOrgMatched ? currencyOptions : [],
    containerSpecOptions: isOrgMatched ? containerSpecOptions : [],
    personnelOptions: isOrgMatched ? personnelOptions : [],
  };
}
