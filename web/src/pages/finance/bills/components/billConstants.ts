import type { Dayjs } from 'dayjs';
import { FinanceBillStatus } from '@/enums.generated';

export const statusOptions: Record<number, { text: string; color: string }> = {
  [FinanceBillStatus.FINANCE_BILL_STATUS_DRAFT]: { text: '草稿', color: 'gold' },
  [FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED]: { text: '已确认', color: 'blue' },
  [FinanceBillStatus.FINANCE_BILL_STATUS_CANCELLED]: { text: '已取消', color: 'default' },
};

export type BillFormValues = {
  statementTitle: string;
  billDate: Dayjs;
  paymentTermsDays?: number;
  note?: string;
};
