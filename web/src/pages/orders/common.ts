import {
  masterDataServiceListAirports,
  masterDataServiceListItems,
  masterDataServiceListOptions,
  masterDataServiceListPorts,
} from '@/services/roncin/masterDataService';
import {
  businessTypeMeta,
  makeValueEnum,
  statusText,
} from '@/constants/statusMeta';
import { OrderBusinessType } from '@/enums.generated';
import { unwrapList } from '@/utils/api';
import { getCurrencies, searchPartnerOptions } from '@/utils/options';

export const businessTypeOptions = [
  {
    label: statusText(
      businessTypeMeta,
      OrderBusinessType.BUSINESS_TYPE_SE,
    ),
    value: OrderBusinessType.BUSINESS_TYPE_SE,
    color: 'blue',
  },
];

export const businessTypeMap = new Map(
  businessTypeOptions.map((opt) => [opt.value, opt]),
);

export const businessTypeValueEnum: Record<number | string, { text: string }> =
  makeValueEnum(
    Object.fromEntries(
      businessTypeOptions.map((option) => [
        option.value,
        businessTypeMeta[option.value],
      ]),
    ),
  );

export const tradeDirectionOptions = [
  { label: '出口', value: 1 },
  { label: '进口', value: 2 },
];

export const tradeDirectionValueEnum: Record<
  number | string,
  { text: string }
> = Object.fromEntries(
  tradeDirectionOptions.map((opt) => [opt.value, { text: opt.label }]),
);

export const tradeTermOptions = [
  { label: 'EXW', value: 1 },
  { label: 'FCA', value: 2 },
  { label: 'FOB', value: 3 },
  { label: 'CFR', value: 4 },
  { label: 'CIF', value: 5 },
  { label: 'CPT', value: 6 },
  { label: 'CIP', value: 7 },
  { label: 'DAP', value: 8 },
  { label: 'DPU', value: 9 },
  { label: 'DDU', value: 10 },
  { label: 'DDP', value: 11 },
  { label: 'LDP', value: 12 },
];

export const paymentTermOptions = [
  { label: '预付 (PP)', value: 1 },
  { label: '到付 (CC)', value: 2 },
];

export const shipmentTypeOptions = [
  { label: '整箱', value: 1 },
  { label: '拼箱', value: 2 },
  { label: '散杂', value: 3 },
];

export const containerOwnershipOptions = [
  { label: '船东箱 (COC)', value: 1 },
  { label: '自备箱 (SOC)', value: 2 },
];

export const shipmentModeOptions = [
  { label: '集运', value: 1 },
  { label: '跨境', value: 2 },
];

export const loadingTermsOptions = [
  { label: 'CY-CY', value: 'CY-CY' },
  { label: 'CY-CFS', value: 'CY-CFS' },
  { label: 'CFS-CY', value: 'CFS-CY' },
  { label: 'CFS-CFS', value: 'CFS-CFS' },
  { label: 'DOOR-CY', value: 'DOOR-CY' },
  { label: 'CY-DOOR', value: 'CY-DOOR' },
  { label: 'DOOR-DOOR', value: 'DOOR-DOOR' },
];

export const seaServiceTypes = [
  { code: 'BOOKING', name: '订舱' },
  { code: 'TRUCKING', name: '拖车' },
  { code: 'STUFFING', name: '内装' },
  { code: 'CUSTOMS_EXPORT', name: '报关' },
  { code: 'CUSTOMS_IMPORT', name: '清关' },
  { code: 'OVERSEA_SEGMENT', name: '海外段' },
  { code: 'INSURANCE', name: '保险' },
  { code: 'PALLET_CHARTER', name: '包板' },
  { code: 'CONTAINER_LEASE', name: '租箱' },
  { code: 'FUMIGATION', name: '熏蒸' },
  { code: 'DOC_BUY', name: '买单' },
  { code: 'CERTIFICATE', name: '办证' },
  { code: 'DOC_PREP', name: '制单' },
  { code: 'DANGEROUS_SERVICE', name: '危险品' },
  { code: 'OVERWEIGHT_SERVICE', name: '超重' },
  { code: 'DOCUMENT_EXCHANGE', name: '换单' },
  { code: 'WAREHOUSING', name: '仓储' },
  { code: 'INSPECTION', name: '报检' },
  { code: 'CONTAINER_PURCHASE', name: '买箱' },
] as const;

