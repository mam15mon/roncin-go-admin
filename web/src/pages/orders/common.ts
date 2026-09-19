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
import {
  masterDataServiceListAirports,
  masterDataServiceListItems,
  masterDataServiceListPorts,
} from '@/services/roncin/masterDataService';
import { unwrapList } from '@/utils/api';
import { getCurrencies, searchPartnerOptions } from '@/utils/options';
import {
  getCachedAirports,
  getCachedPorts,
  getMasterDataOptions,
} from '@/utils/order-options-cache';
import type { OrderTransportMode } from './order-kinds/types';
import type { SelectOption } from './templates';

export const businessTypeOptions = [
  {
    label: statusText(businessTypeMeta, OrderBusinessType.BUSINESS_TYPE_SE),
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
  {
    label: '创建人 (CREATOR)',
    value: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_CREATOR,
  },
  {
    label: '操作专员 (OPERATOR)',
    value: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_OPERATOR,
  },
  {
    label: '业务销售 (SALES)',
    value: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_SALES,
  },
  {
    label: '客服专员 (CUSTOMER_SERVICE)',
    value: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_CUSTOMER_SERVICE,
  },
  {
    label: '单证专员 (DOCUMENT)',
    value: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_DOCUMENT,
  },
  {
    label: '商务采购 (COMMERCIAL)',
    value: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_COMMERCIAL,
  },
  {
    label: '协同助理 (ASSOCIATE)',
    value: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_ASSOCIATE,
  },
  {
    label: '副协同 (ASSOCIATE2)',
    value: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_ASSOCIATE2,
  },
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
  [OrderShippingDocumentStatus.ORDER_SHIPPING_DOCUMENT_STATUS_DRAFT]: {
    text: '草稿',
    status: 'Default',
  },
  [OrderShippingDocumentStatus.ORDER_SHIPPING_DOCUMENT_STATUS_CONFIRMED]: {
    text: '已确认',
    status: 'Processing',
  },
  [OrderShippingDocumentStatus.ORDER_SHIPPING_DOCUMENT_STATUS_RELEASED]: {
    text: '已放货',
    status: 'Success',
  },
};

export const MASTER_DATA_KINDS = {
  REGION: MasterDataKind.MASTER_DATA_KIND_REGION,
  CONTAINER_SPEC: MasterDataKind.MASTER_DATA_KIND_CONTAINER_SPEC,
  SERVICE_TYPE: MasterDataKind.MASTER_DATA_KIND_CHARGE_CATEGORY,
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
} as const;

export async function searchPartnersByRole(
  role: number,
  keyword?: string,
): Promise<{ label: string; value: string; code?: string }[]> {
  return searchPartnerOptions(keyword, { role, enabled: true });
}

/** 陆运/铁路的地点与站点主数据尚未开放；共享代码必须显式关闭，不得静默落入 sea/air 实现。 */
export function isUnimplementedTransportMode(
  transportMode: OrderTransportMode,
): boolean {
  return transportMode === 'land' || transportMode === 'rail';
}

function assertTransportStationsSupported(
  transportMode: OrderTransportMode,
): asserts transportMode is 'sea' | 'air' {
  switch (transportMode) {
    case 'sea':
    case 'air':
      return;
    case 'land':
    case 'rail':
      throw new Error('陆运与铁路订单的地点主数据尚未开放');
    default: {
      // 新运输方式加入 OrderTransportMode 时未更新本分发会在此编译报错。
      const unsupported: never = transportMode;
      throw new Error(`未支持的运输方式：${String(unsupported)}`);
    }
  }
}

export async function searchOrderLocations(
  transportMode: OrderTransportMode,
  keyword?: string,
): Promise<{ label: string; value: string }[]> {
  assertTransportStationsSupported(transportMode);
  const [regionsResponse, transportResponse] = await Promise.all([
    masterDataServiceListItems({
      kind: MASTER_DATA_KINDS.REGION,
      keyword,
      enabled: true,
      page: 1,
      pageSize: 50,
    }),
    transportMode === 'sea'
      ? masterDataServiceListPorts({
          keyword,
          enabled: true,
          page: 1,
          pageSize: 50,
        })
      : masterDataServiceListAirports({
          keyword,
          enabled: true,
          page: 1,
          pageSize: 50,
        }),
  ]);
  const regions = unwrapList(regionsResponse).map((item) => ({
    label: item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
    value: item.id ?? '',
  }));
  const transportLocations =
    transportMode === 'sea'
      ? ((transportResponse.data as API.Port[] | undefined)?.map((item) => ({
          label: `${item.nameZh ? `${item.nameZh} / ` : ''}${item.nameEn} (${item.unLocode})`,
          value: item.id ?? '',
        })) ?? [])
      : ((transportResponse.data as API.Airport[] | undefined)?.map((item) => ({
          label: `${item.nameZh ? `${item.nameZh} / ` : ''}${item.nameEn} (${item.iataCode})`,
          value: item.id ?? '',
        })) ?? []);
  return [...regions, ...transportLocations].filter(
    (item) => item.value !== '',
  );
}

export async function fetchOrderMasterData(
  organizationId: string,
  transportMode: OrderTransportMode,
) {
  assertTransportStationsSupported(transportMode);
  const shouldLoadPorts = transportMode === 'sea';
  const shouldLoadAirports = transportMode === 'air';

  const [masterOptions, ports, airports, currencies] = await Promise.all([
    getMasterDataOptions(organizationId),
    shouldLoadPorts ? getCachedPorts(organizationId) : Promise.resolve([]),
    shouldLoadAirports
      ? getCachedAirports(organizationId)
      : Promise.resolve([]),
    getCurrencies(),
  ]);
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

/** 按运输方式穷尽选择地点候选项；land/rail 显式抛错，不会静默进入机场分支。 */
export function resolveOrderLocationOptions(
  transportMode: OrderTransportMode,
  masterData: Awaited<ReturnType<typeof fetchOrderMasterData>>,
): SelectOption[] {
  assertTransportStationsSupported(transportMode);
  return transportMode === 'sea'
    ? masterData.seaLocationOptions
    : masterData.airLocationOptions;
}
