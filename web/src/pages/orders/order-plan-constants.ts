export type SelectOption = { label: string; value: string | number };

export interface HouseDocItem {
  key: string;
  id?: string;
  houseNo: string;
  releaseType?: string;
  note?: string;
  status?: number;
  omitWhenEmpty?: boolean;
}

export interface MasterDocGroup {
  key: string;
  masterNo: string;
  masterDocumentType?: string;
  masterReleaseMethod?: string;
  houses: HouseDocItem[];
}

export const SEA_MASTER_DOCUMENT_TYPE_OPTIONS = [
  { label: '正本提单 (ORIGINAL B/L)', value: 'ORIGINAL_BL' },
  { label: '海运单 (SEA WAYBILL)', value: 'SEA_WAYBILL' },
];

export const SEA_MASTER_RELEASE_METHOD_OPTIONS = [
  { label: '凭正本放货 (ORIGINAL)', value: 'ORIGINAL' },
  { label: '电放 (TELEX RELEASE)', value: 'TELEX_RELEASE' },
  { label: '快速放货 (EXPRESS RELEASE)', value: 'EXPRESS_RELEASE' },
];

export const SEA_HOUSE_RELEASE_TYPE_OPTIONS = [
  { label: '电放 (TELEX RELEASE)', value: 'TELEX_RELEASE' },
  { label: '正本提单 (ORIGINAL)', value: 'ORIGINAL' },
  { label: '海运单 (SEA WAYBILL)', value: 'SEA_WAYBILL' },
];

const seaHouseReleaseTypeLabelMap = Object.fromEntries(
  SEA_HOUSE_RELEASE_TYPE_OPTIONS.map((option) => [option.value, option.label]),
);

const seaMasterDocumentTypeLabelMap = Object.fromEntries(
  SEA_MASTER_DOCUMENT_TYPE_OPTIONS.map((option) => [
    option.value,
    option.label,
  ]),
);

const seaMasterReleaseMethodLabelMap = Object.fromEntries(
  SEA_MASTER_RELEASE_METHOD_OPTIONS.map((option) => [
    option.value,
    option.label,
  ]),
);

export function formatMasterDocumentType(value?: string) {
  return value ? seaMasterDocumentTypeLabelMap[value] || value : '-';
}

export function formatMasterReleaseMethod(value?: string) {
  return value ? seaMasterReleaseMethodLabelMap[value] || value : '-';
}

export function formatHouseReleaseType(value?: string) {
  return value ? seaHouseReleaseTypeLabelMap[value] || value : '-';
}
