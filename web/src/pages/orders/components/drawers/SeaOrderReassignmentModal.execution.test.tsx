import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
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

describe('改配执行失败保留确认意图', () => {
  it('版本冲突保留输入，明确重新预览后才采用新版本提交', async () => {
    execute.mockRejectedValueOnce(new Error('已被更新，请刷新后重试'));
    const { props } = renderReassignment();
    enterMasterNo();
    await fillConfirmation();
    const confirmation = await openExecutionConfirmation();
    fireEvent.click(
      within(confirmation).getByRole('button', { name: '确认改配' }),
    );
    await waitFor(() =>
      expect(screen.getByText('已被更新，请刷新后重试')).toBeVisible(),
    );
    expect(props.onSuccess).not.toHaveBeenCalled();
    expect(props.onClose).not.toHaveBeenCalled();
    expect(screen.getByPlaceholderText('请输入目标提单号')).toHaveValue(
      'NEWMBL456',
    );
    expect(confirmation).toBeVisible();
    fireEvent.click(
      within(confirmation).getByRole('button', { name: '返回修改' }),
    );
    preview.mockImplementation(async (_params, body) =>
      validPreview(body.target, '8'),
    );
    fireEvent.click(screen.getByRole('button', { name: '重新预览' }));
    await waitFor(() => expect(preview).toHaveBeenCalledTimes(2));
    const refreshedConfirmation = await openExecutionConfirmation();
    fireEvent.click(
      within(refreshedConfirmation).getByRole('button', { name: '确认改配' }),
    );
    await waitFor(() => expect(props.onSuccess).toHaveBeenCalledOnce());
    expect(execute.mock.calls[0][1].expectedOrderVersion).toBe('7');
    expect(execute.mock.calls[1][1].expectedOrderVersion).toBe('8');
    expect(execute.mock.calls[1][1].idempotencyKey).not.toBe(
      execute.mock.calls[0][1].idempotencyKey,
    );
  });

  it('无有效执行响应时保留确认框，不能假报成功或关闭', async () => {
    execute.mockResolvedValueOnce(
      undefined as unknown as API.ExecuteSeaOrderReassignmentResponse,
    );
    const { props } = renderReassignment();
    enterMasterNo();
    await fillConfirmation();
    const confirmation = await openExecutionConfirmation();
    fireEvent.click(
      within(confirmation).getByRole('button', { name: '确认改配' }),
    );
    await waitFor(() =>
      expect(
        screen.getByText('执行改配未返回有效结果，请核对后重试'),
      ).toBeVisible(),
    );
    expect(confirmation).toBeVisible();
    expect(props.onSuccess).not.toHaveBeenCalled();
    expect(props.onClose).not.toHaveBeenCalled();
    fireEvent.click(
      within(confirmation).getByRole('button', { name: '确认改配' }),
    );
    await waitFor(() => expect(props.onSuccess).toHaveBeenCalledOnce());
    expect(execute.mock.calls[1][1]).toEqual(execute.mock.calls[0][1]);
  });

  it('匹配接口省略 false 字段时仍按未命中处理，允许新母单预览', async () => {
    matchCandidate.mockResolvedValueOnce({ success: true });
    renderReassignment();
    enterMasterNo();
    fireEvent.click(screen.getByRole('button', { name: '检查/匹配' }));
    await waitFor(() =>
      expect(
        screen.getByText('未找到已有母单，将按填写的航程新建母单'),
      ).toBeVisible(),
    );
    await waitFor(() => expect(outerConfirmButton()).toBeEnabled());
    expect(
      screen.queryByText('匹配候选母单失败，请重试'),
    ).not.toBeInTheDocument();
  });
});
