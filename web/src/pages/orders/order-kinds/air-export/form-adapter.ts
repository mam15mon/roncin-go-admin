import dayjs from 'dayjs';
import {
  OrderBusinessType,
  OrderPersonnelRole,
  TradeDirection,
} from '@/enums.generated';
import { getAirTemplateSections } from '../../templates/air-template';
import type {
  CreateOrderFormValues,
  OrderDetailFormValues,
} from '../form-values';
import type { OrderKindFormAdapter } from '../types';

export function serializeAirMaster(
  input?: API.AirMasterWaybillInput,
): API.AirMasterWaybillInput | undefined {
  if (!input) return undefined;
  const result = { ...input };
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
  | 'serviceTypeIds'
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

function commonFields(values: Partial<CommonValues>) {
  return {
    customerId: values.customerId,
    customerReferenceNo: values.customerReferenceNo?.trim() || undefined,
    internalReferenceNo: values.internalReferenceNo?.trim() || undefined,
    bookingNo: values.bookingNo?.trim() || undefined,
    paymentTerm: Number(values.paymentTerm || 1),
    tradeTerm: values.tradeTerm,
    bookingAgentId: values.bookingAgentId || undefined,
    foreignAgentId: values.foreignAgentId || undefined,
    contractNo: values.contractNo?.trim() || undefined,
    goodsDescription: values.goodsDescription?.trim() || undefined,
    specialRequirements: values.specialRequirements?.trim() || undefined,
    serviceTypeIds: values.serviceTypeIds,
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
    notes: values.notes?.trim() || undefined,
    bookingNotes: values.bookingNotes?.trim() || undefined,
    allocationNotes: values.allocationNotes?.trim() || undefined,
    operationNotes: values.operationNotes?.trim() || undefined,
  };
}

export const airExportFormAdapter: OrderKindFormAdapter = {
  buildSections: getAirTemplateSections,
  buildCreateDefaults: ({ creator, cargoCategoryOptions }) => ({
    orderDate: dayjs(),
    paymentTerm: 1,
    airMasterMode: 'NEW',
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
  buildCreatePayload: (values) => ({
    ...commonFields(values),
    customerId: values.customerId,
    businessType: OrderBusinessType.BUSINESS_TYPE_AE,
    tradeDirection: TradeDirection.TRADE_DIRECTION_EXPORT,
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
      master: serializeAirMaster(values.airMaster),
      houseNo: values.airHouseNo?.trim(),
      houseContent: values.airHouseContent,
      houseStatus: values.airHouseStatus,
    },
  }),
  buildDetailInitialValues: (order, _documents, personnel) => {
    if (!order) return {};
    const air = order.airExport;
    return {
      ...commonFields(order),
      orderNo: order.orderNo,
      customerId: order.customerId,
      orderDate: dayjs(order.orderDate || order.createdAt),
      // commonFields 序列化为 ISO 字符串；表单 DatePicker 需要 dayjs 对象回显。
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
      airHouseNo: air?.houseNo,
      airHouseContent: air?.houseContent,
      airHouseStatus: air?.houseStatus,
    };
  },
  buildUpdatePayload: (id, expectedVersion, values: OrderDetailFormValues) => ({
    id,
    expectedVersion,
    ...commonFields(values),
    airExport: {
      masterWaybillId: values.airMasterId,
      expectedMasterVersion: values.airMasterExpectedVersion,
      houseNo: values.airHouseNo?.trim(),
      houseContent: values.airHouseContent,
      houseStatus: values.airHouseStatus,
      expectedLinkVersion: values.airExport?.linkVersion,
      changeReason: values.airHouseChangeReason?.trim(),
    },
  }),
};
