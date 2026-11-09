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
    preview.mockRejectedValue(
      Object.assign(new Error('Request failed with status code 409'), {
        response: {
          status: 409,
          data: {
            success: false,
            code: 409,
            message: '目标 MBL 冲突或不可用',
            reason: 'SEA_ORDER_REASSIGNMENT_TARGET_CONFLICT',
          },
        },
      }),
    );
    renderReassignment();
    await enterMasterNo();
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

  it.each([undefined, false])(
    '不可执行预览（isValid=%s）或缺失版本时不允许确认',
    async (isValid) => {
      preview.mockResolvedValueOnce({
        success: true,
        data: {
          ...(isValid === undefined ? {} : { isValid }),
          errors: ['请先移除本票共享箱分配'],
          orderVersion: '7',
          currentLinkVersion: '4',
        },
      });
      renderReassignment();
      await enterMasterNo();
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
    },
  );

  it('加载中不显示业务否定，成功省略 isValid 和 errors 后显示兜底原因', async () => {
    const pending = deferred<API.PreviewSeaOrderReassignmentResponse>();
    preview.mockReturnValueOnce(pending.promise);
    renderReassignment();
    await enterMasterNo();
    await waitFor(() => expect(preview).toHaveBeenCalledOnce());
    expect(
      screen.queryByText('当前目标不满足改配条件'),
    ).not.toBeInTheDocument();
    await act(async () => pending.resolve({ success: true, data: {} }));
    expect(await screen.findByText('当前目标不满足改配条件')).toBeVisible();
    expect(outerConfirmButton()).toBeDisabled();
    expect(execute).not.toHaveBeenCalled();
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
    await enterMasterNo('OLDTARGET');
    await waitFor(() => expect(preview).toHaveBeenCalledOnce());
    await enterMasterNo('CURRENTTARGET');
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
