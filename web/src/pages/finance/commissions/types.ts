import type { Dayjs } from 'dayjs';
import { FinanceCommissionStatus } from '@/enums.generated';

export type CreateValues = {
  // 来源二选一：核销与对冲恰好提供一个，与后端契约一致。
  verificationId?: string;
  nettingId?: string;
  // 候选选中键：`${employeeId}|${personnelRole}`，规则由服务端按来源日期解析。
  candidateKey?: string;
  note?: string;
};

export type RuleValues = {
  organizationId?: string;
  name: string;
  personnelRole: 'SALES' | 'OPERATOR' | 'CUSTOMER_SERVICE';
  calculationBasis: 'REALIZED_PROFIT' | 'REALIZED_REVENUE';
  ratePercent: number;
  effectiveRange?: [Dayjs, Dayjs];
  enabled: boolean;
  note?: string;
  employeeIds?: string[];
};

/** 解析候选选中键为员工与人员身份；非法键返回空。 */
export const parseCandidateKey = (
  key?: string,
): { employeeId?: string; personnelRole?: string } => {
  if (!key) return {};
  const separator = key.indexOf('|');
  if (separator <= 0) return {};
  const employeeId = key.slice(0, separator);
  const personnelRole = key.slice(separator + 1);
  return employeeId && personnelRole ? { employeeId, personnelRole } : {};
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
  [FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT]: {
    text: '草稿',
    color: 'processing',
  },
  [FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED]: {
    text: '已确认',
    color: 'success',
  },
  [FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_PAID]: {
    text: '已发放',
    color: 'blue',
  },
  [FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CANCELLED]: {
    text: '已取消',
    color: 'default',
  },
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
  [values.verificationId, values.nettingId, values.candidateKey].join('|');

/** 来源单号二选一展示：核销单号或对冲单号，两者都空显示占位符。 */
export const commissionSourceNo = (
  record: { verificationNo?: string; nettingNo?: string },
  placeholder = '-',
) => record.verificationNo || record.nettingNo || placeholder;

/**
 * 锁后费用补录冲减建议的页面文案：严格区分「待处理 / 已确认 / 已扣回 / 已取消」。
 * 待处理不等于已确认，已确认不等于已扣回。
 */
export const commissionDecreaseStatusMeta: Record<
  number,
  { text: string; color: string }
> = {
  [FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT]: {
    text: '待处理',
    color: 'processing',
  },
  [FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED]: {
    text: '已确认',
    color: 'warning',
  },
  [FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_PAID]: {
    text: '已扣回',
    color: 'purple',
  },
  [FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CANCELLED]: {
    text: '已取消',
    color: 'default',
  },
};

/** 锁后费用补录来源（冲减建议）的系统来源类型标识。 */
export const LOCKED_FEE_SUPPLEMENT_SOURCE = 'LOCKED_FEE_SUPPLEMENT';

export function getBusinessReason(error: any): string {
  return (
    error?.data?.reason ?? error?.response?.data?.reason ?? error?.reason ?? ''
  );
}

// isReversalAdjustment 判断调整单是否为系统生成的来源反转冲减
// （核销撤销 NETTING 前的反核销 / 对冲撤销），此类调整不可手工取消或确认。
export function isReversalAdjustment(sourceType?: string): boolean {
  return (
    sourceType === 'VERIFICATION_REVERSAL' || sourceType === 'NETTING_REVERSAL'
  );
}

export function getAdjustmentStatusInfo(
  adjustment: API.FinanceCommissionAdjustment,
) {
  const isReversal = isReversalAdjustment(adjustment.sourceType);
  const isDecrease = adjustment.direction === 'DECREASE';

  if (isReversal) {
    if (
      adjustment.status ===
      FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED
    )
      return { text: '待追回', color: 'warning' };
    if (
      adjustment.status ===
      FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_PAID
    )
      return { text: '已追回', color: 'purple' };
    if (
      adjustment.status ===
      FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CANCELLED
    )
      return { text: '已取消', color: 'default' };
    return { text: '反核销草稿', color: 'processing' };
  }

  if (isDecrease) {
    if (
      adjustment.status ===
      FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT
    )
      return { text: '冲减草稿', color: 'processing' };
    if (
      adjustment.status ===
      FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED
    )
      return { text: '待扣回', color: 'warning' };
    if (
      adjustment.status ===
      FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_PAID
    )
      return { text: '已扣回', color: 'purple' };
    return { text: '已取消', color: 'default' };
  }

  if (
    adjustment.status ===
    FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT
  )
    return { text: '增提草稿', color: 'processing' };
  if (
    adjustment.status ===
    FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED
  )
    return { text: '待发放', color: 'success' };
  if (
    adjustment.status === FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_PAID
  )
    return { text: '已发放', color: 'blue' };
  return { text: '已取消', color: 'default' };
}
