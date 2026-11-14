import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React, { useState } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  BillCreationWorkbench,
  mocks,
  resetWorkbenchMocks,
  singlePreviewGroup,
} from './BillCreationWorkbench.fixtures';

describe('批量建账会话隔离', () => {
  beforeEach(resetWorkbenchMocks);

  it('取消固定来源后旧初始化预览不得将新会话推进到账单资料步骤', async () => {
    let resolveInitial:
      | ((value: API.PreviewBillBatchResponse) => void)
      | undefined;
    mocks.preview.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          resolveInitial = resolve;
        }),
    );
    function Harness() {
      const [fixedSource, setFixedSource] = useState(true);
      return (
        <App>
          <button type="button" onClick={() => setFixedSource(false)}>
            改为重新选择费用
          </button>
          <BillCreationWorkbench
            open
            initialFeeIds={fixedSource ? ['fee-A'] : []}
            initialOrganizationId="A"
            onClose={vi.fn()}
          />
        </App>
      );
    }
    renderWithClient(<Harness />);
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(1));
    fireEvent.click(screen.getByRole('button', { name: '改为重新选择费用' }));
    expect(
      await screen.findByRole('button', { name: '费用A' }),
    ).toBeInTheDocument();
    await act(async () => {
      resolveInitial?.({ data: [singlePreviewGroup] });
    });
    expect(screen.getByRole('button', { name: '费用A' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /下一步/ })).toBeInTheDocument();
    expect(screen.queryByText(/生成 0 张账单/)).not.toBeInTheDocument();
    expect(mocks.accounts).not.toHaveBeenCalled();
  });

  it('切换来源后旧提交的迟到成功不覆盖新表单、不触发新会话成功回调', async () => {
    let resolveFirst:
      | ((value: API.CreateBillBatchResponse) => void)
      | undefined;
    mocks.create.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          resolveFirst = resolve;
        }),
    );
    mocks.preview.mockImplementation(
      async ({ feeIds }: API.PreviewBillBatchRequest) => ({
        data: [
          {
            ...singlePreviewGroup,
            groupKey: feeIds[0],
            groupToken: `token-${feeIds[0]}`,
            settlementPartyName:
              feeIds[0] === 'fee-A' ? '原会话单位' : '新会话单位',
            fees: [{ id: feeIds[0] }],
          },
        ],
      }),
    );
    const onCreated = vi.fn();
    function Harness() {
      const [source, setSource] = useState('fee-A');
      return (
        <App>
          <button type="button" onClick={() => setSource('fee-B')}>
            切换来源会话
          </button>
          <BillCreationWorkbench
            open
            initialFeeIds={[source]}
            initialOrganizationId="A"
            onClose={vi.fn()}
            onCreated={onCreated}
          />
        </App>
      );
    }
    renderWithClient(<Harness />);
    await screen.findByDisplayValue('原会话单位');
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(2));
    const button = screen
      .getByText(/生成 1 张账单/)
      .closest('button') as HTMLButtonElement;
    await waitFor(() => expect(button).not.toHaveClass('ant-btn-loading'));
    fireEvent.click(button);
    await waitFor(() => expect(mocks.create).toHaveBeenCalledTimes(1));
    mocks.accounts.mockClear();
    fireEvent.click(screen.getByRole('button', { name: '切换来源会话' }));
    await screen.findByDisplayValue('新会话单位');
    await waitFor(() => expect(mocks.accounts).toHaveBeenCalledTimes(1));
    await act(async () => {
      resolveFirst?.({
        data: { id: 'old-batch', batchNo: 'OLD-BATCH' },
        results: [
          { groupKey: 'fee-A', success: true, bill: { id: 'old-bill' } },
        ],
      });
    });
    expect(screen.getByDisplayValue('新会话单位')).toBeInTheDocument();
    expect(
      screen.queryByText('批次 OLD-BATCH 生成成功'),
    ).not.toBeInTheDocument();
    expect(onCreated).not.toHaveBeenCalled();
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(4));
  });
});
