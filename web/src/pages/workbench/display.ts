import { FinanceCommissionStatus } from '@/enums.generated';
import { formatAmount } from '@/utils/format';

/**
 * 工作台提成状态文案：与卡片三桶语义严格一致，
 * DRAFT=待财务确认、CONFIRMED=已确认待发、PAID=已发放、CANCELLED 不计入有效金额。
 */
export const workbenchCommissionStatusMeta: Record<
  number,
  { text: string; color: string }
> = {
  [FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT]: {
    text: '待财务确认',
    color: 'processing',
  },
  [FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED]: {
    text: '已确认待发',
    color: 'warning',
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

/**
 * 冲减调整三阶段文案：待处理≠已确认，已确认≠已实际扣回。
 * DRAFT=待处理尚未扣回、CONFIRMED=已确认尚未实际扣回、PAID=已扣回。
 */
export const workbenchDecreaseStatusMeta: Record<
  number,
  { text: string; color: string }
> = {
  [FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT]: {
    text: '待处理（尚未扣回）',
    color: 'processing',
  },
  [FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED]: {
    text: '已确认（尚未实际扣回）',
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

export const workbenchPersonnelRoleText = (value?: string) => {
  const meta: Record<string, string> = {
    SALES: '业务人员',
    OPERATOR: '操作人员',
    CUSTOMER_SERVICE: '客服人员',
  };
  return meta[value || ''] || '-';
};

export const workbenchCalculationBasisText = (value?: string) => {
  const meta: Record<string, string> = {
    REALIZED_PROFIT: '已实现毛利',
    REALIZED_REVENUE: '已实现收入',
  };
  return meta[value || ''] || '-';
};

const ORDER_FLOW_STATUS_TEXT: Record<string, string> = {
  DRAFT: '草稿',
  BOOKED: '已订舱',
  SPACE_ALLOCATED: '已配舱',
  TRUCKING_ARRANGED: '已安排拖车',
  DOCUMENT_CUTOFF: '已截单',
  CUSTOMS_DECLARATION_ARRANGED: '已安排报关',
  DOCUMENT_RELEASED: '已放单',
};

const ORDER_TERMINATION_STATUS_TEXT: Record<string, string> = {
  ACTIVE: '进行中',
  TERMINATING: '终止中',
  TERMINATED: '已终止',
};

export const orderFlowStatusText = (value?: string) =>
  ORDER_FLOW_STATUS_TEXT[value || ''] || value || '-';

export const orderTerminationStatusText = (value?: string) =>
  ORDER_TERMINATION_STATUS_TEXT[value || ''] || value || '-';

/**
 * 金额展示：十进制字符串按既有 formatAmount 惯例格式化；
 * 缺省按 0 处理（工作台桶金额缺省即代表无金额）。
 */
export const workbenchAmount = (value?: string) => formatAmount(value ?? '0');

/** 金额 + 币种展示；不同币种只并列展示，绝不裸相加。 */
export const amountWithCurrency = (value?: string, currency?: string) =>
  `${workbenchAmount(value)} ${currency || '-'}`;

/** 来源单号二选一：核销单号或对冲单号。 */
export const workbenchSourceNo = (record: {
  verificationNo?: string;
  nettingNo?: string;
}) => record.verificationNo || record.nettingNo || '-';
