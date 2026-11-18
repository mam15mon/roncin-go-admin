import dayjs, { type Dayjs } from 'dayjs';

export type BillDueMode = 'terms' | 'date';

export type BillDueSettingsValue = {
  dueMode?: BillDueMode;
  paymentTermsDays?: number | null;
  dueDate?: Dayjs | null;
};

export function initialBillDueSettings(source: {
  paymentTermsDays?: number | null;
  dueDate?: string;
}): BillDueSettingsValue {
  return {
    dueMode:
      source.paymentTermsDays != null
        ? 'terms'
        : source.dueDate
          ? 'date'
          : 'terms',
    paymentTermsDays: source.paymentTermsDays ?? undefined,
    dueDate: source.dueDate ? dayjs(source.dueDate) : undefined,
  };
}

/** 纯日期按账单日推导，展示与散客提醒消费同一个最终到期结果。 */
export function billDueResult(
  billDate: string | undefined,
  value: BillDueSettingsValue,
) {
  if (!billDate) return { text: '未约定到期日', effectiveDueDate: undefined };
  if (value.dueMode === 'date') {
    const effectiveDueDate = value.dueDate?.isValid()
      ? value.dueDate.format('YYYY-MM-DD')
      : undefined;
    return {
      text: effectiveDueDate
        ? `${effectiveDueDate} 到期 · 指定日期`
        : '未约定到期日',
      effectiveDueDate,
    };
  }
  if (value.paymentTermsDays == null) {
    return { text: '未约定到期日', effectiveDueDate: undefined };
  }
  const effectiveDueDate = dayjs(billDate)
    .add(value.paymentTermsDays, 'day')
    .format('YYYY-MM-DD');
  return {
    text:
      value.paymentTermsDays === 0
        ? '当天到期 · 现结'
        : `${effectiveDueDate} 到期 · 账期 ${value.paymentTermsDays} 天`,
    effectiveDueDate,
  };
}

/** 两种录入方式只提交其中一种，保留 optional 账期的显式零。 */
export function billDuePayload(value: BillDueSettingsValue): {
  paymentTermsDays?: number;
  dueDate?: string;
} {
  if (value.dueMode === 'date') {
    if (!value.dueDate?.isValid()) throw new Error('请选择指定到期日');
    return { dueDate: value.dueDate.format('YYYY-MM-DD') };
  }
  return value.paymentTermsDays == null
    ? {}
    : { paymentTermsDays: value.paymentTermsDays };
}
