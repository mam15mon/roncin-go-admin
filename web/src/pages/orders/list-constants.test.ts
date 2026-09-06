import { describe, expect, it } from 'vitest';
import {
  OrderClosureStatus,
  OrderTerminationStatus,
} from '@/enums.generated';
import { lifecycleFiltersByStage } from './list-constants';

describe('订单列表生命周期筛选配置', () => {
  it('使用生成枚举构造生命周期筛选值', () => {
    expect(lifecycleFiltersByStage.unreturned).toEqual({
      terminationStatus:
        OrderTerminationStatus.ORDER_TERMINATION_STATUS_ACTIVE,
      closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_OPEN,
    });
    expect(lifecycleFiltersByStage.returned.terminationStatus).toBe(
      OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED,
    );
    expect(lifecycleFiltersByStage.completed.closureStatus).toBe(
      OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED,
    );
  });
});
