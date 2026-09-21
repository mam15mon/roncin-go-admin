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

export const SEA_HOUSE_RELEASE_TYPE_OPTIONS = [
  { label: '电放 (TELEX RELEASE)', value: 'TELEX_RELEASE' },
  { label: '正本提单 (ORIGINAL)', value: 'ORIGINAL' },
  { label: '海运单 (SEA WAYBILL)', value: 'SEA_WAYBILL' },
];

const seaHouseReleaseTypeLabelMap = Object.fromEntries(
  SEA_HOUSE_RELEASE_TYPE_OPTIONS.map((option) => [option.value, option.label]),
);

export function formatHouseReleaseType(value?: string) {
  return value ? seaHouseReleaseTypeLabelMap[value] || value : '-';
}
