import dayjs from 'dayjs';
import { formatDate } from '@/utils/format';
import type { CreateOrderFormValues } from '../../../order-kinds/form-values';
import type { SelectOption } from '../../types';

export type SeaVoyageValues = Pick<
  CreateOrderFormValues,
  | 'originLocationId'
  | 'dischargeLocationId'
  | 'transitLocationId'
  | 'vesselVoyage'
  | 'etd'
  | 'eta'
>;

export type SeaSharedVoyageSource = {
  identity: string;
  voyage: API.SeaTransportExecution;
};

export function seaMasterBillIdentity(
  masterNo?: string,
  shippingLineId?: string,
) {
  return masterNo && shippingLineId && /^[A-Za-z0-9]+$/.test(masterNo)
    ? `${shippingLineId}:${masterNo}`
    : '';
}

export function splitSeaVesselVoyage(value?: string) {
  const normalized = value?.trim() || '';
  if (!normalized) return { vesselName: undefined, voyageNo: undefined };
  const slashIndex = normalized.indexOf('/');
  if (slashIndex >= 0) {
    return {
      vesselName: normalized.slice(0, slashIndex).trim() || undefined,
      voyageNo: normalized.slice(slashIndex + 1).trim() || undefined,
    };
  }
  const lastSpaceIndex = normalized.lastIndexOf(' ');
  if (lastSpaceIndex >= 0) {
    return {
      vesselName: normalized.slice(0, lastSpaceIndex).trim() || undefined,
      voyageNo: normalized.slice(lastSpaceIndex + 1).trim() || undefined,
    };
  }
  return { vesselName: normalized, voyageNo: undefined };
}

export function seaVoyageFormValues(
  voyage: API.SeaTransportExecution,
): SeaVoyageValues {
  return {
    originLocationId: voyage.originLocationId,
    dischargeLocationId: voyage.dischargeLocationId,
    transitLocationId: voyage.transitLocationId,
    vesselVoyage:
      voyage.vesselName || voyage.voyageNo
        ? `${voyage.vesselName || ''} / ${voyage.voyageNo || ''}`
        : undefined,
    etd: voyage.etd ? dayjs(voyage.etd) : undefined,
    eta: voyage.eta ? dayjs(voyage.eta) : undefined,
  };
}

export const seaVoyageFields = [
  'originLocationId',
  'dischargeLocationId',
  'transitLocationId',
  'vesselVoyage',
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
  if (field === 'vesselVoyage') {
    const split = splitSeaVesselVoyage(String(value));
    return `${split.vesselName?.toUpperCase() || ''}/${split.voyageNo?.toUpperCase() || ''}`;
  }
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
    vesselVoyage: '船名航次',
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
    if (field === 'vesselVoyage') return String(value);
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
