import dayjs from 'dayjs';
import type { OrderKindConfig } from './common';

export type CreateOrderFormValues = {
  customerId: string;
  customerReferenceNo?: string;
  internalReferenceNo?: string;
  customerCode?: string;
  tradeTerm: number;
  paymentTerm: number;
  carrierId?: string;
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
  loadingTerms?: string;
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
  seaMasterBillIssuerPartnerId?: string;
  seaMasterBillCandidateId?: string;
  seaMasterBillExpectedCandidateVersion?: number | string;
  seaMasterBillCorrectionReason?: string;
  seaMasterBill?: API.SeaMasterBillInput;
  operatorUserId?: string;
  operatorOrganizationId?: string;
  salesUserId?: string;
  salesOrganizationId?: string;
  customerServiceUserId?: string;
  customerServiceOrganizationId?: string;
  associateUserId?: string;
  associateOrganizationId?: string;
  documentUserId?: string;
  documentOrganizationId?: string;
  commercialUserId?: string;
  commercialOrganizationId?: string;
  associate2UserId?: string;
  associate2OrganizationId?: string;
  creatorUserId?: string;
  creatorOrganizationId?: string;
};

/** 将新建订单表单值整理为创建请求 payload（含岗位人员装配）。 */
export function buildCreateOrderPayload(
  values: CreateOrderFormValues,
  config: OrderKindConfig,
): API.CreateOrderRequest {
  const personnelAssignments: API.OrderPersonnelAssignmentInput[] = [];
  const addPersonnel = (
    role: number,
    userId?: string,
    organizationId?: string,
  ) => {
    if (userId && organizationId) {
      personnelAssignments.push({ role, userId, organizationId });
    }
  };
  addPersonnel(2, values.operatorUserId, values.operatorOrganizationId);
  addPersonnel(3, values.salesUserId, values.salesOrganizationId);
  addPersonnel(
    4,
    values.customerServiceUserId,
    values.customerServiceOrganizationId,
  );
  addPersonnel(7, values.associateUserId, values.associateOrganizationId);
  addPersonnel(5, values.documentUserId, values.documentOrganizationId);
  addPersonnel(6, values.commercialUserId, values.commercialOrganizationId);
  addPersonnel(8, values.associate2UserId, values.associate2OrganizationId);

  let seaMasterBill: API.SeaMasterBillInput | undefined;
  if (values.seaMasterBill) {
    seaMasterBill = values.seaMasterBill;
  } else if (values.seaMasterBillMasterNo || values.seaMasterBillIssuerPartnerId) {
    seaMasterBill = {
      masterNo: values.seaMasterBillMasterNo || '',
      issuerPartnerId: values.seaMasterBillIssuerPartnerId || '',
      candidateId: values.seaMasterBillCandidateId || undefined,
      expectedCandidateVersion:
        values.seaMasterBillExpectedCandidateVersion !== undefined &&
        values.seaMasterBillExpectedCandidateVersion !== null
          ? String(values.seaMasterBillExpectedCandidateVersion)
          : undefined,
      correctionReason:
        values.seaMasterBillCorrectionReason?.trim() || undefined,
    };
  }

  return {
    customerId: values.customerId,
    customerReferenceNo: values.customerReferenceNo?.trim() || undefined,
    internalReferenceNo: values.internalReferenceNo?.trim() || undefined,
    businessType: config.businessType,
    tradeDirection: config.tradeDirection,
    tradeTerm: Number(values.tradeTerm),
    paymentTerm: Number(values.paymentTerm),
    carrierId: values.carrierId || undefined,
    bookingAgentId: values.bookingAgentId || undefined,
    foreignAgentId: values.foreignAgentId || undefined,
    shippingAgentId: values.shippingAgentId || undefined,
    contractNo: values.contractNo?.trim() || undefined,
    cargoValue: values.cargoValue?.trim() || undefined,
    cargoCurrency: values.cargoCurrency || undefined,
    insurancePremium: values.insurancePremium?.trim() || undefined,
    insuranceCurrency: values.insuranceCurrency || undefined,
    unNumber: values.unNumber?.trim() || undefined,
    hazardClass: values.hazardClass?.trim() || undefined,
    factoryName: values.factoryName?.trim() || undefined,
    cargoReadyAt: values.cargoReadyAt
      ? dayjs(values.cargoReadyAt).toISOString()
      : undefined,
    loadingTerms: values.loadingTerms?.trim() || undefined,
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
    shippingDocuments: values.shippingDocuments
      ?.map((doc) => ({
        id: doc.id,
        houseNo: doc.houseNo?.trim() || '',
        releaseType: doc.releaseType?.trim() || undefined,
        note: doc.note?.trim() || undefined,
      }))
      .filter((doc) => !!doc.houseNo),
    containerRequests: values.containerRequests,
    seaMasterBill,
  };
}
