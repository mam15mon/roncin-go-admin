import dayjs from 'dayjs';
import { formatDate } from '@/utils/format';
import type { CreateOrderFormValues } from '../../../order-kinds/form-values';
import type { SelectOption } from '../../types';

export type SeaVoyageValues = Pick<
  CreateOrderFormValues,
  | 'originLocationId'
  | 'dischargeLocationId'
  | 'transitLocationId'
  | 'vesselName'
  | 'voyageNo'
  | 'etd'
  | 'eta'
>;

export type SeaSharedVoyageSource = {
  identity: string;
  voyage: API.SeaTransportExecution;
};

export function seaVoyageMissingFields(
  voyage: Pick<
    API.SeaTransportExecution,
    | 'vesselName'
    | 'voyageNo'
    | 'originLocationId'
    | 'dischargeLocationId'
    | 'etd'
    | 'eta'
  >,
) {
  return [
    !voyage.vesselName && '船名',
    !voyage.voyageNo && '航次',
    !voyage.originLocationId && '起运港',
    !voyage.dischargeLocationId && '卸货港',
    !voyage.etd && 'ETD',
    !voyage.eta && 'ETA',
  ].filter((item): item is string => Boolean(item));
}

export function seaMasterBillIdentity(
  masterNo?: string,
  shippingLineId?: string,
) {
  return masterNo && shippingLineId && /^[A-Za-z0-9]+$/.test(masterNo)
    ? `${shippingLineId}:${masterNo}`
    : '';
}

export function seaVoyageFormValues(
  voyage: API.SeaTransportExecution,
): SeaVoyageValues {
  return {
    originLocationId: voyage.originLocationId,
    dischargeLocationId: voyage.dischargeLocationId,
    transitLocationId: voyage.transitLocationId,
    vesselName: voyage.vesselName || undefined,
    voyageNo: voyage.voyageNo || undefined,
    etd: voyage.etd ? dayjs(voyage.etd) : undefined,
    eta: voyage.eta ? dayjs(voyage.eta) : undefined,
  };
}

export const seaVoyageFields = [
  'originLocationId',
  'dischargeLocationId',
  'transitLocationId',
  'vesselName',
  'voyageNo',
  'etd',
  'eta',
] as const;

export function isEmptySeaVoyageValue(value: unknown) {
  return value === undefined || value === null || value === '';
}

function comparableVoyageValue(
  field: keyof SeaVoyageValues,
  value: SeaVoyageValues[keyof SeaVoyageValues],
) {
  if (isEmptySeaVoyageValue(value)) return '';
  if (field === 'etd' || field === 'eta') {
    return dayjs(value).isValid()
      ? dayjs(value).startOf('day').toISOString()
      : 'invalid';
  }
  if (field === 'vesselName' || field === 'voyageNo')
    return String(value).trim().toUpperCase();
  return String(value);
}

export function seaVoyageDifferences(
  values: SeaVoyageValues,
  voyage: API.SeaTransportExecution,
  locationOptions: SelectOption[] = [],
) {
  const target = seaVoyageFormValues(voyage);
  const labels: Record<keyof SeaVoyageValues, string> = {
    originLocationId: '起运港',
    dischargeLocationId: '卸货港',
    transitLocationId: '中转港',
    vesselName: '船名',
    voyageNo: '航次',
    etd: 'ETD',
    eta: 'ETA',
  };
  const locationNames = {
    originLocationId: voyage.originLocationName,
    dischargeLocationId: voyage.dischargeLocationName,
    transitLocationId: voyage.transitLocationName,
  };
  function display(
    field: keyof SeaVoyageValues,
    value: SeaVoyageValues[keyof SeaVoyageValues],
    shared: boolean,
  ) {
    if (isEmptySeaVoyageValue(value)) return '未填写';
    if (field === 'etd' || field === 'eta')
      return formatDate(dayjs(value).toDate(), 'date');
    if (field === 'vesselName' || field === 'voyageNo') return String(value);
    return (
      (shared ? locationNames[field] : undefined) ||
      locationOptions.find((option) => option.value === value)?.label ||
      (shared ? '主单已选地点' : '当前已选地点')
    );
  }
  return seaVoyageFields.flatMap((field) => {
    if (
      isEmptySeaVoyageValue(values[field]) ||
      comparableVoyageValue(field, values[field]) ===
        comparableVoyageValue(field, target[field])
    )
      return [];
    return [
      {
        field,
        label: labels[field],
        currentValue: display(field, values[field], false),
        sharedValue: display(field, target[field], true),
      },
    ];
  });
}
