import type { StatusMeta } from '@/constants/statusMeta';
import { FinanceBillStatus } from '@/enums.generated';

/**
 * 账单状态展示元数据：账单列表、详情抽屉与建账结果表共用的唯一映射。
 * 账单状态与费用状态（orderFeeStatusMeta）是不同实体状态机，不得合并；
 * 标签渲染复用 @/constants/statusMeta 的 statusTag/statusText 通用函数。
 */
export const billStatusMeta: Record<number, StatusMeta> = {
  [FinanceBillStatus.FINANCE_BILL_STATUS_DRAFT]: {
    text: '草稿',
    color: 'default',
  },
  [FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED]: {
    text: '已确认',
    color: 'blue',
  },
  [FinanceBillStatus.FINANCE_BILL_STATUS_CANCELLED]: {
    text: '已取消',
    color: 'red',
  },
};
