import dayjs from 'dayjs';
import type { SeaExternalConfirmationFormValues } from '../../templates/components/sea/SeaExternalConfirmationFields';

export type ReassignmentFormValues = SeaExternalConfirmationFormValues & {
  masterNo?: string;
  shippingLineId?: string;
  vesselName?: string;
  voyageNo?: string;
  originLocationId?: string;
  dischargeLocationId?: string;
  transitLocationId?: string;
  etd?: dayjs.Dayjs;
  eta?: dayjs.Dayjs;
  responsibilityType?: string;
  responsiblePartnerId?: string;
  reason?: string;
};

/** 预览和执行复用同一目标；候选航程只取用户明确选中的权威数据。 */
export function buildReassignmentTarget(
  values: ReassignmentFormValues,
  targetType: 'candidate' | 'new',
  candidate: API.SeaMasterBillCandidate | null,
  transportExecution: API.SeaTransportExecution | null,
): API.SeaOrderReassignmentTargetInput | undefined {
  if (targetType === 'candidate') {
    if (
      !candidate?.id ||
      !candidate.version ||
      !transportExecution?.id ||
      !transportExecution.version
    ) {
      return undefined;
    }
    return {
      targetType: 'CANDIDATE',
      candidateId: candidate.id,
      candidateVersion: String(candidate.version),
      candidateTeId: transportExecution.id,
      candidateTeVersion: String(transportExecution.version),
      masterNo: candidate.masterNo,
      shippingLineId: transportExecution.shippingLineId,
      vesselName: transportExecution.vesselName,
      voyageNo: transportExecution.voyageNo,
      originLocationId: transportExecution.originLocationId,
      dischargeLocationId: transportExecution.dischargeLocationId,
      transitLocationId: transportExecution.transitLocationId,
      etd: transportExecution.etd,
      eta: transportExecution.eta,
    };
  }
  if (
    !values.masterNo ||
    !/^[A-Za-z0-9]+$/.test(values.masterNo) ||
    !values.shippingLineId
  ) {
    return undefined;
  }
  return {
    targetType: 'NEW',
    masterNo: values.masterNo,
    shippingLineId: values.shippingLineId,
    vesselName: values.vesselName?.trim(),
    voyageNo: values.voyageNo?.trim(),
    originLocationId: values.originLocationId,
    dischargeLocationId: values.dischargeLocationId,
    transitLocationId: values.transitLocationId,
    etd: values.etd ? dayjs(values.etd).toISOString() : undefined,
    eta: values.eta ? dayjs(values.eta).toISOString() : undefined,
  };
}
