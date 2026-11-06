import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  deferred,
  enterMasterNo,
  execute,
  fillConfirmation,
  openExecutionConfirmation,
  renderReassignment,
  setupReassignmentTests,
} from './SeaOrderReassignmentModal.fixtures';

setupReassignmentTests();
describe('改配写响应的订单会话隔离', () => {
  it('旧订单执行结果迟到时不关闭或刷新新订单弹窗', async () => {
    const pendingWrite = deferred<API.ExecuteSeaOrderReassignmentResponse>();
    execute.mockImplementationOnce(() => pendingWrite.promise);
    const { props, rerenderModal } = renderReassignment();
    await enterMasterNo();
    await fillConfirmation();
    const confirmation = await openExecutionConfirmation();
    fireEvent.click(
      within(confirmation).getByRole('button', { name: '确认改配' }),
    );
    await waitFor(() => expect(execute).toHaveBeenCalledOnce());
    rerenderModal({ orderId: 'new-order', orderNo: 'SE20261106002' });
    await enterMasterNo('NEWORDERTARGET');
    await act(async () =>
      pendingWrite.resolve({
        success: true,
        data: { orderId: 'order-123', reassignmentEventId: 'previous-event' },
      }),
    );
    expect(props.onSuccess).not.toHaveBeenCalled();
    expect(props.onClose).not.toHaveBeenCalled();
    expect(screen.getByPlaceholderText('请输入目标提单号')).toHaveValue(
      'NEWORDERTARGET',
    );
    expect(
      screen.queryByText('整票改配成功！航程与母单已更新'),
    ).not.toBeInTheDocument();
  });
});
