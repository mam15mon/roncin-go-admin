import dayjs from 'dayjs';
import {
  ContainerOwnership,
  OrderBusinessType,
  OrderPersonnelRole,
  SeaDocumentStructure,
  ShipmentMode,
  ShipmentType,
  TradeDirection,
  TradeTerm,
} from '@/enums.generated';
import {
  recommendedServiceIDs,
  SEA_SHIPMENT_MODE,
} from '../../sea-order-policy';
import { getSeaTemplateSections } from '../../templates';
import type {
  OrderCreateDefaultsContext,
  OrderKindFormAdapter,
} from '../types';

/** 与 seaExportDefinition 元数据一致的海运出口稳定枚举。 */
const SEA_EXPORT_BUSINESS_TYPE = OrderBusinessType.BUSINESS_TYPE_SE;
const SEA_EXPORT_TRADE_DIRECTION = TradeDirection.TRADE_DIRECTION_EXPORT;

export type CreateOrderFormValues = {
  customerId: string;
  customerReferenceNo?: string;
  bookingNo?: string;
  internalReferenceNo?: string;
  customerCode?: string;
  tradeTerm?: number;
  paymentTerm: number;
  shippingLineId?: string;
  bookingAgentId?: string;
  foreignAgentId?: string;
  shippingAgentId?: string;
  contractNo?: string;
  cargoValue?: string;
  cargoCurrency?: string;
  insurancePremium?: string;
  insuranceCurrency?: string;
  unNumber?: string;
  hazardClass?: string;
  factoryName?: string;
  cargoReadyAt?: string | dayjs.Dayjs;
  declarationCutoffAt?: string | dayjs.Dayjs;
  receivedAt?: string | dayjs.Dayjs;
  shipmentType?: number;
  containerOwnership?: number;
  shipmentMode?: number;
  serviceTypeIds?: string[];
  cargoCategoryIds?: string[];
  originLocationId?: string;
  destinationLocationId?: string;
  dischargeLocationId?: string;
  transitLocationId?: string;
  vesselVoyage?: string;
  etd?: string | dayjs.Dayjs;
  eta?: string | dayjs.Dayjs;
  siCutoff?: string | dayjs.Dayjs;
  docCutoff?: string | dayjs.Dayjs;
  customsCutoff?: string | dayjs.Dayjs;
  vgmCutoff?: string | dayjs.Dayjs;
  goodsDescription?: string;
  totalPackages?: number;
  totalGrossWeightKg?: number;
  totalVolumeCbm?: number;
  totalPackageUnit?: string;
  specialRequirements?: string;
  orderDate?: string | dayjs.Dayjs;
  notes?: string;
  bookingNotes?: string;
  allocationNotes?: string;
  operationNotes?: string;
  shippingDocuments?: API.OrderShippingDocumentInput[];
  containerRequests?: API.OrderContainerRequestInput[];
  seaMasterBillMasterNo?: string;
  seaMasterBillCandidateId?: string;
  seaMasterBillExpectedCandidateVersion?: number | string;
  seaMasterBillCandidateTeId?: string;
  seaMasterBillExpectedCandidateTeVersion?: number | string;
  seaMasterBillCorrectionReason?: string;
  seaMasterBill?: API.SeaMasterBillInput;
  /** 命中共享主单批次时由候选响应写入的分单号清单（含作废），仅用于失焦即时排重提示，不提交。 */
  seaMasterBillBatchHouseNos?: string[];
  operatorUserId?: string;
  salesUserId?: string;
  customerServiceUserId?: string;
  associateUserId?: string;
  documentUserId?: string;
  commercialUserId?: string;
  associate2UserId?: string;
  creatorUserId?: string;
  seaDocumentStructure?: number;
  seaMasterBillContent?: API.SeaBillContent;
  seaHouseBill?: API.SeaHouseBillInput;
  seaDocument?: API.SeaOrderDocumentInput;
};

export type OrderDetailFormValues = Omit<
  CreateOrderFormValues,
  'containerRequests' | 'seaMasterBill' | 'shippingDocuments'
> & {
  containerRequests?: Array<
    API.OrderContainerRequest | API.OrderContainerRequestInput
  >;
  shippingDocuments?: Array<
    API.OrderShippingDocument | API.OrderShippingDocumentInput
  >;
  orderNo?: string;
  seaMasterBill?: API.SeaMasterBillInput | API.SeaMasterBillSummary;
  seaDocumentLinkVersion?: string;
  seaDocumentSummary?: API.SeaOrderDocumentSummary;
};

