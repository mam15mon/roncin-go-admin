import dayjs from 'dayjs';
import {
  ContainerOwnership,
  OrderPersonnelRole,
  ShipmentMode,
  ShipmentType,
} from '@/enums.generated';

export function buildInitialValues(
  order?: API.Order,
  shippingDocs: API.OrderShippingDocument[] = [],
  personnel: API.OrderPersonnel[] = [],
) {
  if (!order) return {};

  const personnelRoleMap: Record<
    number,
    { userId?: string; organizationId?: string }
  > = {};
  for (const p of personnel) {
    if (p.role !== undefined) {
      personnelRoleMap[p.role] = {
        userId: p.userId,
        organizationId: p.organizationId,
      };
    }
  }

  return {
    orderNo: order.orderNo,
    customerId: order.customerId,
    customerReferenceNo: order.customerReferenceNo,
    internalReferenceNo: order.internalReferenceNo,
    tradeTerm: order.tradeTerm,
    paymentTerm: order.paymentTerm,
    carrierId: order.carrierId,
    bookingAgentId: order.bookingAgentId,
    foreignAgentId: order.foreignAgentId,
    shippingAgentId: order.shippingAgentId,
    contractNo: order.contractNo,
    cargoValue: order.cargoValue,
    cargoCurrency: order.cargoCurrency || 'USD',
    insurancePremium: order.insurancePremium,
    insuranceCurrency: order.insuranceCurrency || 'CNY',
    loadingTerms: order.loadingTerms,
    shipmentType: order.shipmentType ?? ShipmentType.SHIPMENT_TYPE_FCL,
    containerOwnership:
      order.containerOwnership ??
      ContainerOwnership.CONTAINER_OWNERSHIP_COC,
    shipmentMode:
      order.shipmentMode ??
      ShipmentMode.SHIPMENT_MODE_TRADITIONAL_FORWARDING,
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
    customsCutoff: order.customsCutoff
      ? dayjs(order.customsCutoff)
      : undefined,
    vgmCutoff: order.vgmCutoff ? dayjs(order.vgmCutoff) : undefined,
    goodsDescription: order.goodsDescription,
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
    creatorOrganizationId:
      personnelRoleMap[OrderPersonnelRole.ORDER_PERSONNEL_ROLE_CREATOR]
        ?.organizationId,
    operatorUserId:
      personnelRoleMap[OrderPersonnelRole.ORDER_PERSONNEL_ROLE_OPERATOR]?.userId,
    operatorOrganizationId:
      personnelRoleMap[OrderPersonnelRole.ORDER_PERSONNEL_ROLE_OPERATOR]
        ?.organizationId,
    salesUserId:
      personnelRoleMap[OrderPersonnelRole.ORDER_PERSONNEL_ROLE_SALES]?.userId,
    salesOrganizationId:
      personnelRoleMap[OrderPersonnelRole.ORDER_PERSONNEL_ROLE_SALES]
        ?.organizationId,
    customerServiceUserId:
      personnelRoleMap[
        OrderPersonnelRole.ORDER_PERSONNEL_ROLE_CUSTOMER_SERVICE
      ]?.userId,
    customerServiceOrganizationId:
      personnelRoleMap[
        OrderPersonnelRole.ORDER_PERSONNEL_ROLE_CUSTOMER_SERVICE
      ]?.organizationId,
    documentUserId:
      personnelRoleMap[OrderPersonnelRole.ORDER_PERSONNEL_ROLE_DOCUMENT]?.userId,
    documentOrganizationId:
      personnelRoleMap[OrderPersonnelRole.ORDER_PERSONNEL_ROLE_DOCUMENT]
        ?.organizationId,
    commercialUserId:
      personnelRoleMap[OrderPersonnelRole.ORDER_PERSONNEL_ROLE_COMMERCIAL]
        ?.userId,
    commercialOrganizationId:
      personnelRoleMap[OrderPersonnelRole.ORDER_PERSONNEL_ROLE_COMMERCIAL]
        ?.organizationId,
    associateUserId:
      personnelRoleMap[OrderPersonnelRole.ORDER_PERSONNEL_ROLE_ASSOCIATE]?.userId,
    associateOrganizationId:
      personnelRoleMap[OrderPersonnelRole.ORDER_PERSONNEL_ROLE_ASSOCIATE]
        ?.organizationId,
    associate2UserId:
      personnelRoleMap[OrderPersonnelRole.ORDER_PERSONNEL_ROLE_ASSOCIATE2]
        ?.userId,
    associate2OrganizationId:
      personnelRoleMap[OrderPersonnelRole.ORDER_PERSONNEL_ROLE_ASSOCIATE2]
        ?.organizationId,
  };
}

export function buildUpdatePayload(
  orderId: string,
  orderVersion: string,
  values: any,
): API.UpdateOrderRequest {
  return {
    id: orderId,
    expectedVersion: orderVersion || '0',
    customerId: values.customerId,
    customerReferenceNo: values.customerReferenceNo?.trim() || undefined,
    internalReferenceNo: values.internalReferenceNo?.trim() || undefined,
    tradeTerm:
      values.tradeTerm !== undefined ? Number(values.tradeTerm) : undefined,
    paymentTerm:
      values.paymentTerm !== undefined
        ? Number(values.paymentTerm)
        : undefined,
    carrierId: values.carrierId || undefined,
    bookingAgentId: values.bookingAgentId || undefined,
    foreignAgentId: values.foreignAgentId || undefined,
    shippingAgentId: values.shippingAgentId || undefined,
    contractNo: values.contractNo?.trim() || undefined,
    cargoValue: values.cargoValue?.trim() || undefined,
    cargoCurrency: values.cargoCurrency || undefined,
    insurancePremium: values.insurancePremium?.trim() || undefined,
    insuranceCurrency: values.insuranceCurrency || undefined,
    loadingTerms: values.loadingTerms?.trim() || undefined,
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
    shippingDocuments: values.shippingDocuments
      ?.map((doc: any) => ({
        ...doc,
        masterNo: doc.masterNo?.trim() || '',
        houseNo: doc.houseNo?.trim() || '',
      }))
      .filter((doc: any) => doc.masterNo || doc.houseNo),
    containerRequests: values.containerRequests,
  };
}
