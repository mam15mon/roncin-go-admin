import { describe, expect, it } from 'vitest';
import {
  OrderClosureStatus,
  OrderFlowStatus,
  OrderTerminationStatus,
} from '@/enums.generated';
import { lifecycleFiltersByStage, orderStatusTabs } from './list-constants';

describe('订单列表状态配置', () => {
  it('从流程状态元数据生成流程标签', () => {
    expect(orderStatusTabs.slice(0, 8).map(({ key, label }) => ({ key, label })))
      .toEqual([
        { key: 'all', label: '全部订单' },
        { key: 'draft', label: '草稿' },
        { key: 'booked', label: '已订舱' },
        { key: 'allocated', label: '已配舱' },
        { key: 'trucking', label: '拖车已安排' },
        { key: 'cutoff', label: '已截单' },
        { key: 'customs', label: '报关已安排' },
        { key: 'released', label: '已放单' },
      ]);
  });

  it('使用生成枚举构造生命周期筛选值', () => {
    expect(lifecycleFiltersByStage.draft.flowStatus).toBe(
      OrderFlowStatus.ORDER_FLOW_STATUS_DRAFT,
    );
    expect(lifecycleFiltersByStage.terminated.terminationStatus).toBe(
      OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED,
    );
    expect(lifecycleFiltersByStage.completed.closureStatus).toBe(
      OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED,
    );
  });
});
