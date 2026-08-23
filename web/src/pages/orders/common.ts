import {
  masterDataServiceListAirports,
  masterDataServiceListOptions,
  masterDataServiceListPorts,
  masterDataServiceListStatusTemplates,
} from '@/services/roncin/masterDataService';
import { partnerServiceListPartners } from '@/services/roncin/partnerService';

export const businessTypeOptions = [
  { label: '海运出口', value: 1, color: 'blue' },
  { label: '海运进口', value: 2, color: 'cyan' },
  { label: '空运出口', value: 3, color: 'geekblue' },
  { label: '空运进口', value: 4, color: 'purple' },
  { label: '陆运', value: 5, color: 'green' },
  { label: '铁路', value: 6, color: 'volcano' },
];

export const businessTypeMap = new Map(
  businessTypeOptions.map((opt) => [opt.value, opt]),
);

export const businessTypeValueEnum: Record<
  number | string,
  { text: string }
> = Object.fromEntries(
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
  { label: '整箱 (FCL)', value: 1 },
  { label: '拼箱 (LCL)', value: 2 },
  { label: '散杂货 (Break Bulk)', value: 3 },
];

export const containerOwnershipOptions = [
  { label: '船东箱 (COC)', value: 1 },
  { label: '自备箱 (SOC)', value: 2 },
];

export const shipmentModeOptions = [
  { label: '拼货 (Consolidation)', value: 1 },
  { label: '跨境 (Cross Border)', value: 2 },
];

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
  BOOKING_AGENT: 3,
  CARRIER: 4,
} as const;

export type OrderKind = 'sea-export' | 'sea-import' | 'air-export' | 'air-import';

export interface OrderKindConfig {
  kind: OrderKind;
  businessType: number;
  tradeDirection: number;
  title: string;
  category: 'sea' | 'air';
  subTitle: string;
}

export const ORDER_KIND_CONFIGS: Record<string, OrderKindConfig> = {
  'sea-export': {
    kind: 'sea-export',
    businessType: 1,
    tradeDirection: 1,
    title: '海运出口订单',
    category: 'sea',
    subTitle: '全链路海运出口货代订单协同，统一管理状态流转、单证箱货与履约里程碑',
  },
  'sea-import': {
    kind: 'sea-import',
    businessType: 2,
    tradeDirection: 2,
    title: '海运进口订单',
    category: 'sea',
    subTitle: '全链路海运进口货代订单协同，统一管理状态流转、单证箱货与履约里程碑',
  },
  'air-export': {
    kind: 'air-export',
    businessType: 3,
    tradeDirection: 1,
    title: '空运出口订单',
    category: 'air',
    subTitle: '全链路空运出口货代订单协同，统一管理状态流转、单证箱货与履约里程碑',
  },
  'air-import': {
    kind: 'air-import',
    businessType: 4,
    tradeDirection: 2,
    title: '空运进口订单',
    category: 'air',
    subTitle: '全链路空运进口货代订单协同，统一管理状态流转、单证箱货与履约里程碑',
  },
};

export function parseOrderKind(pathnameOrKind?: string): OrderKindConfig | undefined {
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
): Promise<{ label: string; value: string }[]> {
  const res = await partnerServiceListPartners({
    role,
    enabled: true,
    keyword,
  });
  return (res.data ?? []).map((p) => ({
    label: p.legalName ? `${p.legalName} (${p.code})` : p.code || p.id || '',
    value: p.id ?? '',
  }));
}

export async function loadStatusTemplatesByBusinessType(
  businessType?: number,
): Promise<{ label: string; value: string }[]> {
  if (!businessType) {
    return [];
  }
  const res = await masterDataServiceListStatusTemplates({
    businessType,
    published: true,
  });
  const options = (res.data ?? [])
    .filter(
      (tpl) =>
        tpl.enabled !== false &&
        tpl.isDefault === true &&
        (tpl.items ?? []).some(
          (item) => item.code === 'DRAFT' && item.enabled !== false,
        ),
    )
    .map((tpl) => ({
      label: `${tpl.name} (v${tpl.version})`,
      value: tpl.id ?? '',
    }))
    .filter((option) => option.value !== '');

  if (options.length !== 1) {
    throw new Error('当前业务类型必须配置且只能配置一个默认状态流转模板');
  }
  return options;
}

export async function fetchOrderMasterData() {
  const [optionsResponse, portsResponse, airportsResponse] = await Promise.all([
    masterDataServiceListOptions(),
    masterDataServiceListPorts({ page: 1, pageSize: 100 }),
    masterDataServiceListAirports({ page: 1, pageSize: 100 }),
  ]);

  const masterOptions = optionsResponse.data ?? [];
  const ports = portsResponse.data ?? [];
  const airports = airportsResponse.data ?? [];

  const serviceTypeOptions = masterOptions
    .filter(
      (item) =>
        isMasterDataKind(item.kind, MASTER_DATA_KINDS.SERVICE_TYPE) &&
        item.enabled !== false,
    )
    .map((item) => ({
      label: item.name ?? '',
      value: item.id ?? '',
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

  return {
    masterOptions,
    ports,
    airports,
    serviceTypeOptions,
    cargoCategoryOptions,
    seaLocationOptions,
    airLocationOptions,
  };
}
