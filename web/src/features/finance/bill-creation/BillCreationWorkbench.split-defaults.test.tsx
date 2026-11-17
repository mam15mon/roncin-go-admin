import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  BillCreationWorkbench,
  mocks,
  resetWorkbenchMocks,
  singlePreviewGroup,
} from './BillCreationWorkbench.fixtures';

describe('批量建账拆分默认值', () => {
  beforeEach(() => {
    resetWorkbenchMocks();
    mocks.preview.mockResolvedValue({ data: [singlePreviewGroup] });
  });

  it('手动选择费用后，订单和税率默认拆分并传入预览', async () => {
    renderWithClient(
      <App>
        <BillCreationWorkbench open onClose={vi.fn()} />
      </App>,
    );
    fireEvent.click(await screen.findByRole('button', { name: '费用A' }));
    fireEvent.click(screen.getByRole('button', { name: '下一步' }));
    const switches = screen.getAllByRole('switch');
    expect(switches).toHaveLength(2);
    for (const toggle of switches) expect(toggle).toBeChecked();
    fireEvent.click(screen.getByRole('button', { name: '下一步' }));
    await waitFor(() =>
      expect(mocks.preview).toHaveBeenCalledWith(
        expect.objectContaining({
          groupingPolicy: expect.objectContaining({
            splitByOrder: true,
            splitByTaxRate: true,
          }),
        }),
        expect.anything(),
      ),
    );
  });

  it('允许关闭拆分，关闭工作台后重新打开恢复默认启用', async () => {
    const onClose = vi.fn();
    const view = (open: boolean) => (
      <App>
        <BillCreationWorkbench
          open={open}
          initialFeeIds={['fee-A']}
          initialOrganizationId="A"
          onClose={onClose}
        />
      </App>
    );
    const { rerender } = renderWithClient(view(true));
    await screen.findByDisplayValue('结算单位甲');
    for (const toggle of screen.getAllByRole('switch')) fireEvent.click(toggle);
    await waitFor(() =>
      expect(mocks.preview).toHaveBeenLastCalledWith(
        expect.objectContaining({
          groupingPolicy: expect.objectContaining({
            splitByOrder: false,
            splitByTaxRate: false,
          }),
        }),
        expect.anything(),
      ),
    );
    rerender(view(false));
    rerender(view(true));
    await screen.findByDisplayValue('结算单位甲');
    for (const toggle of screen.getAllByRole('switch'))
      expect(toggle).toBeChecked();
    await waitFor(() =>
      expect(mocks.preview).toHaveBeenLastCalledWith(
        expect.objectContaining({
          groupingPolicy: expect.objectContaining({
            splitByOrder: true,
            splitByTaxRate: true,
          }),
        }),
        expect.anything(),
      ),
    );
  });
});
