import dayjs from 'dayjs';
import {
  OrderBusinessType,
  OrderPersonnelRole,
  ShipmentMode,
  TradeDirection,
} from '@/enums.generated';
import { getAirTemplateSections } from '../../templates/air-template';
import type {
  CreateOrderFormValues,
  OrderDetailFormValues,
} from '../form-values';
import type { OrderKindFormAdapter } from '../types';

/** 仅清理声明控件注册产生的空子消息，其他零值及正文显式清空原样保留。 */
export function serializeAirWaybillContent(
  content?: API.AirWaybillContent,
): API.AirWaybillContent | undefined {
  if (content === undefined) return undefined;
  const result = { ...content };
  for (const field of [
    'carriageValue',
    'customsValue',
    'insuranceValue',
  ] as const) {
    const declaration = result[field];
    if (declaration && !declaration.kind && !declaration.amount) {
      delete result[field];
    }
  }
  for (const field of ['prepaid', 'collect'] as const) {
    const summary = result[field];
    if (
      summary &&
      Object.values(summary).every((value) => value == null || value === '')
    ) {
      delete result[field];
    }
  }
  return result;
}

export function serializeAirMaster(
  input?: API.AirMasterWaybillInput,
): API.AirMasterWaybillInput | undefined {
  if (!input) return undefined;
  const result = {
    ...input,
    waybillContent: serializeAirWaybillContent(input.waybillContent),
  };
  for (const key of [
    'etd',
    'eta',
    'actualDepartureAt',
    'actualArrivalAt',
    'documentCutoffAt',
    'customsCutoffAt',
  ] as const) {
    result[key] = input[key] ? dayjs(input[key]).toISOString() : '';
  }
  return result;
}

const personnelFields = [
  ['creatorUserId', OrderPersonnelRole.ORDER_PERSONNEL_ROLE_CREATOR],
  ['operatorUserId', OrderPersonnelRole.ORDER_PERSONNEL_ROLE_OPERATOR],
  ['salesUserId', OrderPersonnelRole.ORDER_PERSONNEL_ROLE_SALES],
  [
    'customerServiceUserId',
    OrderPersonnelRole.ORDER_PERSONNEL_ROLE_CUSTOMER_SERVICE,
  ],
  ['documentUserId', OrderPersonnelRole.ORDER_PERSONNEL_ROLE_DOCUMENT],
  ['commercialUserId', OrderPersonnelRole.ORDER_PERSONNEL_ROLE_COMMERCIAL],
  ['associateUserId', OrderPersonnelRole.ORDER_PERSONNEL_ROLE_ASSOCIATE],
  ['associate2UserId', OrderPersonnelRole.ORDER_PERSONNEL_ROLE_ASSOCIATE2],
] as const;

type CommonValues = Pick<
  CreateOrderFormValues,
  | 'shipmentMode'
  | 'cargoValue'
  | 'cargoCurrency'
  | 'insurancePremium'
  | 'insuranceCurrency'
  | 'receivedAt'
  | 'cargoReadyAt'
  | 'factoryName'
  | 'customerId'
  | 'customerReferenceNo'
  | 'internalReferenceNo'
  | 'bookingNo'
  | 'paymentTerm'
  | 'tradeTerm'
  | 'bookingAgentId'
  | 'foreignAgentId'
  | 'contractNo'
  | 'goodsDescription'
  | 'specialRequirements'
  | 'cargoCategoryIds'
  | 'unNumber'
  | 'hazardClass'
  | 'dangerousPackingGroup'
  | 'declarationCutoffAt'
  | 'totalPackages'
  | 'totalGrossWeightKg'
  | 'totalVolumeCbm'
  | 'totalPackageUnit'
  | 'notes'
  | 'bookingNotes'
  | 'allocationNotes'
  | 'operationNotes'
>;

/** 更新是整份表单提交：可清经营字段显式携带空值，避免 optional 省略保留旧值。 */
function commonFields(values: Partial<CommonValues>, clearEmpty = false) {
  const text = (value?: string) =>
    value?.trim() || (clearEmpty ? '' : undefined);
  const time = (value?: CommonValues['receivedAt']) =>
    value ? dayjs(value).toISOString() : clearEmpty ? '' : undefined;
  return {
    shipmentMode: values.shipmentMode ?? (clearEmpty ? 0 : undefined),
    cargoValue: text(values.cargoValue),
    cargoCurrency: text(values.cargoCurrency),
    insurancePremium: text(values.insurancePremium),
    insuranceCurrency: text(values.insuranceCurrency),
    factoryName: text(values.factoryName),
    receivedAt: time(values.receivedAt),
    cargoReadyAt: time(values.cargoReadyAt),
    customerId: values.customerId,
    customerReferenceNo: text(values.customerReferenceNo),
    internalReferenceNo: text(values.internalReferenceNo),
    bookingNo: text(values.bookingNo),
    paymentTerm: Number(values.paymentTerm || 1),
    tradeTerm: values.tradeTerm ?? (clearEmpty ? 0 : undefined),
    bookingAgentId: values.bookingAgentId || (clearEmpty ? '' : undefined),
    foreignAgentId: values.foreignAgentId || (clearEmpty ? '' : undefined),
    contractNo: text(values.contractNo),
    goodsDescription: text(values.goodsDescription),
    specialRequirements: text(values.specialRequirements),
    cargoCategoryIds: values.cargoCategoryIds,
    unNumber: values.unNumber?.trim() || undefined,
    hazardClass: values.hazardClass?.trim() || undefined,
    // UNSPECIFIED(0) 视同未选：未选 / 不适用（Class 1/2/7）都不携带。
    dangerousPackingGroup: values.dangerousPackingGroup || undefined,
    declarationCutoffAt: values.declarationCutoffAt
      ? dayjs(values.declarationCutoffAt).toISOString()
      : undefined,
    totalPackages: values.totalPackages,
    totalGrossWeightKg: values.totalGrossWeightKg,
    totalVolumeCbm: values.totalVolumeCbm,
    totalPackageUnit: values.totalPackageUnit,
    notes: text(values.notes),
    bookingNotes: text(values.bookingNotes),
    allocationNotes: text(values.allocationNotes),
    operationNotes: text(values.operationNotes),
  };
}

