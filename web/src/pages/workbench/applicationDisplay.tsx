import { Tag } from 'antd';
import Decimal from 'decimal.js';
import { WorkbenchCommissionApplicationStatus } from '@/enums.generated';
import { formatAmount } from '@/utils/format';

/**
 * 本人月度提成申请状态文案：只使用工作台独立枚举（禁止借用财务枚举）。
 * PENDING_REVIEW=审批中、REJECTED=已驳回（原申请可重提）、APPROVED=已批准。
 */
export const workbenchApplicationStatusMeta: Record<
  number,
  { text: string; color: string }
> = {
  [WorkbenchCommissionApplicationStatus.WORKBENCH_COMMISSION_APPLICATION_STATUS_PENDING_REVIEW]:
    { text: '审批中', color: 'processing' },
  [WorkbenchCommissionApplicationStatus.WORKBENCH_COMMISSION_APPLICATION_STATUS_REJECTED]:
    { text: '已驳回', color: 'error' },
  [WorkbenchCommissionApplicationStatus.WORKBENCH_COMMISSION_APPLICATION_STATUS_APPROVED]:
    { text: '已批准', color: 'success' },
};

export function applicationStatusTag(status?: number) {
  const meta = workbenchApplicationStatusMeta[status ?? 0];
  return <Tag color={meta?.color || 'default'}>{meta?.text || '-'}</Tag>;
}

/**
 * 本位币金额合计：可申请分组金额全部为组织本位币口径（服务端契约保证），
 * 同币种才允许相加；使用 Decimal 精确累加，不裸相加浮点。
 */
export const sumBaseAmounts = (values: Array<string | undefined>): string =>
  values
    .reduce<Decimal>(
      (acc, value) => acc.plus(new Decimal(value ?? 0)),
      new Decimal(0),
    )
    .toFixed(2);

/** 归属月展示：YYYY-MM 原样展示，缺省占位。 */
export const applicationMonthText = (value?: string) => value || '-';

/** 金额展示复用工作台口径：缺省按 0 处理。 */
export const applicationAmount = (value?: string) => formatAmount(value ?? '0');
