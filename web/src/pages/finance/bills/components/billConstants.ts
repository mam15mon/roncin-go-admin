import type { Dayjs } from 'dayjs';

export const statusOptions: Record<string, { text: string; color: string }> = {
  DRAFT: { text: '草稿', color: 'gold' },
  CONFIRMED: { text: '已确认', color: 'blue' },
  CANCELLED: { text: '已取消', color: 'default' },
};

export type BillFormValues = {
  statementTitle: string;
  billDate: Dayjs;
  paymentTermsDays?: number;
  note?: string;
};
