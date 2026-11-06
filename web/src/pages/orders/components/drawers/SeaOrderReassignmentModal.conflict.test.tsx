// @vitest-environment jsdom
// HappyDOM 会误匹配 Ant Design 消息堆叠的复合选择器，改用 jsdom 验证真实可见性。

import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  enterMasterNo,
  execute,
  fillConfirmation,
  openExecutionConfirmation,
  preview,
  renderReassignment,
  setupReassignmentTests,
  validPreview,
} from './SeaOrderReassignmentModal.fixtures';

setupReassignmentTests();
describe('改配版本冲突的显式重新预览', () => {
  it('版本冲突保留输入，明确重新预览后才采用新版本提交', async () => {
    execute.mockRejectedValueOnce(new Error('已被更新，请刷新后重试'));
    const { props } = renderReassignment();
    await enterMasterNo();
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
});
