import type { OrderStatusTabItem } from '@/components/ui';
import { orderFlowStatusMeta, statusText } from '@/constants/statusMeta';
import {
  OrderClosureStatus,
  OrderFlowStatus,
  OrderTerminationStatus,
} from '@/enums.generated';

function flowStatusTab(
  key: string,
  status: OrderFlowStatus,
  badgeColor: string,
): OrderStatusTabItem {
  return {
    key,
    label: statusText(orderFlowStatusMeta, status, '未知状态'),
    badgeColor,
  };
}

export const orderStatusTabs: OrderStatusTabItem[] = [
  { key: 'all', label: '全部订单' },
  flowStatusTab(
    'draft',
    OrderFlowStatus.ORDER_FLOW_STATUS_DRAFT,
    '#d9d9d9',
  ),
  flowStatusTab(
    'booked',
    OrderFlowStatus.ORDER_FLOW_STATUS_BOOKED,
    '#1677ff',
  ),
  flowStatusTab(
    'allocated',
    OrderFlowStatus.ORDER_FLOW_STATUS_SPACE_ALLOCATED,
    '#13c2c2',
  ),
  flowStatusTab(
    'trucking',
    OrderFlowStatus.ORDER_FLOW_STATUS_TRUCKING_ARRANGED,
    '#722ed1',
  ),
  flowStatusTab(
    'cutoff',
    OrderFlowStatus.ORDER_FLOW_STATUS_DOCUMENT_CUTOFF,
    '#eb2f96',
  ),
  flowStatusTab(
    'customs',
    OrderFlowStatus.ORDER_FLOW_STATUS_CUSTOMS_DECLARATION_ARRANGED,
    '#2f54eb',
  ),
  flowStatusTab(
    'released',
    OrderFlowStatus.ORDER_FLOW_STATUS_DOCUMENT_RELEASED,
    '#52c41a',
  ),
  { key: 'terminating', label: '退关中', badgeColor: '#fa8c16' },
  { key: 'terminated', label: '已退关', badgeColor: '#ff4d4f' },
  { key: 'completed', label: '已完结', badgeColor: '#52c41a' },
  { key: 'abnormal', label: '异常挂起', badgeColor: '#ff4d4f' },
];

export const lifecycleFiltersByStage: Record<
  string,
  {
    flowStatus?: number;
    terminationStatus?: number;
    closureStatus?: number;
    hasActiveException?: boolean;
  }
> = {
  draft: { flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_DRAFT },
  booked: { flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_BOOKED },
  allocated: {
    flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_SPACE_ALLOCATED,
  },
  trucking: {
    flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_TRUCKING_ARRANGED,
  },
  cutoff: {
    flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_DOCUMENT_CUTOFF,
  },
  customs: {
    flowStatus:
      OrderFlowStatus.ORDER_FLOW_STATUS_CUSTOMS_DECLARATION_ARRANGED,
  },
  released: {
    flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_DOCUMENT_RELEASED,
  },
  terminating: {
    terminationStatus:
      OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATING,
  },
  terminated: {
    terminationStatus:
      OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED,
  },
  completed: {
    closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED,
  },
  abnormal: { hasActiveException: true },
  unreturned: {
    terminationStatus:
      OrderTerminationStatus.ORDER_TERMINATION_STATUS_ACTIVE,
    closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_OPEN,
  },
  returned: {
    terminationStatus:
      OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED,
  },
};
