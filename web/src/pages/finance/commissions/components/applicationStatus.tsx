import { Tag } from 'antd';
import { FinanceCommissionApplicationStatus } from '@/enums.generated';

/**
 * 财务侧月度申请状态文案：使用财务独立枚举（禁止借用工作台枚举）。
 * 待审批≠已批准，驳回后员工只能在原申请上重提。
 */
export const financeApplicationStatusMeta: Record<
  number,
  { text: string; color: string }
> = {
  [FinanceCommissionApplicationStatus.FINANCE_COMMISSION_APPLICATION_STATUS_PENDING_REVIEW]:
    { text: '待审批', color: 'processing' },
  [FinanceCommissionApplicationStatus.FINANCE_COMMISSION_APPLICATION_STATUS_REJECTED]:
    { text: '已驳回', color: 'error' },
  [FinanceCommissionApplicationStatus.FINANCE_COMMISSION_APPLICATION_STATUS_APPROVED]:
    { text: '已批准', color: 'success' },
};

export function applicationStatusTag(status?: number) {
  const meta = financeApplicationStatusMeta[status ?? 0];
  return <Tag color={meta?.color || 'default'}>{meta?.text || '-'}</Tag>;
}

export const APPLICATION_STATUS_FILTER_OPTIONS = [
  {
    label: '待审批',
    value:
      FinanceCommissionApplicationStatus.FINANCE_COMMISSION_APPLICATION_STATUS_PENDING_REVIEW,
  },
  {
    label: '已驳回',
    value:
      FinanceCommissionApplicationStatus.FINANCE_COMMISSION_APPLICATION_STATUS_REJECTED,
  },
  {
    label: '已批准',
    value:
      FinanceCommissionApplicationStatus.FINANCE_COMMISSION_APPLICATION_STATUS_APPROVED,
  },
];
