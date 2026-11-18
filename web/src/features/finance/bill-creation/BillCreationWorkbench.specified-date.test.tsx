import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { App, ConfigProvider } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  BillCreationWorkbench,
  mocks,
  resetWorkbenchMocks,
  singlePreviewGroup,
} from './BillCreationWorkbench.fixtures';

describe('批量建账指定到期日', () => {
  beforeEach(() => {
    resetWorkbenchMocks();
    const group = {
      ...singlePreviewGroup,
      billDate: '2026-09-01',
      defaultPaymentTermsDays: 30,
      groupToken: 'system-date-token',
    };
    mocks.preview
      .mockResolvedValueOnce({
        data: [
          { ...group, configurationComplete: false, groupToken: undefined },
        ],
      })
      .mockResolvedValue({ data: [group] });
  });
  it('切换指定日期并刷新预览后保留输入，提交只带 dueDate', async () => {
    renderWithClient(
      <ConfigProvider theme={{ token: { motion: false } }}>
        <App>
          <BillCreationWorkbench
            open
            initialFeeIds={['fee-A']}
            initialOrganizationId="A"
            onClose={vi.fn()}
          />
        </App>
      </ConfigProvider>,
    );
    expect(
      await screen.findByText('2026-10-01 到期 · 账期 30 天'),
    ).toBeInTheDocument();
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(2));
    await waitFor(() =>
      expect(
        screen.queryByText('这张账单还有必填资料没补全'),
      ).not.toBeInTheDocument(),
    );
    fireEvent.click(screen.getByRole('button', { name: '调整到期日' }));
    fireEvent.click(screen.getByRole('radio', { name: '指定日期' }));
    const input = screen.getByLabelText('指定到期日');
    fireEvent.change(input, { target: { value: '2026-12-01' } });
    fireEvent.keyDown(input, { key: 'Enter', code: 'Enter' });
    fireEvent.blur(input);
    expect(
      await screen.findByText('2026-12-01 到期 · 指定日期'),
    ).toBeInTheDocument();
    mocks.preview.mockResolvedValueOnce({
      data: [
        {
          ...singlePreviewGroup,
          billDate: '2026-09-01',
          defaultPaymentTermsDays: 30,
          settlementPartyName: '结算单位甲（刷新后）',
          groupToken: 'refreshed-date-token',
        },
      ],
    });
    fireEvent.click(screen.getByRole('button', { name: /刷新账单/ }));
    expect(await screen.findByText('结算单位甲（刷新后）')).toBeVisible();
    expect(mocks.preview).toHaveBeenCalledTimes(3);
    expect(screen.getByText('2026-12-01 到期 · 指定日期')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /生成 1 张账单/ }));
    await waitFor(() => expect(mocks.create).toHaveBeenCalledTimes(1));
    const group = mocks.create.mock.calls[0][0].groups[0];
    expect(group).toHaveProperty('groupToken', 'refreshed-date-token');
    expect(group).toHaveProperty('dueDate', '2026-12-01');
    expect(group).not.toHaveProperty('paymentTermsDays');
  });
});
