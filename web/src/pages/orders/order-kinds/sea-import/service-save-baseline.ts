import type { OrderDetailFormValues } from '../form-values';

/** 服务保存只推进本次版本，整票与海运单证仍使用各自原编辑基线。 */
export function acceptsImportServiceSaveBaseline(
  values: OrderDetailFormValues | undefined,
  previousVersion: string,
  saved: API.OrderImportHandling,
  order: API.Order | undefined,
  documents: API.SeaOrderDocuments | undefined,
  organizationId: string | undefined,
): boolean {
  if (
    !values?.id ||
    values.id !== saved.orderId ||
    order?.id !== values.id ||
    !organizationId ||
    order.organizationId !== organizationId ||
    values.version !== previousVersion ||
    !/^\d+$/.test(previousVersion) ||
    !saved.orderVersion ||
    !/^\d+$/.test(saved.orderVersion) ||
    BigInt(saved.orderVersion) !== BigInt(previousVersion) + 1n ||
    order.version !== saved.orderVersion ||
    documents?.orderId !== values.id
  )
    return false;
  const before = values.seaMasterBill as API.SeaMasterBillSummary | undefined;
  const after = order.seaMasterBill;
  if (!before?.masterBillId || !after?.masterBillId) return false;
  return (
    before.masterBillId === after.masterBillId &&
    values.seaMasterBillExpectedCandidateVersion != null &&
    String(values.seaMasterBillExpectedCandidateVersion) === after.version &&
    values.seaMasterBillContentVersion === documents.masterBill?.version &&
    documents.masterBill?.id === before.masterBillId &&
    before.transportExecutionId === after.transportExecutionId &&
    before.transportExecutionVersion === after.transportExecutionVersion &&
    values.seaDocumentStructure === documents.documentStructure &&
    values.seaDocumentLinkVersion === documents.linkVersion &&
    values.seaDocumentLinkVersion === order.seaDocumentLinkVersion &&
    values.seaHouseBill?.id === documents.houseBill?.id &&
    values.seaHouseBill?.expectedVersion === documents.houseBill?.version
  );
}
