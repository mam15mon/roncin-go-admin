import type { MasterDocGroup } from './order-plan-constants';

export type ShippingDocumentFormValue = API.OrderShippingDocumentInput & {
  status?: number;
};

let docKeySeq = 0;
export const nextDocKey = (prefix: string) => `${prefix}_${Date.now()}_${++docKeySeq}`;

export function rawDocsToGroups(
  rawDocs?: ShippingDocumentFormValue[],
): MasterDocGroup[] {
  if (!rawDocs || rawDocs.length === 0) {
    return [
      {
        key: nextDocKey('mg'),
        masterNo: '',
        houses: [
          { key: nextDocKey('h'), houseNo: '', releaseType: undefined, note: '' },
        ],
      },
    ];
  }

  const groupMap = new Map<string, MasterDocGroup>();
  const groups: MasterDocGroup[] = [];

  for (const doc of rawDocs) {
    const rawMaster = doc.masterNo || '';
    const masterKey = rawMaster.trim().toLowerCase();

    let group = groupMap.get(masterKey);
    if (!group) {
      group = {
        key: nextDocKey('mg'),
        masterNo: rawMaster,
        masterDocumentType: doc.masterDocumentType,
        masterReleaseMethod: doc.masterReleaseMethod,
        houses: [],
      };
      groupMap.set(masterKey, group);
      groups.push(group);
    }

    group.houses.push({
      key: nextDocKey('h'),
      id: doc.id,
      houseNo: doc.houseNo || '',
      releaseType: doc.releaseType,
      note: doc.note,
      status: doc.status,
      omitWhenEmpty: false,
    });
  }

  for (const g of groups) {
    if (g.houses.length === 0) {
      g.houses.push({
        key: nextDocKey('h'),
        houseNo: '',
        releaseType: undefined,
        note: '',
      });
    }
  }

  return groups.length > 0
    ? groups
    : [
        {
          key: nextDocKey('mg'),
          masterNo: '',
          houses: [
            {
              key: nextDocKey('h'),
              houseNo: '',
              releaseType: undefined,
              note: '',
            },
          ],
        },
      ];
}

export function groupsToRawDocs(
  groups: MasterDocGroup[],
): API.OrderShippingDocumentInput[] {
  const result: API.OrderShippingDocumentInput[] = [];
  for (const g of groups) {
    const masterNo = g.masterNo;
    for (const h of g.houses) {
      const isEmptyPlaceholder =
        !h.id &&
        !h.houseNo.trim() &&
        !h.releaseType?.trim() &&
        !h.note?.trim() &&
        (h.omitWhenEmpty ||
          (!masterNo.trim() &&
            !g.masterDocumentType?.trim() &&
            !g.masterReleaseMethod?.trim()));
      if (isEmptyPlaceholder) {
        continue;
      }
      result.push({
        id: h.id,
        masterNo,
        masterDocumentType: g.masterDocumentType,
        masterReleaseMethod: g.masterReleaseMethod,
        houseNo: h.houseNo,
        releaseType: h.releaseType,
        note: h.note,
      });
    }
  }
  return result;
}

export function getShippingDocumentsValidationMessage(
  docs?: ShippingDocumentFormValue[],
): string | undefined {
  const houseNos = new Set<string>();

  for (const doc of docs || []) {
    if (!doc.masterNo?.trim()) {
      return '请填写主单号';
    }
    const houseNo = doc.houseNo?.trim();
    if (!houseNo) {
      return '请填写分单号';
    }

    const normalizedHouseNo = houseNo.toLowerCase();
    if (houseNos.has(normalizedHouseNo)) {
      return `分单号 ${houseNo} 重复`;
    }
    houseNos.add(normalizedHouseNo);
  }

  return undefined;
}

export function getDuplicateMasterNo(groups: MasterDocGroup[]): string | undefined {
  const masterNos = new Set<string>();

  for (const group of groups) {
    const masterNo = group.masterNo.trim();
    if (!masterNo) {
      continue;
    }
    const normalizedMasterNo = masterNo.toLowerCase();
    if (masterNos.has(normalizedMasterNo)) {
      return masterNo;
    }
    masterNos.add(normalizedMasterNo);
  }

  return undefined;
}