/** 计算海运出口新建默认值：只依赖显式传入的创建人与主数据候选项。 */
export function buildSeaExportCreateDefaults(
  context: OrderCreateDefaultsContext,
): Partial<CreateOrderFormValues> {
  const defaultCargoCategoryId =
    context.cargoCategoryOptions.find((item) => item.code === 'GENERAL')
      ?.value ??
    context.cargoCategoryOptions.find((item) => item.label === '普货')?.value;
  return {
    orderDate: dayjs(),
    shipmentMode: ShipmentMode.SHIPMENT_MODE_TRADITIONAL_FORWARDING,
    shipmentType: ShipmentType.SHIPMENT_TYPE_FCL,
    tradeTerm: TradeTerm.TRADE_TERM_CIF,
    serviceTypeIds: recommendedServiceIDs(
      context.serviceTypeOptions,
      SEA_SHIPMENT_MODE.TRADITIONAL_FORWARDING,
    ),
    cargoCategoryIds:
      typeof defaultCargoCategoryId === 'string'
        ? [defaultCargoCategoryId]
        : undefined,
    creatorUserId: context.creator?.userId,
  };
}

/** 将海运出口新建表单值整理为创建请求 payload（含岗位人员装配）。 */
export function buildSeaExportCreatePayload(
  values: CreateOrderFormValues,
): API.CreateOrderRequest {
  const personnelAssignments: API.OrderPersonnelAssignmentInput[] = [];
  const addPersonnel = (role: OrderPersonnelRole, userId?: string) => {
    if (userId) {
      personnelAssignments.push({ role, userId });
    }
  };
  addPersonnel(
    OrderPersonnelRole.ORDER_PERSONNEL_ROLE_OPERATOR,
    values.operatorUserId,
  );
  addPersonnel(
    OrderPersonnelRole.ORDER_PERSONNEL_ROLE_SALES,
    values.salesUserId,
  );
  addPersonnel(
    OrderPersonnelRole.ORDER_PERSONNEL_ROLE_CUSTOMER_SERVICE,
    values.customerServiceUserId,
  );
  addPersonnel(
    OrderPersonnelRole.ORDER_PERSONNEL_ROLE_ASSOCIATE,
    values.associateUserId,
  );
  addPersonnel(
    OrderPersonnelRole.ORDER_PERSONNEL_ROLE_DOCUMENT,
    values.documentUserId,
  );
  addPersonnel(
    OrderPersonnelRole.ORDER_PERSONNEL_ROLE_COMMERCIAL,
    values.commercialUserId,
  );
  addPersonnel(
    OrderPersonnelRole.ORDER_PERSONNEL_ROLE_ASSOCIATE2,
    values.associate2UserId,
  );

  const resolvedMasterNo =
    values.seaMasterBillMasterNo?.trim() ||
    values.seaMasterBill?.masterNo?.trim();

  let seaMasterBill: API.SeaMasterBillInput | undefined;
  if (resolvedMasterNo) {
    seaMasterBill = {
      ...(values.seaMasterBill || {}),
      masterNo: resolvedMasterNo,
      candidateId:
        values.seaMasterBillCandidateId ||
        values.seaMasterBill?.candidateId ||
        undefined,
      expectedCandidateVersion:
        values.seaMasterBillExpectedCandidateVersion !== undefined &&
        values.seaMasterBillExpectedCandidateVersion !== null
          ? String(values.seaMasterBillExpectedCandidateVersion)
          : values.seaMasterBill?.expectedCandidateVersion !== undefined &&
              values.seaMasterBill?.expectedCandidateVersion !== null
            ? String(values.seaMasterBill.expectedCandidateVersion)
            : undefined,
      candidateTeId:
        values.seaMasterBillCandidateTeId ||
        values.seaMasterBill?.candidateTeId ||
        undefined,
      expectedCandidateTeVersion:
        values.seaMasterBillExpectedCandidateTeVersion !== undefined &&
        values.seaMasterBillExpectedCandidateTeVersion !== null
          ? String(values.seaMasterBillExpectedCandidateTeVersion)
          : values.seaMasterBill?.expectedCandidateTeVersion !== undefined &&
              values.seaMasterBill?.expectedCandidateTeVersion !== null
            ? String(values.seaMasterBill.expectedCandidateTeVersion)
            : undefined,
      correctionReason:
        values.seaMasterBillCorrectionReason?.trim() ||
        values.seaMasterBill?.correctionReason?.trim() ||
        undefined,
    };
  }

  let seaDocument: API.SeaOrderDocumentInput | undefined;
  if (values.seaDocument) {
    seaDocument = values.seaDocument;
  } else {
    // DIRECT 不提交 HBL 隐藏草稿：切换模式保留的本地值只用于切回 HOUSE 时恢复。
    const houseBill =
      values.seaDocumentStructure ===
        SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE && values.seaHouseBill
        ? {
            id: values.seaHouseBill.id,
            houseNo: values.seaHouseBill.houseNo ?? '',
            issuerSource: values.seaHouseBill.issuerSource,
            issuerPartnerId: values.seaHouseBill.issuerPartnerId || undefined,
            note: values.seaHouseBill.note?.trim() || undefined,
            content: values.seaHouseBill.content,
          }
        : undefined;

    seaDocument = {
      documentStructure: values.seaDocumentStructure,
      masterBillContent: values.seaMasterBillContent,
      houseBill,
    };
  }

  return {
    customerId: values.customerId,
    customerReferenceNo: values.customerReferenceNo?.trim() || undefined,
    bookingNo: values.bookingNo?.trim() || undefined,
    internalReferenceNo: values.internalReferenceNo?.trim() || undefined,
    businessType: SEA_EXPORT_BUSINESS_TYPE,
    tradeDirection: SEA_EXPORT_TRADE_DIRECTION,
    tradeTerm:
      values.tradeTerm !== undefined && values.tradeTerm !== null
        ? Number(values.tradeTerm)
        : undefined,
    paymentTerm: Number(values.paymentTerm),
    shippingLineId: values.shippingLineId || undefined,
    bookingAgentId: values.bookingAgentId || undefined,
    foreignAgentId: values.foreignAgentId || undefined,
    shippingAgentId: values.shippingAgentId || undefined,
    contractNo: values.contractNo?.trim() || undefined,
    cargoValue:
      values.cargoValue?.trim() && values.cargoCurrency
        ? values.cargoValue.trim()
        : undefined,
    cargoCurrency:
      values.cargoValue?.trim() && values.cargoCurrency
        ? values.cargoCurrency
        : undefined,
    insurancePremium:
      values.insurancePremium?.trim() && values.insuranceCurrency
        ? values.insurancePremium.trim()
        : undefined,
    insuranceCurrency:
      values.insurancePremium?.trim() && values.insuranceCurrency
        ? values.insuranceCurrency
        : undefined,
    unNumber: values.unNumber?.trim() || undefined,
    hazardClass: values.hazardClass?.trim() || undefined,
    factoryName: values.factoryName?.trim() || undefined,
    cargoReadyAt: values.cargoReadyAt
      ? dayjs(values.cargoReadyAt).toISOString()
      : undefined,
    declarationCutoffAt: values.declarationCutoffAt
      ? dayjs(values.declarationCutoffAt).toISOString()
      : undefined,
    receivedAt: values.receivedAt
      ? dayjs(values.receivedAt).toISOString()
      : undefined,
    shipmentType:
      values.shipmentType !== undefined && values.shipmentType !== null
        ? Number(values.shipmentType)
        : undefined,
    containerOwnership:
      values.containerOwnership !== undefined &&
      values.containerOwnership !== null
        ? Number(values.containerOwnership)
        : undefined,
    shipmentMode:
      values.shipmentMode !== undefined && values.shipmentMode !== null
        ? Number(values.shipmentMode)
        : undefined,
    serviceTypeIds: values.serviceTypeIds,
    cargoCategoryIds: values.cargoCategoryIds,
    originLocationId: values.originLocationId || undefined,
    destinationLocationId: values.destinationLocationId || undefined,
    dischargeLocationId: values.dischargeLocationId || undefined,
    transitLocationId: values.transitLocationId || undefined,
    vesselVoyage: values.vesselVoyage?.trim() || undefined,
    etd: values.etd ? dayjs(values.etd).toISOString() : undefined,
    eta: values.eta ? dayjs(values.eta).toISOString() : undefined,
    siCutoff: values.siCutoff
      ? dayjs(values.siCutoff).toISOString()
      : undefined,
    docCutoff: values.docCutoff
      ? dayjs(values.docCutoff).toISOString()
      : undefined,
    customsCutoff: values.customsCutoff
      ? dayjs(values.customsCutoff).toISOString()
      : undefined,
    vgmCutoff: values.vgmCutoff
      ? dayjs(values.vgmCutoff).toISOString()
      : undefined,
    goodsDescription: values.goodsDescription?.trim() || undefined,
    totalPackages:
      values.totalPackages !== undefined && values.totalPackages !== null
        ? Number(values.totalPackages)
        : undefined,
    totalGrossWeightKg:
      values.totalGrossWeightKg !== undefined &&
      values.totalGrossWeightKg !== null
        ? Number(values.totalGrossWeightKg)
        : undefined,
    totalVolumeCbm:
      values.totalVolumeCbm !== undefined && values.totalVolumeCbm !== null
        ? Number(values.totalVolumeCbm)
        : undefined,
    totalPackageUnit: values.totalPackageUnit?.trim() || undefined,
    specialRequirements: values.specialRequirements?.trim() || undefined,
    orderDate: values.orderDate
      ? dayjs(values.orderDate).toISOString()
      : undefined,
    notes: values.notes?.trim() || undefined,
    bookingNotes: values.bookingNotes?.trim() || undefined,
    allocationNotes: values.allocationNotes?.trim() || undefined,
    operationNotes: values.operationNotes?.trim() || undefined,
    personnelAssignments,
    shippingDocuments: undefined,
    containerRequests: values.containerRequests,
    seaMasterBill,
    seaDocument,
  };
}

