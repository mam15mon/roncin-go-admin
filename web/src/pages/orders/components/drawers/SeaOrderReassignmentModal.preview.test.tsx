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
  outerConfirmButton,
  preview,
  renderReassignment,
  setupReassignmentTests,
  validPreview,
} from './SeaOrderReassignmentModal.fixtures';

setupReassignmentTests();

describe('改配预览与当前目标绑定', () => {
  it('预览失败展示具体业务原因，确认不可用且保留输入', async () => {
    preview.mockRejectedValue(new Error('目标 MBL 冲突或不可用'));
    renderReassignment();
    enterMasterNo();
    await waitFor(() =>
      expect(screen.getByText('目标 MBL 冲突或不可用')).toBeVisible(),
    );
    expect(outerConfirmButton()).toBeDisabled();
    fireEvent.click(outerConfirmButton());
    expect(execute).not.toHaveBeenCalled();
    expect(
      screen.queryByText('未获取到有效的订单或母单关联版本，请刷新重试！'),
    ).not.toBeInTheDocument();
    expect(screen.getByPlaceholderText('请输入目标提单号')).toHaveValue(
      'NEWMBL456',
    );
  });

  it('服务端返回不可执行预览或缺失版本时不允许确认', async () => {
    preview.mockResolvedValueOnce({
      success: true,
      data: {
        isValid: false,
        errors: ['请先移除本票共享箱分配'],
        orderVersion: '7',
        currentLinkVersion: '4',
      },
    });
    renderReassignment();
    enterMasterNo();
    await waitFor(() =>
      expect(screen.getByText('请先移除本票共享箱分配')).toBeVisible(),
    );
    expect(outerConfirmButton()).toBeDisabled();
    preview.mockResolvedValueOnce({ success: true, data: { isValid: true } });
    fireEvent.click(screen.getByRole('button', { name: '重新预览' }));
    await waitFor(() =>
      expect(
        screen.getByText('改配预览缺少订单或关联版本，请重新预览'),
      ).toBeVisible(),
    );
    expect(outerConfirmButton()).toBeDisabled();
  });

  it('目标变化立即阻止提交，旧预览迟到也不能替换新目标和版本', async () => {
    const oldPreview = deferred<API.PreviewSeaOrderReassignmentResponse>();
    const newPreview = deferred<API.PreviewSeaOrderReassignmentResponse>();
    preview.mockImplementation((_params, body) =>
      body.target.masterNo === 'OLDTARGET'
        ? oldPreview.promise
        : newPreview.promise,
    );
    renderReassignment();
    enterMasterNo('OLDTARGET');
    await waitFor(() => expect(preview).toHaveBeenCalledOnce());
    enterMasterNo('CURRENTTARGET');
    await waitFor(() => expect(preview).toHaveBeenCalledTimes(2));
    expect(outerConfirmButton()).toBeDisabled();
    await act(async () =>
      newPreview.resolve(
        validPreview({ targetType: 'NEW', masterNo: 'CURRENTTARGET' }, '20'),
      ),
    );
    await waitFor(() => expect(outerConfirmButton()).toBeEnabled());
    await act(async () =>
      oldPreview.resolve({
        success: true,
        data: {
          isValid: false,
          errors: ['过期目标错误'],
          orderVersion: '1',
          currentLinkVersion: '1',
        },
      }),
    );
    expect(outerConfirmButton()).toBeEnabled();
    expect(screen.queryByText('过期目标错误')).not.toBeInTheDocument();
    await fillConfirmation();
    const confirmation = await openExecutionConfirmation();
    fireEvent.click(
      within(confirmation).getByRole('button', { name: '确认改配' }),
    );
    await waitFor(() => expect(execute).toHaveBeenCalledOnce());
    expect(execute.mock.calls[0][1]).toMatchObject({
      expectedOrderVersion: '20',
      target: { masterNo: 'CURRENTTARGET' },
    });
  });
});
