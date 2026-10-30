import { SeaHouseBillIssuerSource } from '@/enums.generated';
import type { OrderDetailFormValues } from './order-kinds/form-values';
import {
  DEFAULT_BILL_FORM,
  DEFAULT_FREIGHT_TERMS,
  DEFAULT_RELEASE_TYPE,
  DEFAULT_TRANSPORT_TERMS,
} from './templates/components/sea/seaDocumentSectionConstants';

export function seaBillContentFormValues(
  content?: API.SeaBillContent,
): API.SeaBillContent {
  return {
    ...content,
    transportTerms: content?.transportTerms?.trim() || DEFAULT_TRANSPORT_TERMS,
    freightTerms: content?.freightTerms?.trim() || DEFAULT_FREIGHT_TERMS,
    billForm: content?.billForm ?? DEFAULT_BILL_FORM,
    releaseType: content?.releaseType ?? DEFAULT_RELEASE_TYPE,
  };
}

export function seaDocumentFormValues(
  documents?: API.SeaOrderDocuments,
): Partial<OrderDetailFormValues> {
  if (!documents) return {};
  const house = documents.houseBill;
  return {
    seaDocumentStructure: documents.documentStructure,
    seaDocumentLinkVersion: documents.linkVersion,
    seaMasterBillContentVersion: documents.masterBill?.version,
    seaMasterBillContent: seaBillContentFormValues(
      documents.masterBill?.content,
    ),
    seaHouseBill: house
      ? {
          id: house.id,
          houseNo: house.houseNo ?? '',
          issuerSource: house.issuerSource ?? 0,
          issuerPartnerId:
            house.issuerSource ===
            SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_OTHER_PARTNER
              ? house.issuerPartnerId
              : undefined,
          note: house.note,
          content: seaBillContentFormValues(house.content),
          expectedVersion: house.version,
          status: house.status,
        }
      : undefined,
  };
}

// Proto 省略字段、空输入与对象键顺序不代表业务修改，比较前统一成稳定结构。
function canonical(value: unknown): string {
  const normalize = (item: unknown): unknown => {
    if (item === null || item === undefined || item === '') return undefined;
    if (typeof item === 'string') return item;
    if (Array.isArray(item)) return item.map(normalize);
    if (typeof item === 'object')
      return Object.fromEntries(
        Object.entries(item)
          .sort(([a], [b]) => a.localeCompare(b))
          .map(([key, entry]) => [key, normalize(entry)])
          .filter(([, entry]) => entry !== undefined),
      );
    return item;
  };
  return JSON.stringify(normalize(value));
}

export function buildSeaDocumentUpdate(
  values: Pick<
    OrderDetailFormValues,
    | 'seaMasterBillContent'
    | 'seaHouseBill'
    | 'seaMasterBillContentVersion'
    | 'seaDocumentLinkVersion'
  >,
  documents: API.SeaOrderDocuments,
): API.SeaOrderDocumentInput | undefined {
  const baseline = seaDocumentFormValues(documents);
  const masterChanged =
    canonical(values.seaMasterBillContent) !==
    canonical(baseline.seaMasterBillContent);
  const editableHouse = (house?: API.SeaHouseBillInput) =>
    house
      ? {
          houseNo: house.houseNo,
          issuerSource: house.issuerSource,
          issuerPartnerId: house.issuerPartnerId,
          note: house.note,
          content: house.content,
        }
      : undefined;
  const houseChanged =
    canonical(editableHouse(values.seaHouseBill)) !==
    canonical(editableHouse(baseline.seaHouseBill));
  if (!masterChanged && !houseChanged) return undefined;
  if (masterChanged && !values.seaMasterBillContentVersion)
    throw new Error('主单草稿缺少原版本，请保留输入并刷新核对后重试');
  if (
    houseChanged &&
    (!values.seaDocumentLinkVersion ||
      !values.seaHouseBill?.expectedVersion ||
      values.seaHouseBill.id !== documents.houseBill?.id)
  )
    throw new Error('分单草稿的身份或原版本不完整，请保留输入并刷新核对后重试');
  return {
    expectedMblVersion: masterChanged
      ? values.seaMasterBillContentVersion
      : undefined,
    masterBillContent: masterChanged ? values.seaMasterBillContent : undefined,
    expectedLinkVersion: houseChanged
      ? values.seaDocumentLinkVersion
      : undefined,
    houseBill: houseChanged
      ? {
          ...editableHouse(values.seaHouseBill),
          issuerSource: values.seaHouseBill?.issuerSource ?? 0,
          id: values.seaHouseBill?.id,
          houseNo: values.seaHouseBill?.houseNo ?? '',
          expectedVersion: values.seaHouseBill?.expectedVersion,
        }
      : undefined,
  };
}

/** 专用命令仅推进它负责的文档快照，其余草稿继续对应原版本。 */
export function mergeSeaDocumentBaseline(
  current: API.SeaOrderDocuments | undefined,
  latest: API.SeaOrderDocuments,
  document?: 'master' | 'house' | 'mode',
): API.SeaOrderDocuments {
  return document === 'master'
    ? { ...current, masterBill: latest.masterBill }
    : document === 'house'
      ? {
          ...current,
          houseBill: latest.houseBill,
          linkVersion: latest.linkVersion,
        }
      : latest;
}