export function requireSeaServiceTypeOptions<
  T extends { code?: string; label: string; value: string | number },
>(options: T[]): T[] {
  return seaServiceTypes.map(({ code, name }) => {
    const option = options.find((item) => item.code === code);
    if (!option) {
      throw new Error(`缺少海运服务类型主数据：${name}（${code}）`);
    }
    return option;
  });
}

export const orderPersonnelRoleOptions = [
  { label: '创建人 (CREATOR)', value: 1 },
  { label: '操作专员 (OPERATOR)', value: 2 },
  { label: '业务销售 (SALES)', value: 3 },
  { label: '客服专员 (CUSTOMER_SERVICE)', value: 4 },
  { label: '单证专员 (DOCUMENT)', value: 5 },
  { label: '商务采购 (COMMERCIAL)', value: 6 },
  { label: '协同助理 (ASSOCIATE)', value: 7 },
  { label: '副协同 (ASSOCIATE2)', value: 8 },
];

export const orderPersonnelRoleValueEnum: Record<
  number | string,
  { text: string }
> = Object.fromEntries(
  orderPersonnelRoleOptions.map((opt) => [opt.value, { text: opt.label }]),
);

export const shippingDocumentStatusValueEnum: Record<
  number,
  { text: string; status: 'Default' | 'Processing' | 'Success' }
> = {
  1: { text: '草稿', status: 'Default' },
  2: { text: '已确认', status: 'Processing' },
  3: { text: '已放货', status: 'Success' },
};

export const MASTER_DATA_KINDS = {
  REGION: 'MASTER_DATA_KIND_REGION',
  CONTAINER_SPEC: 'MASTER_DATA_KIND_CONTAINER_SPEC',
  SERVICE_TYPE: 'MASTER_DATA_KIND_SERVICE_TYPE',
  CARGO_CATEGORY: 'MASTER_DATA_KIND_CARGO_CATEGORY',
} as const;

export function isMasterDataKind(
  value: number | string | undefined,
  kind: (typeof MASTER_DATA_KINDS)[keyof typeof MASTER_DATA_KINDS],
) {
  return value === kind;
}

export const PARTNER_ROLES = {
  CUSTOMER: 1,
  SUPPLIER: 2,
  FOREIGN_AGENT: 3,
  CARRIER: 4,
} as const;

export type OrderKind = 'sea-export';

export interface OrderKindConfig {
  kind: OrderKind;
  businessType: number;
  tradeDirection: number;
  title: string;
  category: 'sea' | 'air';
}

export const ORDER_KIND_CONFIGS: Record<string, OrderKindConfig> = {
  'sea-export': {
    kind: 'sea-export',
    businessType: 1,
    tradeDirection: 1,
    title: '海运出口订单',
    category: 'sea',
  },
};

export function parseOrderKind(
  pathnameOrKind?: string,
): OrderKindConfig | undefined {
  if (!pathnameOrKind) return undefined;
  if (ORDER_KIND_CONFIGS[pathnameOrKind]) {
    return ORDER_KIND_CONFIGS[pathnameOrKind];
  }
  const match = pathnameOrKind.match(/\/orders\/([^/]+)/);
  if (match && ORDER_KIND_CONFIGS[match[1]]) {
    return ORDER_KIND_CONFIGS[match[1]];
  }
  return undefined;
}

export async function searchPartnersByRole(
  role: number,
  keyword?: string,
): Promise<{ label: string; value: string; code?: string }[]> {
  return searchPartnerOptions(keyword, { role, enabled: true });
}

