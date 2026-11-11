import { act, fireEvent, screen } from '@testing-library/react';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import {
  OrderAllowedAction,
  OrderClosureStatus,
  OrderFlowStatus,
  OrderTerminationStatus,
} from '@/enums.generated';
import {
  bundle,
  getOrderReadMock,
  getPortSaveState,
  renderDetail,
  resetPortSaveState,
} from './detail-port-save.fixtures';
import {
  confirmOrderClosure,
  confirmOrderFlow,
  confirmOrderTermination,
} from './order-detail-transitions';

vi.mock('./order-detail-transitions', () => ({
  confirmOrderFlow: vi.fn().mockReturnValue(true),
  confirmOrderTermination: vi.fn().mockReturnValue(true),
  confirmOrderClosure: vi.fn().mockReturnValue(true),
}));

const state = getPortSaveState();
const notice = '这个订单已经发生变动，请重新刷新。';

beforeEach(() => {
  resetPortSaveState();
  vi.spyOn(document, 'hasFocus').mockReturnValue(true);
  vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible');
  state.order = {
    ...state.order,
    flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_BOOKED,
    terminationStatus: OrderTerminationStatus.ORDER_TERMINATION_STATUS_ACTIVE,
    closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_OPEN,
    allowedActions: [
      OrderAllowedAction.ORDER_ALLOWED_ACTION_TRANSITION_FLOW,
      OrderAllowedAction.ORDER_ALLOWED_ACTION_START_TERMINATION,
      OrderAllowedAction.ORDER_ALLOWED_ACTION_CLOSE,
    ],
    allowedTargetFlowStatuses: [
      OrderFlowStatus.ORDER_FLOW_STATUS_SPACE_ALLOCATED,
    ],
  };
});
afterEach(() => vi.restoreAllMocks());

it.each([
  [
    '进度',
    '流转到已配舱',
    confirmOrderFlow,
    { flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_SPACE_ALLOCATED },
  ],
  [
    '退关',
    '发起退关',
    confirmOrderTermination,
    {
      terminationStatus:
        OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATING,
    },
  ],
  [
    '完结',
    '完结订单',
    confirmOrderClosure,
    { closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED },
  ],
] as const)(
  '本人执行%s后不误报，后续外部修改仍提示',
  async (_label, button, confirm, change) => {
    const view = renderDetail();
    fireEvent.change(screen.getByLabelText('起运港'), {
      target: { value: 'unsaved-port' },
    });
    fireEvent.click(screen.getByRole('button', { name: button }));
    const complete = vi.mocked(confirm).mock.calls[0][3];
    const written = { ...state.order, ...change, version: '5' };
    state.loadData.mockImplementation(async () => {
      state.order = written;
      return bundle();
    });
    await act(async () => {
      await complete(written);
    });
    await act(async () => {
      await view.queryClient.refetchQueries({
        queryKey: ['orders', 'change-notice'],
      });
    });
    expect(screen.queryByText(notice)).toBeNull();
    expect(screen.getByLabelText('起运港')).toHaveValue('unsaved-port');
    expect(state.form?.getFieldValue('version')).toBe('4');
    getOrderReadMock().mockResolvedValue({
      data: { ...written, version: '6' },
    });
    await act(async () => {
      await view.queryClient.refetchQueries({
        queryKey: ['orders', 'change-notice'],
      });
    });
    expect(await screen.findByText(notice)).toBeVisible();
  },
);
