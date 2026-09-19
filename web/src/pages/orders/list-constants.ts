import { OrderClosureStatus, OrderTerminationStatus } from '@/enums.generated';

export const lifecycleFiltersByStage: Record<
  string,
  {
    terminationStatus?: number;
    closureStatus?: number;
  }
> = {
  completed: {
    closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED,
  },
  unreturned: {
    terminationStatus: OrderTerminationStatus.ORDER_TERMINATION_STATUS_ACTIVE,
    closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_OPEN,
  },
  returned: {
    terminationStatus:
      OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED,
  },
};
