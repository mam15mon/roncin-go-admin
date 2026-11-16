import { useQuery } from '@tanstack/react-query';
import { enterpriseResourceServiceListEnterpriseResourceRegionOptions } from '@/services/roncin/enterpriseResourceService';
import { ensureListResponse, unwrapList } from '@/utils/api';

/** 资源编辑弹窗的地区候选按国家和父级隔离，关闭弹窗时不加载。 */
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
  const active = editorOpen && isAddressTab && countryCode === 'CN';
  const optionsFor = (level: number, parentCode?: string) => ({
    queryKey: [
      'enterprise-resources',
      'region-options',
      {
        countryCode,
        level,
        parentCode,
        provinceCode: level === 3 ? provinceCode : undefined,
      },
    ],
    enabled: active && (level === 1 || Boolean(parentCode)),
    meta: { errorMessage: '行政区划候选加载失败' },
    queryFn: async ({ signal }: { signal: AbortSignal }) => {
      const response =
        await enterpriseResourceServiceListEnterpriseResourceRegionOptions(
          { level, parentCode, page: 1, pageSize: 200 },
          { signal, skipErrorHandler: true },
        );
      return unwrapList(
        ensureListResponse(response, '行政区划候选加载失败'),
      ).flatMap((item) =>
        item.code ? [{ value: item.code, label: item.name ?? item.code }] : [],
      );
    },
  });
  const provinces = useQuery(optionsFor(1));
  const cities = useQuery(optionsFor(2, provinceCode));
  const districts = useQuery(optionsFor(3, cityCode));

  return {
    provinceOptions: active ? (provinces.data ?? []) : [],
    cityOptions: active && provinceCode ? (cities.data ?? []) : [],
    districtOptions: active && cityCode ? (districts.data ?? []) : [],
  };
}
