import type { StatusMeta } from '@/constants/statusMeta';
import { billBusinessStatusMeta } from '@/features/finance/bill-status';

/** 只展示服务端当前待办投影；未投影时不根据金额或其他进度猜测。 */
export function getBillProgressStatus(
  bill: API.FinanceBill,
): StatusMeta | undefined {
  return bill.businessStatus === undefined
    ? undefined
    : billBusinessStatusMeta[bill.businessStatus];
}
