import { OrderBusinessType } from '@/enums.generated';
import type { OrderDetailFormValues } from '../form-values';

/** 服务保存只推进本次版本，整票与海运单证仍使用各自原编辑基线。 */
export function acceptsImportServiceSaveBaseline(
  values: OrderDetailFormValues | undefined,
  previousVersion: string,
  saved: API.OrderImportHandling,
  order: API.Order | undefined,
  documents: API.SeaOrderDocuments | undefined,
  organizationId: string | undefined,
  businessType: number,
): boolean {
  if (
    !values?.id ||
    values.id !== saved.orderId ||
    order?.id !== values.id ||
    !organizationId ||
    order.organizationId !== organizationId ||
    order.businessType !== businessType ||
    values.version !== previousVersion ||
    !/^\d+$/.test(previousVersion) ||
    !saved.orderVersion ||
    !/^\d+$/.test(saved.orderVersion) ||
    BigInt(saved.orderVersion) !== BigInt(previousVersion) + 1n ||
    order.version !== saved.orderVersion
  )
    return false;
  if (businessType === OrderBusinessType.BUSINESS_TYPE_AI) {
    const before = values.airExport;
    const after = order.airExport;
    const unchangedVersion = (left?: string, right?: string) =>
      Boolean(
        left && /^\d+$/.test(left) && BigInt(left) > 0n && left === right,
      );
    if (
      !before?.master?.id ||
      before.master.id !== after?.master?.id ||
      before.master.organizationId !== organizationId ||
      after.master?.organizationId !== organizationId ||
      before.master.businessType !== 'AI' ||
      after.master?.businessType !== 'AI' ||
      !unchangedVersion(before.master.version, after.master?.version) ||
      !before.linkId ||
      before.linkId !== after.linkId ||
      !unchangedVersion(before.linkVersion, after.linkVersion) ||
      before.documentMode !== after.documentMode ||
      (before.active ?? false) !== (after.active ?? false)
    )
      return false;
    if (before.documentMode === 'DIRECT') return !before.house && !after.house;
    return (
      before.documentMode === 'HOUSE' &&
      Boolean(
        before.house?.id &&
          before.house.id === after.house?.id &&
          unchangedVersion(before.house.version, after.house?.version),
      )
    );
  }
  if (
    businessType !== OrderBusinessType.BUSINESS_TYPE_SI ||
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
