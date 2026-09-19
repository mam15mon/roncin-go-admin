import { useCallback, useEffect, useState } from 'react';
import { enterpriseResourceServiceListEnterpriseResourceRegionOptions } from '@/services/roncin/enterpriseResourceService';
import { unwrapList } from '@/utils/api';

/** 资源编辑弹窗内省市区级联候选项加载。 */
export function useRegionOptions({
  editorOpen,
  isAddressTab,
  countryCode,
  provinceCode,
  cityCode,
}: {
  editorOpen: boolean;
  isAddressTab: boolean;
  countryCode?: string;
  provinceCode?: string;
  cityCode?: string;
}) {
  const [provinceOptions, setProvinceOptions] = useState<
    { label: string; value: string }[]
  >([]);
  const [cityOptions, setCityOptions] = useState<
    { label: string; value: string }[]
  >([]);
  const [districtOptions, setDistrictOptions] = useState<
    { label: string; value: string }[]
  >([]);

  const loadRegions = useCallback(
    async (level: number, parentCode?: string) => {
      const response =
        await enterpriseResourceServiceListEnterpriseResourceRegionOptions({
          level,
          parentCode,
          page: 1,
          pageSize: 200,
        });
      return unwrapList(response).flatMap((item) =>
        item.code ? [{ value: item.code, label: item.name ?? item.code }] : [],
      );
    },
    [],
  );

  useEffect(() => {
    if (editorOpen && isAddressTab && countryCode === 'CN')
      void loadRegions(1).then(setProvinceOptions);
  }, [countryCode, editorOpen, isAddressTab, loadRegions]);
  useEffect(() => {
    if (editorOpen && provinceCode)
      void loadRegions(2, provinceCode).then(setCityOptions);
  }, [editorOpen, loadRegions, provinceCode]);
  useEffect(() => {
    if (editorOpen && cityCode)
      void loadRegions(3, cityCode).then(setDistrictOptions);
  }, [cityCode, editorOpen, loadRegions]);

  return { provinceOptions, cityOptions, districtOptions };
}
