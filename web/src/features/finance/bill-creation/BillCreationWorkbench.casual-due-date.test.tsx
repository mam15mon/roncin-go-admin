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

describe('散客建账到期提醒', () => {
  beforeEach(() => resetWorkbenchMocks());
  it('散客指定晚于创建日的日期显示预警，仍可生成账单', async () => {
    mocks.preview.mockResolvedValue({
      data: [
        {
          ...singlePreviewGroup,
          billDate: '2026-11-18',
          isCasual: true,
          defaultPaymentTermsDays: 0,
          groupToken: 'casual-token',
        },
      ],
    });
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
    expect(await screen.findByText('当天到期 · 现结')).toBeInTheDocument();
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(2));
    await waitFor(() =>
      expect(
        screen.getByText('默认结算账户｜测试银行 · 6222｜CNY'),
      ).toBeVisible(),
    );
    fireEvent.click(screen.getByRole('button', { name: '调整到期日' }));
    fireEvent.click(screen.getByRole('radio', { name: '指定日期' }));
    const input = screen.getByLabelText('指定到期日');
    fireEvent.change(input, { target: { value: '2026-11-28' } });
    fireEvent.keyDown(input, { key: 'Enter', code: 'Enter' });
    fireEvent.blur(input);
    expect(
      await screen.findByText(/当前到期日为 2026-11-28（账单日后 10 天）/),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /生成 1 张账单/ }));
    await waitFor(() => expect(mocks.create).toHaveBeenCalledTimes(1));
    expect(mocks.create.mock.calls[0][0].groups[0]).toHaveProperty(
      'dueDate',
      '2026-11-28',
    );
  });
});
