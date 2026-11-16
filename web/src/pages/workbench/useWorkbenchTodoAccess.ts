import { useAccess } from '@/app/access';
import { ORDER_BUSINESS_ROUTES } from '@/features/orders/navigation';

export type WorkbenchTodoKind = 'fees' | 'abnormal';

/** 逐笔待办复用订单读取权限；审核入口权限不能替代费用读取。 */
export function useWorkbenchTodoAccess() {
  const access = useAccess();
  const canRead = (operation: 'fee.read' | 'abnormal_case.read') =>
    Object.keys(ORDER_BUSINESS_ROUTES).some(
      (kind) =>
        access.canOrder(Number(kind), 'read') &&
        access.canOrder(Number(kind), operation),
    );
  return {
    fees: canRead('fee.read'),
    abnormal: canRead('abnormal_case.read'),
  };
}
