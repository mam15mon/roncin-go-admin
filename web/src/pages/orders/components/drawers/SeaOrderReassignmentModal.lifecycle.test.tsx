import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  candidate,
  deferred,
  enterMasterNo,
  execute,
  fillConfirmation,
  matchCandidate,
  openExecutionConfirmation,
  outerConfirmButton,
  preview,
  renderReassignment,
  setupReassignmentTests,
  validPreview,
} from './SeaOrderReassignmentModal.fixtures';

setupReassignmentTests();

describe('改配弹窗会话隔离', () => {
  it('关闭重开同一订单后重新读取版本，旧会话预览不能回填', async () => {
    const previous = deferred<API.PreviewSeaOrderReassignmentResponse>();
    preview.mockImplementationOnce(() => previous.promise);
    const { rerenderModal } = renderReassignment();
    enterMasterNo();
    await waitFor(() => expect(preview).toHaveBeenCalledOnce());
    rerenderModal({ open: false });
    preview.mockImplementation(async (_params, body) =>
      validPreview(body.target, '30'),
    );
    rerenderModal({ open: true });
    expect(screen.getByPlaceholderText('请输入目标提单号')).toHaveValue('');
    expect(outerConfirmButton()).toBeDisabled();
    enterMasterNo();
    await waitFor(() => expect(outerConfirmButton()).toBeEnabled());
    await act(async () =>
      previous.resolve(
        validPreview({ targetType: 'NEW', masterNo: 'OLDSESSION' }, '1'),
      ),
    );
    await fillConfirmation();
    const confirmation = await openExecutionConfirmation();
    fireEvent.click(
      within(confirmation).getByRole('button', { name: '确认改配' }),
    );
    await waitFor(() => expect(execute).toHaveBeenCalledOnce());
    expect(execute.mock.calls[0][1]).toMatchObject({
      expectedOrderVersion: '30',
      target: { masterNo: 'NEWMBL456' },
    });
  });

  it('用户改了目标后，迟到的候选匹配不得覆盖新输入', async () => {
    const previous = deferred<API.MatchSeaMasterBillCandidateResponse>();
    matchCandidate.mockImplementationOnce(() => previous.promise);
    renderReassignment();
    enterMasterNo('EXISTING123');
    fireEvent.click(screen.getByRole('button', { name: '检查/匹配' }));
    await waitFor(() => expect(matchCandidate).toHaveBeenCalledOnce());
    expect(outerConfirmButton()).toBeDisabled();
    enterMasterNo('CURRENTTARGET');
    await waitFor(() => expect(outerConfirmButton()).toBeEnabled());
    await act(async () =>
      previous.resolve({ success: true, matched: true, candidate }),
    );
    expect(screen.getByPlaceholderText('请输入目标提单号')).toHaveValue(
      'CURRENTTARGET',
    );
    expect(
      screen.queryByText('已匹配现有共享母单：EXISTING123'),
    ).not.toBeInTheDocument();
    expect(preview.mock.calls.at(-1)?.[1].target).toMatchObject({
      targetType: 'NEW',
      masterNo: 'CURRENTTARGET',
    });
  });

  it('旧订单执行结果迟到时不关闭或刷新新订单弹窗', async () => {
    const pendingWrite = deferred<API.ExecuteSeaOrderReassignmentResponse>();
    execute.mockImplementationOnce(() => pendingWrite.promise);
    const { props, rerenderModal } = renderReassignment();
    enterMasterNo();
    await fillConfirmation();
    const confirmation = await openExecutionConfirmation();
    fireEvent.click(
      within(confirmation).getByRole('button', { name: '确认改配' }),
    );
    await waitFor(() => expect(execute).toHaveBeenCalledOnce());
    rerenderModal({ orderId: 'new-order', orderNo: 'SE20261106002' });
    enterMasterNo('NEWORDERTARGET');
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
