import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  BillCreationWorkbench,
  mocks,
  resetWorkbenchMocks,
  singlePreviewGroup,
} from './BillCreationWorkbench.fixtures';

const ar = { ...singlePreviewGroup, groupKey: 'ar', groupToken: 'token-ar' };
const ap = {
  ...singlePreviewGroup,
  groupKey: 'ap',
  groupToken: 'token-ap',
  direction: 'PAYABLE',
  settlementPartyName: '应付单位',
  fees: [{ id: 'fee-B', feeName: '应付费用' }],
};
const billA = { id: 'bill-ar', billNo: 'B-AR' };
const billB = { id: 'bill-ap', billNo: 'B-AP' };
const partial = {
  data: { id: 'batch-1', batchNo: 'BATCH-1', billCount: 1, bills: [billA] },
  results: [
    { groupKey: 'ar', success: true, bill: billA },
    // ProtoJSON 不输出 false，也不输出失败组的 bill。
    {
      groupKey: 'ap',
      error: { reason: 'FINANCE_BILL_PREVIEW_STALE', message: '费用已变化' },
    },
  ],
};

async function renderReady(onCreated = vi.fn()) {
  const view = renderWithClient(
    <App>
      <BillCreationWorkbench
        open
        initialFeeIds={['fee-A', 'fee-B']}
        initialOrganizationId="A"
        onClose={vi.fn()}
        onCreated={onCreated}
      />
    </App>,
  );
  await screen.findByDisplayValue('结算单位甲');
  await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(2));
  const button = screen.getByRole('button', { name: /生成 2 张账单/ });
  await waitFor(() => expect(button).not.toHaveClass('ant-btn-loading'));
  fireEvent.click(button);
  return view;
}

describe('批量建账部分结果与原请求重试', () => {
  beforeEach(() => {
    resetWorkbenchMocks();
    mocks.preview.mockResolvedValue({ data: [ar, ap] });
  });

  it('逐组显示部分成功，失败组重试保持整次原请求与幂等键', async () => {
    const onCreated = vi.fn();
    mocks.create.mockResolvedValueOnce(partial).mockResolvedValueOnce({
      data: { ...partial.data, billCount: 2, bills: [billA, billB] },
      results: [
        { groupKey: 'ar', success: true, bill: billA },
        { groupKey: 'ap', success: true, bill: billB },
      ],
    });
    await renderReady(onCreated);
    expect(
      await screen.findByText('批次 BATCH-1：成功 1 组，失败 1 组'),
    ).toBeInTheDocument();
    expect(
      screen.getByText('费用或创建日期已发生变化，请返回调整失败组后重新预览'),
    ).toBeInTheDocument();
    expect(onCreated).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByRole('button', { name: '重试失败组' }));
    await waitFor(() => expect(mocks.create).toHaveBeenCalledTimes(2));
    expect(mocks.create.mock.calls[1][0]).toEqual(
      mocks.create.mock.calls[0][0],
    );
    expect(
      await screen.findByText('批次 BATCH-1 生成成功'),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: '重试失败组' }),
    ).not.toBeInTheDocument();
  });

  it('返回调整仅重取失败组费用，保留其输入并为新意图换幂等键', async () => {
    mocks.create.mockResolvedValueOnce(partial);
    await renderReady();
    await screen.findByText('批次 BATCH-1：成功 1 组，失败 1 组');
    const original = mocks.create.mock.calls[0][0];
    mocks.preview.mockResolvedValue({ data: [ap] });
    fireEvent.click(screen.getByRole('button', { name: '返回调整失败组' }));
    await waitFor(() =>
      expect(mocks.preview).toHaveBeenLastCalledWith(
        expect.objectContaining({ feeIds: ['fee-B'] }),
        expect.anything(),
      ),
    );
    expect(await screen.findByDisplayValue('应付单位')).toBeInTheDocument();
    const button = screen.getByRole('button', { name: /生成 1 张账单/ });
    await waitFor(() => expect(button).not.toHaveClass('ant-btn-loading'));
    fireEvent.click(button);
    await waitFor(() => expect(mocks.create).toHaveBeenCalledTimes(2));
    const adjusted = mocks.create.mock.calls[1][0];
    expect(adjusted.idempotencyKey).not.toBe(original.idempotencyKey);
    expect(adjusted.groups).toEqual([
      expect.objectContaining({
        groupKey: 'ap',
        feeIds: ['fee-B'],
        statementTitle: '应付单位',
      }),
    ]);
    await act(async () => {});
  });

  it('响应未知时冻结配置，确认原请求后才显示成功', async () => {
    mocks.create.mockResolvedValueOnce(undefined).mockResolvedValueOnce({
      data: { id: 'batch-1', batchNo: 'BATCH-1', bills: [billA, billB] },
      results: [
        { groupKey: 'ar', success: true, bill: billA },
        { groupKey: 'ap', success: true, bill: billB },
      ],
    });
    await renderReady();
    expect(
      await screen.findByText('上次生成结果还未确认，请先确认结果再继续。'),
    ).toBeInTheDocument();
    expect(screen.getByDisplayValue('结算单位甲')).toBeDisabled();
    expect(screen.getByRole('button', { name: /刷新账单/ })).toBeDisabled();
    expect(screen.getByRole('button', { name: /上一步/ })).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: /确认原请求结果/ }));
    await waitFor(() => expect(mocks.create).toHaveBeenCalledTimes(2));
    expect(mocks.create.mock.calls[1][0]).toEqual(
      mocks.create.mock.calls[0][0],
    );
    expect(
      await screen.findByText('批次 BATCH-1 生成成功'),
    ).toBeInTheDocument();
  });
});
