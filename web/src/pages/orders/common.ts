import {
  masterDataServiceListAirports,
  masterDataServiceListCurrencies,
  masterDataServiceListOptions,
  masterDataServiceListPorts,
} from '@/services/roncin/masterDataService';
import { partnerServiceListPartners } from '@/services/roncin/partnerService';

export const businessTypeOptions = [
  { label: '海运出口', value: 1, color: 'blue' },
];

export const businessTypeMap = new Map(
  businessTypeOptions.map((opt) => [opt.value, opt]),
);

export const businessTypeValueEnum: Record<number | string, { text: string }> =
  Object.fromEntries(
    businessTypeOptions.map((opt) => [opt.value, { text: opt.label }]),
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

export const seaServiceTypeNames = [
  '订舱',
  '拖车',
  '内装',
  '报关',
  '清关',
  '海外段',
  '保险',
  '租箱',
  '熏蒸',
  '买单',
  '办证',
  '制单',
  '危险品',
  '超重',
  '仓储',
  '买箱',
] as const;

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
  BOOKING_AGENT: 2,
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
  const res = await partnerServiceListPartners({
    role,
    enabled: true,
    keyword,
    page: 1,
    pageSize: 200,
  });
  return (res.data ?? []).map((p) => ({
    label: p.legalName ? `${p.legalName} (${p.code})` : p.code || p.id || '',
    value: p.id ?? '',
    code: p.code,
  }));
}

export async function fetchOrderMasterData() {
  const [optionsResponse, portsResponse, airportsResponse, currenciesResponse] =
    await Promise.all([
      masterDataServiceListOptions(),
      masterDataServiceListPorts({ page: 1, pageSize: 200 }),
      masterDataServiceListAirports({ page: 1, pageSize: 200 }),
      masterDataServiceListCurrencies({ page: 1, pageSize: 200 }),
    ]);

  const masterOptions = optionsResponse.data ?? [];
  const ports = portsResponse.data ?? [];
  const airports = airportsResponse.data ?? [];
  const currencies = currenciesResponse.data ?? [];

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
