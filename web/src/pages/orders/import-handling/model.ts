import dayjs, { type Dayjs } from 'dayjs';
import { ImportHandlingStepKind } from '@/enums.generated';

export const IMPORT_SERVICES = [
  {
    kind: ImportHandlingStepKind.IMPORT_HANDLING_STEP_KIND_ARRIVAL,
    label: '到货处理',
  },
  {
    kind: ImportHandlingStepKind.IMPORT_HANDLING_STEP_KIND_EXCHANGE,
    label: '换单',
  },
  {
    kind: ImportHandlingStepKind.IMPORT_HANDLING_STEP_KIND_CUSTOMS,
    label: '清关',
  },
  {
    kind: ImportHandlingStepKind.IMPORT_HANDLING_STEP_KIND_PICKUP,
    label: '提货',
  },
  {
    kind: ImportHandlingStepKind.IMPORT_HANDLING_STEP_KIND_DELIVERY,
    label: '派送',
  },
] as const;

export type HandlingSnapshot = API.OrderImportHandling & {
  orderId: string;
  orderVersion: string;
  steps: API.ImportHandlingStep[];
};
export type HandlingFormValues = {
  exchangeAgentId?: string;
  reason?: string;
  steps: Array<
    Omit<API.ImportHandlingStep, 'occurredAt'> & { occurredAt?: string | Dayjs }
  >;
};

export function validVersion(version?: string): version is string {
  return (
    typeof version === 'string' && /^\d+$/.test(version) && BigInt(version) > 0n
  );
}

/** 五项完整快照是必填合同；省略的非optional undertaken 仅表示 false。 */
export function handlingSnapshot(
  value: API.OrderImportHandling | undefined,
  orderId: string,
): HandlingSnapshot {
  if (
    !value ||
    value.orderId !== orderId ||
    !validVersion(value.orderVersion) ||
    value.steps?.length !== IMPORT_SERVICES.length
  )
    throw new Error('进口服务资料不完整，请重新刷新');
  const byKind = new Map(value.steps.map((step) => [step.kind, step]));
  if (byKind.size !== IMPORT_SERVICES.length)
    throw new Error('进口服务环节重复，请重新刷新');
  const steps = IMPORT_SERVICES.map(({ kind }) => {
    const step = byKind.get(kind);
    if (
      !step ||
      (step.occurredAt !== undefined &&
        (!step.occurredAt ||
          !step.undertaken ||
          !dayjs(step.occurredAt).isValid()))
    )
      throw new Error('进口服务完成事实不合法，请重新刷新');
    return {
      ...step,
      undertaken: step.undertaken === true,
      referenceNo: step.referenceNo ?? '',
      notes: step.notes ?? '',
      attachmentIds: step.attachmentIds ?? [],
    };
  });
  return { ...value, orderId, orderVersion: value.orderVersion, steps };
}

export function handlingFormValues(
  snapshot: HandlingSnapshot,
): HandlingFormValues {
  return {
    exchangeAgentId: snapshot.exchangeAgentId,
    steps: snapshot.steps.map((step) => ({
      ...step,
      attachmentIds: [...(step.attachmentIds ?? [])],
      occurredAt: step.occurredAt ? dayjs(step.occurredAt) : undefined,
    })),
  };
}

export function handlingPayload(
  snapshot: HandlingSnapshot,
  values: HandlingFormValues,
): API.SaveImportHandlingRequest {
  if (
    !validVersion(snapshot.orderVersion) ||
    values.steps?.length !== IMPORT_SERVICES.length
  )
    throw new Error('进口服务提交基线缺失，请刷新后重试');
  return {
    orderId: snapshot.orderId,
    expectedVersion: snapshot.orderVersion,
    exchangeAgentId: values.exchangeAgentId || undefined,
    reason: values.reason?.trim() || undefined,
    steps: IMPORT_SERVICES.map(({ kind }, index) => {
      const value = values.steps[index];
      if (value.kind !== kind)
        throw new Error('进口服务环节顺序不合法，请刷新后重试');
      const original = snapshot.steps[index]?.occurredAt;
      const selected = value.occurredAt ? dayjs(value.occurredAt) : undefined;
      // 日期控件仅表示毫秒；时刻未改时保留服务端完整纳秒精度，避免无意纠正完成事实。
      const occurredAt = selected
        ? original && selected.valueOf() === dayjs(original).valueOf()
          ? original
          : selected.toISOString()
        : undefined;
      return {
        kind,
        undertaken: value.undertaken === true,
        occurredAt: value.undertaken ? occurredAt : undefined,
        responsibleUserId: value.responsibleUserId || undefined,
        referenceNo: value.referenceNo?.trim() || '',
        notes: value.notes?.trim() || '',
        attachmentIds: value.attachmentIds ?? [],
      };
    }),
  };
}

function timeFingerprint(value?: string): string {
  if (!value) return '';
  const fraction = /\.(\d+)(?:Z|[+-]\d{2}:\d{2})$/.exec(value)?.[1] ?? '';
  return `${dayjs(value).toISOString()}:${fraction.padEnd(9, '0').slice(3, 9)}`;
}

function stepFingerprint(step: API.ImportHandlingStep): string {
  return JSON.stringify([
    step.kind,
    step.undertaken === true,
    timeFingerprint(step.occurredAt),
    step.responsibleUserId || '',
    step.referenceNo?.trim() || '',
    step.notes?.trim() || '',
    step.attachmentIds ?? [],
  ]);
}

export function needsHandlingReason(
  before: HandlingSnapshot,
  next: API.SaveImportHandlingRequest,
): boolean {
  return (
    before.steps.some(
      (step, index) =>
        step.occurredAt &&
        stepFingerprint(step) !== stepFingerprint(next.steps?.[index] ?? {}),
    ) ||
    Boolean(
      before.steps.find(
        (step) =>
          step.kind ===
          ImportHandlingStepKind.IMPORT_HANDLING_STEP_KIND_EXCHANGE,
      )?.occurredAt &&
        (before.exchangeAgentId || '') !== (next.exchangeAgentId || ''),
    )
  );
}

export function cancelledServices(
  before: HandlingSnapshot,
  next: API.SaveImportHandlingRequest,
): string[] {
  return IMPORT_SERVICES.flatMap(({ label }, index) =>
    before.steps[index].undertaken && next.steps?.[index]?.undertaken !== true
      ? [label]
      : [],
  );
}

export function serviceStatus(step: API.ImportHandlingStep): string {
  return step.undertaken !== true
    ? '不适用'
    : step.occurredAt
      ? '已完成'
      : '未完成';
}

export function handlingChanged(
  before: HandlingSnapshot,
  next: API.OrderImportHandling,
): boolean {
  return (
    before.orderVersion !== next.orderVersion ||
    (before.exchangeAgentId || '') !== (next.exchangeAgentId || '') ||
    before.steps.some(
      (step, index) =>
        stepFingerprint(step) !== stepFingerprint(next.steps?.[index] ?? {}),
    )
  );
}
