import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it } from 'vitest';
import {
  BillCreationWorkbench,
  mocks,
  pickBillDate,
  resetWorkbenchMocks,
  singlePreviewGroup,
  sleepInAct,
} from './BillCreationWorkbench.fixtures';

describe('BillCreationWorkbench 预览竞态与规模', () => {
  beforeEach(() => {
    resetWorkbenchMocks();
  });

  it('账单日期快速从 D1 改到 D2，迟到的 D1 预览不得覆盖 D2 结果', async () => {
    const pending: Array<(value: unknown) => void> = [];
    mocks.preview.mockImplementation(
      () =>
        new Promise((resolve) => {
          pending.push(resolve);
        }),
    );
    renderWithClient(
      <App>
        <BillCreationWorkbench
          open
          initialFeeIds={['fee-A']}
          initialOrganizationId="A"
          onClose={vi.fn()}
        />
      </App>,
    );
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(1));
    await act(async () => {
      pending.shift()?.({
        previewToken: 'token-init',
        data: [singlePreviewGroup],
      });
    });
    expect(await screen.findByDisplayValue('结算单位甲')).toBeInTheDocument();
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(2));
    await act(async () => {
      pending.shift()?.({
        previewToken: 'token-init',
        data: [singlePreviewGroup],
      });
    });
    await sleepInAct(700);
    const baseline = mocks.preview.mock.calls.length;

    // 请求 1：日期改为 D1，保持挂起。
    pickBillDate('2026-09-05');
    await waitFor(() =>
      expect(mocks.preview).toHaveBeenCalledTimes(baseline + 1),
    );
    // 请求 2：立即改为 D2，同样挂起。
    pickBillDate('2026-09-18');
    await waitFor(() =>
      expect(mocks.preview).toHaveBeenCalledTimes(baseline + 2),
    );
    expect(pending).toHaveLength(2);

    // 请求 2（D2）先完成并生效；resolve 的后续状态更新包进 act。
    await act(async () => {
      pending[1]?.({
        previewToken: 'token-d2',
        data: [
          {
            ...singlePreviewGroup,
            baseCurrencyAmount: '200.00',
            estimatedInvoiceAmount: '200.00',
          },
        ],
      });
    });
    expect(await screen.findByText('200.00 CNY')).toBeInTheDocument();

    // 请求 1（D1）后完成，不得覆盖金额、完成状态或创建令牌。
    await act(async () => {
      pending[0]?.({
        previewToken: 'token-d1',
        data: [
          {
            ...singlePreviewGroup,
            baseCurrencyAmount: '999.00',
            estimatedInvoiceAmount: '999.00',
          },
        ],
      });
    });
    await sleepInAct(300);
    expect(screen.queryByText('999.00 CNY')).not.toBeInTheDocument();
    expect(screen.getByText('200.00 CNY')).toBeInTheDocument();

    const createBtn = await screen.findByRole('button', {
      name: /原子生成 1 张账单/,
    });
    await waitFor(() => expect(createBtn).not.toHaveClass('ant-btn-loading'));
    fireEvent.click(createBtn);
    await waitFor(() => expect(mocks.create).toHaveBeenCalledTimes(1));
    expect(mocks.create).toHaveBeenCalledWith(
      expect.objectContaining({ previewToken: 'token-d2' }),
      expect.anything(),
    );
  });

  it('预计开票币种与汇率的迟到预览同样不覆盖新结果', async () => {
    const pending: Array<(value: unknown) => void> = [];
    mocks.preview.mockImplementation(
      () =>
        new Promise((resolve) => {
          pending.push(resolve);
        }),
    );
    renderWithClient(
      <App>
        <BillCreationWorkbench
          open
          initialFeeIds={['fee-A']}
          initialOrganizationId="A"
          onClose={vi.fn()}
        />
      </App>,
    );
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(1));
    await act(async () => {
      pending.shift()?.({
        previewToken: 'token-init',
        data: [singlePreviewGroup],
      });
    });
    expect(await screen.findByDisplayValue('结算单位甲')).toBeInTheDocument();
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(2));
    await act(async () => {
      pending.shift()?.({
        previewToken: 'token-init',
        data: [singlePreviewGroup],
      });
    });
    await sleepInAct(700);
    const baseline = mocks.preview.mock.calls.length;

    // 请求 1：预计开票币种改为 USD，保持挂起。
    fireEvent.mouseDown(screen.getByLabelText('预计开票币种'));
    fireEvent.click(await screen.findByText('USD - 美元'));
    await waitFor(() =>
      expect(mocks.preview).toHaveBeenCalledTimes(baseline + 1),
    );
    // 请求 2：继续填写预计开票汇率，同样挂起。
    fireEvent.change(
      screen.getByPlaceholderText('服务端按预计口径计算，可调整'),
      { target: { value: '6.5' } },
    );
    await waitFor(() =>
      expect(mocks.preview).toHaveBeenCalledTimes(baseline + 2),
    );
    expect(pending).toHaveLength(2);

    // 请求 2 先完成并生效；resolve 的后续状态更新包进 act。
    await act(async () => {
      pending[1]?.({
        previewToken: 'token-rate',
        data: [
          {
            ...singlePreviewGroup,
            baseCurrencyAmount: '300.00',
            estimatedInvoiceAmount: '300.00',
          },
        ],
      });
    });
    expect(await screen.findByText('300.00 CNY')).toBeInTheDocument();

    // 请求 1 后完成，不得覆盖。
    await act(async () => {
      pending[0]?.({
        previewToken: 'token-currency',
        data: [
          {
            ...singlePreviewGroup,
            baseCurrencyAmount: '111.00',
            estimatedInvoiceAmount: '111.00',
          },
        ],
      });
    });
    await sleepInAct(300);
    expect(screen.queryByText('111.00 CNY')).not.toBeInTheDocument();
    expect(screen.getByText('300.00 CNY')).toBeInTheDocument();

    const createBtn = await screen.findByRole('button', {
      name: /原子生成 1 张账单/,
    });
    await waitFor(() => expect(createBtn).not.toHaveClass('ant-btn-loading'));
    fireEvent.click(createBtn);
    await waitFor(() => expect(mocks.create).toHaveBeenCalledTimes(1));
    expect(mocks.create).toHaveBeenCalledWith(
      expect.objectContaining({ previewToken: 'token-rate' }),
      expect.anything(),
    );
  });
});