/** 将海运出口订单聚合与单证、人员映射为详情表单初始值。 */
export function buildSeaExportDetailInitialValues(
  order?: API.Order,
  shippingDocs: API.OrderShippingDocument[] = [],
  personnel: API.OrderPersonnel[] = [],
): Partial<OrderDetailFormValues> {
  if (!order) return {};

  const personnelRoleMap: Record<number, { userId?: string }> = {};
  for (const p of personnel) {
    if (p.role !== undefined) {
      personnelRoleMap[p.role] = {
        userId: p.userId,
      };
    }
  }

  return {
    orderNo: order.orderNo,
    customerId: order.customerId,
    customerReferenceNo: order.customerReferenceNo,
    bookingNo: order.bookingNo,
    internalReferenceNo: order.internalReferenceNo,
    tradeTerm: order.tradeTerm,
    paymentTerm: order.paymentTerm,
    shippingLineId: order.shippingLineId,
    bookingAgentId: order.bookingAgentId,
    foreignAgentId: order.foreignAgentId,
    shippingAgentId: order.shippingAgentId,
    contractNo: order.contractNo,
    cargoValue: order.cargoValue,
    cargoCurrency: order.cargoValue?.trim()
      ? order.cargoCurrency || 'USD'
      : undefined,
    insurancePremium: order.insurancePremium,
    insuranceCurrency: order.insurancePremium?.trim()
      ? order.insuranceCurrency || 'CNY'
      : undefined,
    unNumber: order.unNumber,
    hazardClass: order.hazardClass,
    factoryName: order.factoryName,
    cargoReadyAt: order.cargoReadyAt ? dayjs(order.cargoReadyAt) : undefined,
    declarationCutoffAt: order.declarationCutoffAt
      ? dayjs(order.declarationCutoffAt)
      : undefined,
    receivedAt: order.receivedAt ? dayjs(order.receivedAt) : undefined,
    shipmentType: order.shipmentType ?? ShipmentType.SHIPMENT_TYPE_FCL,
    containerOwnership:
      order.containerOwnership ?? ContainerOwnership.CONTAINER_OWNERSHIP_COC,
    shipmentMode:
      order.shipmentMode ?? ShipmentMode.SHIPMENT_MODE_TRADITIONAL_FORWARDING,
    serviceTypeIds: order.serviceTypeIds ?? [],
    cargoCategoryIds: order.cargoCategoryIds ?? [],
    originLocationId: order.originLocationId,
    destinationLocationId: order.destinationLocationId,
    dischargeLocationId: order.dischargeLocationId,
    transitLocationId: order.transitLocationId,
    vesselVoyage: order.vesselVoyage,
    etd: order.etd ? dayjs(order.etd) : undefined,
    eta: order.eta ? dayjs(order.eta) : undefined,
    siCutoff: order.siCutoff ? dayjs(order.siCutoff) : undefined,
    docCutoff: order.docCutoff ? dayjs(order.docCutoff) : undefined,
    customsCutoff: order.customsCutoff ? dayjs(order.customsCutoff) : undefined,
    vgmCutoff: order.vgmCutoff ? dayjs(order.vgmCutoff) : undefined,
    goodsDescription: order.goodsDescription,
    specialRequirements: order.specialRequirements,
    totalPackages: order.totalPackages,
    totalGrossWeightKg: order.totalGrossWeightKg,
    totalVolumeCbm: order.totalVolumeCbm,
    totalPackageUnit: order.totalPackageUnit || 'CTNS',
    orderDate: order.orderDate
      ? dayjs(order.orderDate)
      : dayjs(order.createdAt),
    notes: order.notes,
    bookingNotes: order.bookingNotes,
    allocationNotes: order.allocationNotes,
    operationNotes: order.operationNotes,
    shippingDocuments:
      shippingDocs.length > 0 ? shippingDocs : order.shippingDocuments,
    containerRequests: order.containerRequests,
    creatorUserId:
      personnelRoleMap[OrderPersonnelRole.ORDER_PERSONNEL_ROLE_CREATOR]?.userId,
    operatorUserId:
      personnelRoleMap[OrderPersonnelRole.ORDER_PERSONNEL_ROLE_OPERATOR]
        ?.userId,
    salesUserId:
      personnelRoleMap[OrderPersonnelRole.ORDER_PERSONNEL_ROLE_SALES]?.userId,
    customerServiceUserId:
      personnelRoleMap[OrderPersonnelRole.ORDER_PERSONNEL_ROLE_CUSTOMER_SERVICE]
        ?.userId,
    documentUserId:
      personnelRoleMap[OrderPersonnelRole.ORDER_PERSONNEL_ROLE_DOCUMENT]
        ?.userId,
    commercialUserId:
      personnelRoleMap[OrderPersonnelRole.ORDER_PERSONNEL_ROLE_COMMERCIAL]
        ?.userId,
    associateUserId:
      personnelRoleMap[OrderPersonnelRole.ORDER_PERSONNEL_ROLE_ASSOCIATE]
        ?.userId,
    associate2UserId:
      personnelRoleMap[OrderPersonnelRole.ORDER_PERSONNEL_ROLE_ASSOCIATE2]
        ?.userId,
    seaMasterBillMasterNo: order.seaMasterBill?.masterNo,
    seaMasterBillCandidateId: undefined,
    seaMasterBillExpectedCandidateVersion: order.seaMasterBill?.version,
    seaMasterBillCorrectionReason: undefined,
    seaMasterBill: order.seaMasterBill,
    seaDocumentStructure: order.seaDocumentStructure,
    seaDocumentLinkVersion: order.seaDocumentLinkVersion,
    seaDocumentSummary: order.seaDocumentSummary,
  };
}

