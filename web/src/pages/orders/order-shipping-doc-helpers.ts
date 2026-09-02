import type { HouseDocItem } from './order-plan-constants';

export type ShippingDocumentFormValue = API.OrderShippingDocumentInput & {
  status?: number;
};

let docKeySeq = 0;
export const nextDocKey = (prefix: string) =>
  `${prefix}_${Date.now()}_${++docKeySeq}`;

export function rawDocsToHouses(
  rawDocs?: ShippingDocumentFormValue[],
): HouseDocItem[] {
  if (!rawDocs || rawDocs.length === 0) {
    return [
      {
        key: nextDocKey('h'),
        houseNo: '',
        releaseType: undefined,
        note: '',
        omitWhenEmpty: true,
      },
    ];
  }

  return rawDocs.map((doc) => ({
    key: nextDocKey('h'),
    id: doc.id,
    houseNo: doc.houseNo || '',
    releaseType: doc.releaseType,
    note: doc.note,
    status: doc.status,
    omitWhenEmpty: false,
  }));
}

export function housesToRawDocs(
  houses: HouseDocItem[],
): API.OrderShippingDocumentInput[] {
  const result: API.OrderShippingDocumentInput[] = [];
  for (const h of houses) {
    const trimmedHouseNo = h.houseNo.trim();
    const isEmptyPlaceholder =
      !h.id &&
      !trimmedHouseNo &&
      !h.releaseType?.trim() &&
      !h.note?.trim();
    if (isEmptyPlaceholder) {
      continue;
    }
    result.push({
      id: h.id,
      houseNo: trimmedHouseNo,
      releaseType: h.releaseType?.trim() || undefined,
      note: h.note?.trim() || undefined,
    });
  }
  return result;
}

export function getShippingDocumentsValidationMessage(
  docs?: ShippingDocumentFormValue[],
): string | undefined {
  const houseNos = new Set<string>();

  for (const doc of docs || []) {
    const houseNo = doc.houseNo?.trim();
    if (!houseNo) {
      if (doc.releaseType?.trim() || doc.note?.trim()) {
        return '请填写分单号 (HBL)';
      }
      continue;
    }

    const normalizedHouseNo = houseNo.toLowerCase();
    if (houseNos.has(normalizedHouseNo)) {
      return `分单号 ${houseNo} 重复`;
    }
    houseNos.add(normalizedHouseNo);
  }

  return undefined;
}
