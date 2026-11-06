// @vitest-environment jsdom
// HappyDOM 会误匹配 Ant Design 消息堆叠的复合选择器，改用 jsdom 验证真实可见性。

import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  enterMasterNo,
  execute,
  fillConfirmation,
  matchCandidate,
  openExecutionConfirmation,
  outerConfirmButton,
  renderReassignment,
  setupReassignmentTests,
} from './SeaOrderReassignmentModal.fixtures';

setupReassignmentTests();

describe('改配执行失败保留确认意图', () => {
  it('无有效执行响应时保留确认框，不能假报成功或关闭', async () => {
    execute.mockResolvedValueOnce(
      undefined as unknown as API.ExecuteSeaOrderReassignmentResponse,
    );
    const { props } = renderReassignment();
    await enterMasterNo();
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
    await enterMasterNo();
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