/** 将海运出口详情表单值整理为更新请求 payload。 */
export function buildSeaExportUpdatePayload(
  orderId: string,
  orderVersion: string,
  values: OrderDetailFormValues,
): API.UpdateOrderRequest {
  return {
    id: orderId,
    expectedVersion: orderVersion || '0',
    customerId: values.customerId,
    customerReferenceNo: values.customerReferenceNo?.trim() || undefined,
    bookingNo: values.bookingNo?.trim() || undefined,
    internalReferenceNo: values.internalReferenceNo?.trim() || undefined,
    tradeTerm:
      values.tradeTerm !== undefined ? Number(values.tradeTerm) : undefined,
    paymentTerm:
      values.paymentTerm !== undefined ? Number(values.paymentTerm) : undefined,
    shippingLineId: values.shippingLineId || undefined,
    bookingAgentId: values.bookingAgentId || undefined,
    foreignAgentId: values.foreignAgentId || undefined,
    shippingAgentId: values.shippingAgentId || undefined,
    contractNo: values.contractNo?.trim() || undefined,
    cargoValue:
      values.cargoValue?.trim() && values.cargoCurrency
        ? values.cargoValue.trim()
        : undefined,
    cargoCurrency:
      values.cargoValue?.trim() && values.cargoCurrency
        ? values.cargoCurrency
        : undefined,
    insurancePremium:
      values.insurancePremium?.trim() && values.insuranceCurrency
        ? values.insurancePremium.trim()
        : undefined,
    insuranceCurrency:
      values.insurancePremium?.trim() && values.insuranceCurrency
        ? values.insuranceCurrency
        : undefined,
    unNumber: values.unNumber?.trim() || undefined,
    hazardClass: values.hazardClass?.trim() || undefined,
    factoryName: values.factoryName?.trim() || undefined,
    cargoReadyAt: values.cargoReadyAt
      ? dayjs(values.cargoReadyAt).toISOString()
      : undefined,
    declarationCutoffAt: values.declarationCutoffAt
      ? dayjs(values.declarationCutoffAt).toISOString()
      : undefined,
    receivedAt: values.receivedAt
      ? dayjs(values.receivedAt).toISOString()
      : undefined,
    shipmentType:
      values.shipmentType !== undefined
        ? Number(values.shipmentType)
        : undefined,
    containerOwnership:
      values.containerOwnership !== undefined
        ? Number(values.containerOwnership)
        : undefined,
    shipmentMode:
      values.shipmentMode !== undefined
        ? Number(values.shipmentMode)
        : undefined,
    serviceTypeIds: values.serviceTypeIds,
    cargoCategoryIds: values.cargoCategoryIds,
    originLocationId: values.originLocationId || undefined,
    destinationLocationId: values.destinationLocationId || undefined,
    dischargeLocationId: values.dischargeLocationId || undefined,
    transitLocationId: values.transitLocationId || undefined,
    vesselVoyage: values.vesselVoyage?.trim() || undefined,
    etd: values.etd ? dayjs(values.etd).toISOString() : undefined,
    eta: values.eta ? dayjs(values.eta).toISOString() : undefined,
    siCutoff: values.siCutoff
      ? dayjs(values.siCutoff).toISOString()
      : undefined,
    docCutoff: values.docCutoff
      ? dayjs(values.docCutoff).toISOString()
      : undefined,
    customsCutoff: values.customsCutoff
      ? dayjs(values.customsCutoff).toISOString()
      : undefined,
    vgmCutoff: values.vgmCutoff
      ? dayjs(values.vgmCutoff).toISOString()
      : undefined,
    goodsDescription: values.goodsDescription?.trim() || undefined,
    specialRequirements: values.specialRequirements?.trim() || undefined,
    totalPackages:
      values.totalPackages !== undefined
        ? Number(values.totalPackages)
        : undefined,
    totalGrossWeightKg:
      values.totalGrossWeightKg !== undefined
        ? Number(values.totalGrossWeightKg)
        : undefined,
    totalVolumeCbm:
      values.totalVolumeCbm !== undefined
        ? Number(values.totalVolumeCbm)
        : undefined,
    totalPackageUnit: values.totalPackageUnit?.trim() || undefined,
    notes: values.notes?.trim() || undefined,
    bookingNotes: values.bookingNotes?.trim() || undefined,
    allocationNotes: values.allocationNotes?.trim() || undefined,
    operationNotes: values.operationNotes?.trim() || undefined,
    shippingDocuments: undefined,
    containerRequests: values.containerRequests
      ?.filter(
        (request) =>
          typeof request.containerSpecId === 'string' &&
          request.containerSpecId !== '' &&
          typeof request.quantity === 'number',
      )
      .map((request) => ({
        id: request.id,
        containerSpecId: request.containerSpecId as string,
        quantity: request.quantity as number,
      })),
    seaMasterBill: values.seaMasterBillMasterNo?.trim()
      ? {
          masterNo: values.seaMasterBillMasterNo.trim(),
          candidateId: values.seaMasterBillCandidateId || undefined,
          expectedCandidateVersion:
            values.seaMasterBillExpectedCandidateVersion !== undefined &&
            values.seaMasterBillExpectedCandidateVersion !== null
              ? String(values.seaMasterBillExpectedCandidateVersion)
              : undefined,
          correctionReason:
            values.seaMasterBillCorrectionReason?.trim() || undefined,
        }
      : undefined,
    seaDocument: values.seaDocument,
  };
}

export const seaExportFormAdapter: OrderKindFormAdapter = {
  buildSections: getSeaTemplateSections,
  buildCreateDefaults: buildSeaExportCreateDefaults,
  buildCreatePayload: buildSeaExportCreatePayload,
  buildDetailInitialValues: buildSeaExportDetailInitialValues,
  buildUpdatePayload: buildSeaExportUpdatePayload,
};
