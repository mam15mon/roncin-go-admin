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
import {
  ContainerOwnership,
  MasterDataKind,
  OrderBusinessType,
  OrderPersonnelRole,
  OrderShippingDocumentStatus,
  PartnerRoleType,
  PaymentTerm,
  ShipmentMode,
  ShipmentType,
  TradeDirection,
  TradeTerm,
} from '@/enums.generated';
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
  { label: '出口', value: TradeDirection.TRADE_DIRECTION_EXPORT },
  { label: '进口', value: TradeDirection.TRADE_DIRECTION_IMPORT },
];

export const tradeDirectionValueEnum: Record<
  number | string,
  { text: string }
> = Object.fromEntries(
  tradeDirectionOptions.map((opt) => [opt.value, { text: opt.label }]),
);

export const tradeTermOptions = [
  { label: 'EXW', value: TradeTerm.TRADE_TERM_EXW },
  { label: 'FCA', value: TradeTerm.TRADE_TERM_FCA },
  { label: 'FOB', value: TradeTerm.TRADE_TERM_FOB },
  { label: 'CFR', value: TradeTerm.TRADE_TERM_CFR },
  { label: 'CIF', value: TradeTerm.TRADE_TERM_CIF },
  { label: 'CPT', value: TradeTerm.TRADE_TERM_CPT },
  { label: 'CIP', value: TradeTerm.TRADE_TERM_CIP },
  { label: 'DAP', value: TradeTerm.TRADE_TERM_DAP },
  { label: 'DPU', value: TradeTerm.TRADE_TERM_DPU },
  { label: 'DDU', value: TradeTerm.TRADE_TERM_DDU },
  { label: 'DDP', value: TradeTerm.TRADE_TERM_DDP },
  { label: 'LDP', value: TradeTerm.TRADE_TERM_LDP },
];

export const paymentTermOptions = [
  { label: '预付 (PP)', value: PaymentTerm.PAYMENT_TERM_PREPAID },
  { label: '到付 (CC)', value: PaymentTerm.PAYMENT_TERM_COLLECT },
];

export const shipmentTypeOptions = [
  { label: '整箱', value: ShipmentType.SHIPMENT_TYPE_FCL },
  { label: '拼箱', value: ShipmentType.SHIPMENT_TYPE_LCL },
  { label: '散杂', value: ShipmentType.SHIPMENT_TYPE_BREAK_BULK },
];

export const containerOwnershipOptions = [
  { label: '船东箱 (COC)', value: ContainerOwnership.CONTAINER_OWNERSHIP_COC },
  { label: '自备箱 (SOC)', value: ContainerOwnership.CONTAINER_OWNERSHIP_SOC },
];

export const shipmentModeOptions = [
  {
    label: '集运',
    value: ShipmentMode.SHIPMENT_MODE_TRADITIONAL_FORWARDING,
  },
  { label: '跨境', value: ShipmentMode.SHIPMENT_MODE_CROSS_BORDER },
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
  { label: '创建人 (CREATOR)', value: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_CREATOR },
  { label: '操作专员 (OPERATOR)', value: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_OPERATOR },
  { label: '业务销售 (SALES)', value: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_SALES },
  { label: '客服专员 (CUSTOMER_SERVICE)', value: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_CUSTOMER_SERVICE },
  { label: '单证专员 (DOCUMENT)', value: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_DOCUMENT },
  { label: '商务采购 (COMMERCIAL)', value: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_COMMERCIAL },
  { label: '协同助理 (ASSOCIATE)', value: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_ASSOCIATE },
  { label: '副协同 (ASSOCIATE2)', value: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_ASSOCIATE2 },
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
  [OrderShippingDocumentStatus.ORDER_SHIPPING_DOCUMENT_STATUS_DRAFT]: { text: '草稿', status: 'Default' },
  [OrderShippingDocumentStatus.ORDER_SHIPPING_DOCUMENT_STATUS_CONFIRMED]: { text: '已确认', status: 'Processing' },
  [OrderShippingDocumentStatus.ORDER_SHIPPING_DOCUMENT_STATUS_RELEASED]: { text: '已放货', status: 'Success' },
};

export const MASTER_DATA_KINDS = {
  REGION: MasterDataKind.MASTER_DATA_KIND_REGION,
  CONTAINER_SPEC: MasterDataKind.MASTER_DATA_KIND_CONTAINER_SPEC,
  SERVICE_TYPE: MasterDataKind.MASTER_DATA_KIND_SERVICE_TYPE,
  CARGO_CATEGORY: MasterDataKind.MASTER_DATA_KIND_CARGO_CATEGORY,
} as const;

export function isMasterDataKind(
  value: number | undefined,
  kind: MasterDataKind,
) {
  return value === kind;
}

export const PARTNER_ROLES = {
  CUSTOMER: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
  SUPPLIER: PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER,
  FOREIGN_AGENT: PartnerRoleType.PARTNER_ROLE_TYPE_FOREIGN_AGENT,
  CARRIER: PartnerRoleType.PARTNER_ROLE_TYPE_CARRIER,
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
    businessType: OrderBusinessType.BUSINESS_TYPE_SE,
    tradeDirection: TradeDirection.TRADE_DIRECTION_EXPORT,
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
      kind: MASTER_DATA_KINDS.REGION,
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
