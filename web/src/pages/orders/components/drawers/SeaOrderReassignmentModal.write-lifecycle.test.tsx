import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
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
  it('详情刷新完成前保持提交中，完成后才提示成功并关闭', async () => {
    const refresh = deferred<void>();
    const { props } = renderReassignment({
      onSuccess: vi.fn(() => refresh.promise),
    });
    await enterMasterNo();
    await fillConfirmation();
    const confirmation = await openExecutionConfirmation();
    fireEvent.click(
      within(confirmation).getByRole('button', { name: '确认改配' }),
    );
    await waitFor(() => expect(props.onSuccess).toHaveBeenCalledOnce());
    expect(props.onClose).not.toHaveBeenCalled();
    expect(
      screen.queryByText('整票改配成功！航程与母单已更新'),
    ).not.toBeInTheDocument();
    expect(
      within(confirmation).getByRole('button', { name: '返回修改' }),
    ).toBeDisabled();
    await act(async () => refresh.resolve());
    await waitFor(() => expect(props.onClose).toHaveBeenCalledOnce());
    expect(
      screen.getByText('整票改配成功！航程与母单已更新'),
    ).toBeInTheDocument();
    expect(execute).toHaveBeenCalledOnce();
  });

  it('改配写入成功但刷新失败时提示已成功，不误报执行失败', async () => {
    const { props } = renderReassignment({
      onSuccess: vi
        .fn()
        .mockRejectedValue(new Error('详情刷新失败，请手动刷新。')),
    });
    await enterMasterNo();
    await fillConfirmation();
    const confirmation = await openExecutionConfirmation();
    fireEvent.click(
      within(confirmation).getByRole('button', { name: '确认改配' }),
    );
    expect(
      await screen.findByText(
        '整票改配已成功，详情刷新失败，请手动刷新。请勿重复提交改配。',
      ),
    ).toBeInTheDocument();
    expect(
      screen.queryByText('整票改配成功！航程与母单已更新'),
    ).not.toBeInTheDocument();
    expect(props.onClose).toHaveBeenCalledOnce();
    expect(execute).toHaveBeenCalledOnce();
  });

  it('详情刷新迟到时不关闭新订单弹窗或弹出旧订单成功提示', async () => {
    const refresh = deferred<void>();
    const { props, rerenderModal } = renderReassignment({
      onSuccess: vi.fn(() => refresh.promise),
    });
    await enterMasterNo();
    await fillConfirmation();
    const confirmation = await openExecutionConfirmation();
    fireEvent.click(
      within(confirmation).getByRole('button', { name: '确认改配' }),
    );
    await waitFor(() => expect(props.onSuccess).toHaveBeenCalledOnce());
    rerenderModal({ orderId: 'new-order' });
    await enterMasterNo('NEWORDERTARGET');
    await act(async () => refresh.resolve());
    expect(props.onClose).not.toHaveBeenCalled();
    expect(
      screen.queryByText('整票改配成功！航程与母单已更新'),
    ).not.toBeInTheDocument();
    expect(screen.getByPlaceholderText('请输入目标提单号')).toHaveValue(
      'NEWORDERTARGET',
    );
  });

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
