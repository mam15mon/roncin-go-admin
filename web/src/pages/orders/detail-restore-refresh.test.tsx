import { act, fireEvent, screen } from '@testing-library/react';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import {
  OrderAllowedAction,
  OrderBusinessType,
  OrderTerminationStatus,
  SeaDocumentStructure,
} from '@/enums.generated';
import {
  bundle,
  getOrderReadMock,
  getPortSaveState,
  renderDetail,
  resetPortSaveState,
} from './detail-port-save.fixtures';
import { confirmOrderTermination } from './order-detail-transitions';

vi.mock('./order-detail-transitions', () => ({
  confirmOrderTermination: vi.fn().mockReturnValue(true),
  confirmOrderFlow: vi.fn(),
  confirmOrderClosure: vi.fn(),
}));
const state = getPortSaveState();
const notice = '这个订单已经发生变动，请重新刷新。';
beforeEach(() => {
  resetPortSaveState();
  vi.spyOn(document, 'hasFocus').mockReturnValue(true);
  vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible');
  state.order = {
    ...state.order,
    businessType: OrderBusinessType.BUSINESS_TYPE_SE,
    terminationStatus:
      OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED,
    seaMasterBill: undefined,
    seaDocumentLinkVersion: undefined,
    seaDocumentStructure: undefined,
    allowedActions: [
      OrderAllowedAction.ORDER_ALLOWED_ACTION_CANCEL_TERMINATION,
    ],
  };
});
afterEach(() => vi.restoreAllMocks());

it('退关后恢复原关联刷新成功，不误报本人变动且更晚外部更新仍提示', async () => {
  const view = renderDetail();
  expect(screen.getByText('已退关')).toBeVisible();
  fireEvent.click(screen.getByRole('button', { name: '恢复订单' }));
  const complete = vi.mocked(confirmOrderTermination).mock.calls[0][3];
  const restored: API.Order = {
    ...state.order,
    version: '5',
    terminationStatus: OrderTerminationStatus.ORDER_TERMINATION_STATUS_ACTIVE,
    seaMasterBill: {
      masterNo: 'MBL123',
      masterBillId: 'mbl-1',
      version: '4',
      shippingLineId: 'carrier-1',
      transportExecutionId: 'te-1',
      transportExecutionVersion: '2',
    },
    seaDocumentLinkVersion: '1',
    seaDocumentStructure: SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
    allowedActions: [OrderAllowedAction.ORDER_ALLOWED_ACTION_EDIT],
  };
  state.loadData.mockImplementation(async () => {
    state.order = restored;
    return bundle();
  });
  await act(async () => {
    await complete(restored);
  });
  expect(screen.queryByText('已退关')).toBeNull();
  expect(screen.queryByText(notice)).toBeNull();
  await act(async () => {
    await view.queryClient.refetchQueries({
      queryKey: ['orders', 'change-notice'],
    });
  });
  expect(screen.queryByText(notice)).toBeNull();
  getOrderReadMock().mockResolvedValue({ data: { ...restored, version: '6' } });
  await act(async () => {
    await view.queryClient.refetchQueries({
      queryKey: ['orders', 'change-notice'],
    });
  });
  expect(await screen.findByText(notice)).toBeVisible();
});
