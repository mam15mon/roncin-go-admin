import type { StatusMeta } from '@/constants/statusMeta';
import {
  FinanceBillBusinessStatus,
  FinanceBillFeeLockFact,
  FinanceBillInvoiceProgress,
  FinanceBillSettlementProgress,
  FinanceBillStatus,
} from '@/enums.generated';

/**
 * 账单状态展示元数据：账单列表、详情抽屉与建账结果表共用的唯一映射。
 * 账单状态是独立实体状态机（费用无独立状态，是否建账由有效账单关联决定），
 * 不得与其他实体状态合并；标签渲染复用 @/constants/statusMeta 的
 * statusTag/statusText 通用函数。
 * 账单免人工确认：新建即为 CONFIRMED（展示为「正常」），不再产生草稿；
 * 草稿枚举仅保留在契约中，不作为筛选或展示选项。
 */
export const billStatusMeta: Record<number, StatusMeta> = {
  [FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED]: {
    text: '正常',
    color: 'blue',
  },
  [FinanceBillStatus.FINANCE_BILL_STATUS_CANCELLED]: {
    text: '已取消',
    color: 'red',
  },
};

/** 当前业务待办由服务端投影，筛选、列表与导出共用文案。 */
export const billBusinessStatusMeta: Record<number, StatusMeta> = {
  [FinanceBillBusinessStatus.FINANCE_BILL_BUSINESS_STATUS_UNINVOICED]: {
    text: '未开票',
    color: 'default',
  },
  [FinanceBillBusinessStatus.FINANCE_BILL_BUSINESS_STATUS_PARTIALLY_INVOICED]: {
    text: '部分开票',
    color: 'orange',
  },
  [FinanceBillBusinessStatus.FINANCE_BILL_BUSINESS_STATUS_UNVERIFIED]: {
    text: '未核销',
    color: 'default',
  },
  [FinanceBillBusinessStatus.FINANCE_BILL_BUSINESS_STATUS_PARTIALLY_VERIFIED]: {
    text: '部分核销',
    color: 'orange',
  },
  [FinanceBillBusinessStatus.FINANCE_BILL_BUSINESS_STATUS_COMPLETED]: {
    text: '已完成',
    color: 'success',
  },
};

/**
 * 账单结算/开票进度与完成状态：服务端读取时推导（零额账单无需核销、无需开票即完成），
 * 页面只展示，不在前端重算。未经投影的账单（如建账响应）为 UNSPECIFIED，不展示。
 * 「无需开票」只说明无需开票，不伪装为已开票。
 */
export const billSettlementProgressMeta: Record<number, StatusMeta> = {
  [FinanceBillSettlementProgress.FINANCE_BILL_SETTLEMENT_PROGRESS_UNSETTLED]: {
    text: '未结清',
    color: 'default',
  },
  [FinanceBillSettlementProgress.FINANCE_BILL_SETTLEMENT_PROGRESS_PARTIAL]: {
    text: '部分结清',
    color: 'orange',
  },
  [FinanceBillSettlementProgress.FINANCE_BILL_SETTLEMENT_PROGRESS_SETTLED]: {
    text: '已结清',
    color: 'green',
  },
};

export const billInvoiceProgressMeta: Record<number, StatusMeta> = {
  [FinanceBillInvoiceProgress.FINANCE_BILL_INVOICE_PROGRESS_NONE]: {
    text: '未开票',
    color: 'default',
  },
  [FinanceBillInvoiceProgress.FINANCE_BILL_INVOICE_PROGRESS_PARTIAL]: {
    text: '部分开票',
    color: 'orange',
  },
  [FinanceBillInvoiceProgress.FINANCE_BILL_INVOICE_PROGRESS_FULL]: {
    text: '已开票',
    color: 'green',
  },
  [FinanceBillInvoiceProgress.FINANCE_BILL_INVOICE_PROGRESS_NOT_REQUIRED]: {
    text: '无需开票',
    color: 'cyan',
  },
};

const billFeeLockFactText: Record<number, string> = {
  [FinanceBillFeeLockFact.FINANCE_BILL_FEE_LOCK_FACT_INVOICE]: '发票',
  [FinanceBillFeeLockFact.FINANCE_BILL_FEE_LOCK_FACT_VERIFICATION]: '核销',
  [FinanceBillFeeLockFact.FINANCE_BILL_FEE_LOCK_FACT_NETTING]: '已确认对冲',
  [FinanceBillFeeLockFact.FINANCE_BILL_FEE_LOCK_FACT_NETTING_DRAFT]: '草稿对冲',
  [FinanceBillFeeLockFact.FINANCE_BILL_FEE_LOCK_FACT_INVOICE_EXEMPTION]:
    '无需发票标记',
};

/** 费用锁定事实的中文名称，按服务端固定顺序；ProtoJSON 省略空数组时返回空列表。 */
export function billFeeLockFactLabels(bill?: API.FinanceBill): string[] {
  return (bill?.feeLockFacts ?? [])
    .map((fact) => billFeeLockFactText[fact])
    .filter((label): label is string => Boolean(label));
}

/**
 * 账单字段是否受保护：存在发票、核销、已确认对冲等事实时只能改备注、对账标题与到期日。
 * 草稿对冲只阻止增删费用、不限制字段修改，与服务端受保护口径一致；最终以服务端校验为准。
 */
export function isBillFieldProtected(bill?: API.FinanceBill): boolean {
  return (bill?.feeLockFacts ?? []).some(
    (fact) =>
      fact !== FinanceBillFeeLockFact.FINANCE_BILL_FEE_LOCK_FACT_NETTING_DRAFT,
  );
}