export async function searchOrderLocations(
  category: 'sea' | 'air',
  keyword?: string,
): Promise<{ label: string; value: string }[]> {
  const [regionsResponse, transportResponse] = await Promise.all([
    masterDataServiceListItems({
      kind: 3,
      keyword,
      enabled: true,
      page: 1,
      pageSize: 50,
    }),
    category === 'sea'
      ? masterDataServiceListPorts({ keyword, enabled: true, page: 1, pageSize: 50 })
      : masterDataServiceListAirports({ keyword, enabled: true, page: 1, pageSize: 50 }),
  ]);
  const regions = unwrapList(regionsResponse).map((item) => ({
    label: item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
    value: item.id ?? '',
  }));
  const transportLocations =
    category === 'sea'
      ? (transportResponse.data as API.Port[] | undefined)?.map((item) => ({
          label: `${item.nameZh ? `${item.nameZh} / ` : ''}${item.nameEn} (${item.unLocode})`,
          value: item.id ?? '',
        })) ?? []
      : (transportResponse.data as API.Airport[] | undefined)?.map((item) => ({
          label: `${item.nameZh ? `${item.nameZh} / ` : ''}${item.nameEn} (${item.iataCode})`,
          value: item.id ?? '',
        })) ?? [];
  return [...regions, ...transportLocations].filter((item) => item.value !== '');
}

export async function fetchOrderMasterData() {
  const [optionsResponse, portsResponse, airportsResponse, currencies] =
    await Promise.all([
      masterDataServiceListOptions(),
      masterDataServiceListPorts({ page: 1, pageSize: 50, enabled: true }),
      masterDataServiceListAirports({ page: 1, pageSize: 50, enabled: true }),
      getCurrencies(),
    ]);

  const masterOptions = unwrapList(optionsResponse);
  const ports = unwrapList(portsResponse);
  const airports = unwrapList(airportsResponse);
  const serviceTypeOptions = masterOptions
    .filter(
      (item) =>
        isMasterDataKind(item.kind, MASTER_DATA_KINDS.SERVICE_TYPE) &&
        item.enabled !== false,
    )
    .map((item) => ({
      label: item.name ?? '',
      value: item.id ?? '',
      code: item.code,
    }));

  const cargoCategoryOptions = masterOptions
    .filter(
      (item) =>
        isMasterDataKind(item.kind, MASTER_DATA_KINDS.CARGO_CATEGORY) &&
        item.enabled !== false,
    )
    .map((item) => ({
      label: item.name ?? '',
      value: item.id ?? '',
      code: item.code,
    }));

  const regionOptions = masterOptions
    .filter(
      (item) =>
        isMasterDataKind(item.kind, MASTER_DATA_KINDS.REGION) &&
        item.enabled !== false,
    )
    .map((item) => ({
      label: item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      value: item.id ?? '',
    }));

  const portOptions = ports
    .filter((item) => item.enabled !== false)
    .map((item) => ({
      label: `${item.nameZh ? `${item.nameZh} / ` : ''}${item.nameEn} (${item.unLocode})`,
      value: item.id ?? '',
    }));

  const airportOptions = airports
    .filter((item) => item.enabled !== false)
    .map((item) => ({
      label: `${item.nameZh ? `${item.nameZh} / ` : ''}${item.nameEn} (${item.iataCode})`,
      value: item.id ?? '',
    }));

  const seaLocationOptions = [...regionOptions, ...portOptions];
  const airLocationOptions = [...regionOptions, ...airportOptions];
  const currencyOptions = currencies
    .filter((item) => item.enabled !== false)
    .map((item) => ({
      label: `${item.code} - ${item.name}`,
      value: item.code ?? '',
    }))
    .filter((item) => item.value !== '');

  return {
    masterOptions,
    ports,
    airports,
    currencies,
    serviceTypeOptions,
    cargoCategoryOptions,
    seaLocationOptions,
    airLocationOptions,
    currencyOptions,
  };
}