/** AE/AI 共用经营与单证映射，方向只由品类注册项显式传入。 */
export function buildAirCreatePayload(
  values: CreateOrderFormValues,
  businessType: OrderBusinessType,
  tradeDirection: TradeDirection,
): API.CreateOrderRequest {
  return {
    ...commonFields(values),
    customerId: values.customerId,
    businessType,
    tradeDirection,
    orderDate: values.orderDate
      ? dayjs(values.orderDate).toISOString()
      : undefined,
    personnelAssignments: personnelFields.flatMap(([field, role]) =>
      role !== OrderPersonnelRole.ORDER_PERSONNEL_ROLE_CREATOR && values[field]
        ? [{ role, userId: String(values[field]) }]
        : [],
    ),
    airExport: {
      masterWaybillId:
        values.airMasterMode === 'EXISTING' ? values.airMasterId : undefined,
      expectedMasterVersion:
        values.airMasterMode === 'EXISTING'
          ? values.airMasterExpectedVersion
          : undefined,
      master:
        values.airMasterMode === 'EXISTING'
          ? undefined
          : serializeAirMaster(values.airMaster),
      documentMode: values.airDocumentMode,
      house:
        values.airDocumentMode === 'HOUSE'
          ? {
              houseNo: values.airHouseNo?.trim(),
              content: serializeAirWaybillContent(values.airHouseContent),
              status: values.airHouseStatus,
            }
          : undefined,
    },
  };
}

export const airExportFormAdapter: OrderKindFormAdapter = {
  buildSections: getAirTemplateSections,
  buildCreateDefaults: ({ creator, cargoCategoryOptions }) => ({
    orderDate: dayjs(),
    paymentTerm: 1,
    shipmentMode: ShipmentMode.SHIPMENT_MODE_TRADITIONAL_FORWARDING,
    airMasterMode: 'NEW',
    airDocumentMode: 'HOUSE',
    creatorUserId: creator?.userId,
    totalPackageUnit: 'CTNS',
    cargoCategoryIds: cargoCategoryOptions.find(
      (item) => item.code === 'GENERAL',
    )?.value
      ? [
          String(
            cargoCategoryOptions.find((item) => item.code === 'GENERAL')?.value,
          ),
        ]
      : [],
  }),
  buildCreatePayload: (values) =>
    buildAirCreatePayload(
      values,
      OrderBusinessType.BUSINESS_TYPE_AE,
      TradeDirection.TRADE_DIRECTION_EXPORT,
    ),
  buildDetailInitialValues: (order, _documents, personnel) => {
    if (!order) return {};
    const air = order.airExport;
    return {
      ...commonFields(order),
      orderNo: order.orderNo,
      customerId: order.customerId,
      orderDate: dayjs(order.orderDate || order.createdAt),
      // commonFields 序列化为 ISO 字符串；表单 DatePicker 需要 dayjs 对象回显。
      receivedAt: order.receivedAt ? dayjs(order.receivedAt) : undefined,
      cargoReadyAt: order.cargoReadyAt ? dayjs(order.cargoReadyAt) : undefined,
      declarationCutoffAt: order.declarationCutoffAt
        ? dayjs(order.declarationCutoffAt)
        : undefined,
      ...Object.fromEntries(
        personnelFields.map(([field, role]) => [
          field,
          personnel.find((item) => item.role === role)?.userId,
        ]),
      ),
      airExport: air,
      airMaster: air?.master?.content,
      airMasterMode: 'EXISTING',
      airMasterId: air?.master?.id,
      airMasterExpectedVersion: air?.master?.version,
      airDocumentMode:
        air?.documentMode === 'DIRECT' || air?.documentMode === 'HOUSE'
          ? air.documentMode
          : undefined,
      airHouseNo: air?.house?.houseNo,
      airHouseContent: air?.house?.content,
      airHouseStatus: air?.house?.status,
    };
  },
  buildUpdatePayload: (id, expectedVersion, values: OrderDetailFormValues) => ({
    id,
    expectedVersion,
    ...commonFields(values, true),
    airExport: {
      masterWaybillId: values.airExport?.master?.id,
      expectedMasterVersion: values.airExport?.master?.version,
      // 已有订单的关系身份取打开时的基线，表单不能借普通保存切换模式或主单。
      documentMode: values.airExport?.documentMode,
      house:
        values.airExport?.documentMode === 'HOUSE'
          ? {
              houseNo: values.airHouseNo?.trim(),
              content: serializeAirWaybillContent(values.airHouseContent),
              status: values.airHouseStatus,
              expectedVersion: values.airExport.house?.version,
            }
          : undefined,
      expectedLinkVersion: values.airExport?.linkVersion,
      changeReason: values.airHouseChangeReason?.trim(),
    },
  }),
};
