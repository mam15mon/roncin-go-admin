import { useAccess } from '@/app/access';

export type WorkbenchTodoKind = 'fees' | 'abnormal';

/** 逐笔待办复用订单读取权限；审核入口权限不能替代费用读取。 */
export function useWorkbenchTodoAccess() {
  const access = useAccess();
  const canRead = (operation: 'fee.read' | 'abnormal_case.read') =>
    ['SE', 'AE'].some(
      (kind) =>
        access.canOrder(kind, 'read') && access.canOrder(kind, operation),
    );
  return {
    fees: canRead('fee.read'),
    abnormal: canRead('abnormal_case.read'),
  };
}
