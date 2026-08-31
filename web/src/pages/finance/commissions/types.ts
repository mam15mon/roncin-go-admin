import type { Dayjs } from 'dayjs';
import { FinanceCommissionStatus } from '@/enums.generated';

export type CreateValues = {
  verificationId: string;
  employeeId: string;
  ruleId: string;
  note?: string;
};

export type RuleValues = {
  name: string;
  personnelRole: 'SALES' | 'OPERATOR' | 'CUSTOMER_SERVICE';
  calculationBasis: 'REALIZED_PROFIT' | 'REALIZED_REVENUE';
  ratePercent: number;
  effectiveRange?: [Dayjs, Dayjs];
  enabled: boolean;
  note?: string;
};

export type AdjustmentValues = {
  orderId: string;
  direction: 'INCREASE' | 'DECREASE';
  amount: string;
  reason: string;
  note?: string;
};

export const commissionStatusMeta: Record<
  number,
  { text: string; color: string }
> = {
  [FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT]: { text: '草稿', color: 'processing' },
  [FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED]: { text: '已确认', color: 'success' },
  [FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_PAID]: { text: '已发放', color: 'blue' },
  [FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CANCELLED]: { text: '已取消', color: 'default' },
};

export const personnelRoleMeta: Record<string, string> = {
  SALES: '业务人员',
  OPERATOR: '操作人员',
  CUSTOMER_SERVICE: '客服人员',
};

export const personnelRoleText = (value?: string) =>
  personnelRoleMeta[value || ''] || '业务人员';

export const calculationBasisMeta: Record<string, string> = {
  REALIZED_PROFIT: '已实现毛利',
  REALIZED_REVENUE: '已实现收入',
};

export const calculationBasisText = (value?: string) =>
  calculationBasisMeta[value || ''] || '已实现毛利';

export const cnyExchangeRateSourceText = (value?: string) => {
  if (value === 'BASE_CURRENCY') return '本位币即 CNY';
  if (value === 'DERIVED') return '倒数派生';
  return value || '-';
};

export const decimalText = (value?: string) => {
  if (!value) return '0';
  return value.replace(/(\.\d*?[1-9])0+$|\.0+$/, '$1');
};

export const calculationSignature = (values: Partial<CreateValues>) =>
  [values.verificationId, values.ruleId, values.employeeId].join('|');

export function getBusinessReason(error: any): string {
  return (
    error?.data?.reason ??
    error?.response?.data?.reason ??
    error?.reason ??
    ''
  );
}

export function getAdjustmentStatusInfo(
  adjustment: API.FinanceCommissionAdjustment,
) {
  const isReversal = adjustment.sourceType === 'VERIFICATION_REVERSAL';
  const isDecrease = adjustment.direction === 'DECREASE';

  if (isReversal) {
    if (adjustment.status === FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED)
      return { text: '待追回', color: 'warning' };
    if (adjustment.status === FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_PAID)
      return { text: '已追回', color: 'purple' };
    if (adjustment.status === FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CANCELLED)
      return { text: '已取消', color: 'default' };
    return { text: '反核销草稿', color: 'processing' };
  }

  if (isDecrease) {
    if (adjustment.status === FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT)
      return { text: '冲减草稿', color: 'processing' };
    if (adjustment.status === FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED)
      return { text: '待扣回', color: 'warning' };
    if (adjustment.status === FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_PAID)
      return { text: '已扣回', color: 'purple' };
    return { text: '已取消', color: 'default' };
  }

  if (adjustment.status === FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT)
    return { text: '增提草稿', color: 'processing' };
  if (adjustment.status === FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED)
    return { text: '待发放', color: 'success' };
  if (adjustment.status === FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_PAID)
    return { text: '已发放', color: 'blue' };
  return { text: '已取消', color: 'default' };
}
