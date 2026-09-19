import Decimal from 'decimal.js';

/** 从未知错误对象中安全取出可展示文案。 */
export function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error) return error.message;
  if (typeof error === 'object' && error !== null && 'message' in error) {
    const messageValue = (error as { message?: unknown }).message;
    if (typeof messageValue === 'string' && messageValue) return messageValue;
  }
  return fallback;
}

export interface ResultConfig {
  key: string;
  role: 'ORIGINAL' | 'CREATED';
  title: string;
  targetType: 'CURRENT' | 'NEW' | 'CANDIDATE';
  candidateId?: string;
  candidateVersion?: string;
  candidateTeId?: string;
  candidateTeVersion?: string;
  masterNo?: string;
  shippingLineId?: string;
  vesselName?: string;
  voyageNo?: string;
  originLocationId?: string;
  dischargeLocationId?: string;
  transitLocationId?: string;
  etd?: string;
  eta?: string;
  internalReferenceNo?: string;
  bookingNotes?: string;
  allocationNotes?: string;
  operationNotes?: string;
  houseNo?: string;
  issuerSource?: string;
  issuerPartnerId?: string;
  houseBillNote?: string;
}

export interface FeeCurrencySummary {
  key: string;
  direction: string;
  currency: string;
  baseline: Decimal;
  assignedByResult: Record<string, Decimal>;
  remaining: Decimal;
}

export function buildSeaOrderSplitTargets(
  results: ResultConfig[],
): API.SeaOrderSplitTargetInput[] {
  return results.map((result) => {
    const usesCurrentMasterBill = result.targetType === 'CURRENT';
    return {
      clientTargetKey: result.key,
      targetType: result.targetType,
      candidateId:
        result.targetType === 'CANDIDATE' ? result.candidateId : undefined,
      candidateVersion:
        result.targetType === 'CANDIDATE' ? result.candidateVersion : undefined,
      candidateTeId:
        result.targetType === 'CANDIDATE' ? result.candidateTeId : undefined,
      candidateTeVersion:
        result.targetType === 'CANDIDATE'
          ? result.candidateTeVersion
          : undefined,
      masterNo: usesCurrentMasterBill ? undefined : result.masterNo,
      shippingLineId: usesCurrentMasterBill ? undefined : result.shippingLineId,
      vesselName: usesCurrentMasterBill ? undefined : result.vesselName,
      voyageNo: usesCurrentMasterBill ? undefined : result.voyageNo,
      originLocationId: usesCurrentMasterBill
        ? undefined
        : result.originLocationId,
      dischargeLocationId: usesCurrentMasterBill
        ? undefined
        : result.dischargeLocationId,
      transitLocationId: usesCurrentMasterBill
        ? undefined
        : result.transitLocationId,
      etd: usesCurrentMasterBill ? undefined : result.etd,
      eta: usesCurrentMasterBill ? undefined : result.eta,
    };
  });
}

export function calculateFeeCurrencySummaries(
  fees: API.SeaOrderSplitDraftFeeItem[],
  assignments: Record<string, string>,
  resultKeys: string[],
): FeeCurrencySummary[] {
  const summaries = new Map<string, FeeCurrencySummary>();

  for (const fee of fees) {
    if (!fee.id || !fee.direction || !fee.currency || !fee.totalAmount) {
      throw new Error('草稿费用缺少 ID、方向、币种或金额，无法计算费用守恒');
    }
    const key = `${fee.direction}:${fee.currency}`;
    let summary = summaries.get(key);
    if (!summary) {
      summary = {
        key,
        direction: fee.direction,
        currency: fee.currency,
        baseline: new Decimal(0),
        assignedByResult: Object.fromEntries(
          resultKeys.map((resultKey) => [resultKey, new Decimal(0)]),
        ),
        remaining: new Decimal(0),
      };
      summaries.set(key, summary);
    }

    const amount = new Decimal(fee.totalAmount);
    summary.baseline = summary.baseline.add(amount);
    const resultKey = assignments[fee.id];
    if (resultKey && summary.assignedByResult[resultKey]) {
      summary.assignedByResult[resultKey] =
        summary.assignedByResult[resultKey].add(amount);
    }
  }

  return Array.from(summaries.values())
    .map((summary) => ({
      ...summary,
      remaining: summary.baseline.sub(
        Object.values(summary.assignedByResult).reduce(
          (total, amount) => total.add(amount),
          new Decimal(0),
        ),
      ),
    }))
    .sort((left, right) => left.key.localeCompare(right.key));
}
