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

describe('批量建账系统创建日期', () => {
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
  it('展示服务端创建日期并按它预估到期日，预览与提交均不携带人工日期', async () => {
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
    expect(await screen.findByText('2026-09-01')).toBeInTheDocument();
    expect(screen.getByText('创建日期')).toBeInTheDocument();
    expect(
      await screen.findByText('2026-10-01 到期 · 账期 30 天'),
    ).toBeInTheDocument();
    expect(screen.queryByLabelText('创建日期')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('账单日期')).not.toBeInTheDocument();
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(2));
    await waitFor(() =>
      expect(
        screen.queryByText('这张账单还有必填资料没补全'),
      ).not.toBeInTheDocument(),
    );
    for (const [request] of mocks.preview.mock.calls) {
      for (const config of request.groupConfigs ?? [])
        expect(config).not.toHaveProperty('billDate');
    }
    fireEvent.click(screen.getByRole('button', { name: /生成 1 张账单/ }));
    await waitFor(() => expect(mocks.create).toHaveBeenCalledTimes(1));
    const group = mocks.create.mock.calls[0][0].groups[0];
    expect(group.groupToken).toBe('system-date-token');
    expect(group.paymentTermsDays).toBe(30);
    expect(group).not.toHaveProperty('billDate');
  });
});
