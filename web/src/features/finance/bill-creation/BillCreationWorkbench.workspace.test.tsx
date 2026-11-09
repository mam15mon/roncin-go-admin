import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it } from 'vitest';
import {
  BillCreationWorkbench,
  mocks,
  resetWorkbenchMocks,
} from './BillCreationWorkbench.fixtures';

describe('建账当前公司工作区', () => {
  beforeEach(() => resetWorkbenchMocks());
  it('身份未就绪不请求，切换工作区后只查询当前公司并隔离迟到候选', async () => {
    let resolveA: ((value: unknown) => void) | undefined;
    mocks.organizationId = undefined;
    mocks.candidates.mockImplementation(({ organizationId }) =>
      organizationId === 'A'
        ? new Promise((resolve) => {
            resolveA = resolve;
          })
        : Promise.resolve({
            data: [
              {
                id: 'fee-B',
                feeName: '费用B',
                organizationId: 'B',
                canBill: true,
              },
            ],
          }),
    );
    const view = renderWithClient(
      <App>
        <BillCreationWorkbench open onClose={vi.fn()} />
      </App>,
    );
    expect(mocks.candidates).not.toHaveBeenCalled();
    expect(screen.queryByLabelText('所属公司')).not.toBeInTheDocument();
    mocks.organizationId = 'A';
    view.rerender(
      <QueryClientProvider client={view.queryClient}>
        <App>
          <BillCreationWorkbench key="A" open onClose={vi.fn()} />
        </App>
      </QueryClientProvider>,
    );
    await waitFor(() =>
      expect(mocks.candidates).toHaveBeenCalledWith(
        expect.objectContaining({ organizationId: 'A' }),
      ),
    );
    mocks.organizationId = 'B';
    view.rerender(
      <QueryClientProvider client={view.queryClient}>
        <App>
          <BillCreationWorkbench key="B" open onClose={vi.fn()} />
        </App>
      </QueryClientProvider>,
    );
    expect(
      await screen.findByRole('button', { name: '费用B' }),
    ).toBeInTheDocument();
    await act(async () =>
      resolveA?.({ data: [{ id: 'fee-A', feeName: '费用A' }] }),
    );
    expect(
      screen.queryByRole('button', { name: '费用A' }),
    ).not.toBeInTheDocument();
    expect(mocks.organizations).not.toHaveBeenCalled();
  });

  it('当前公司从 A 切到 B 后丢弃迟到的 A 预览', async () => {
    let resolveA: ((value: unknown) => void) | undefined;
    mocks.preview.mockImplementation(({ organizationId }) =>
      organizationId === 'A'
        ? new Promise((resolve) => {
            resolveA = resolve;
          })
        : Promise.resolve({
            previewToken: 'token-b',
            data: [
              {
                groupKey: 'group-b',
                settlementPartyName: '结算单位B',
                fees: [],
              },
            ],
          }),
    );
    const view = renderWithClient(
      <App>
        <BillCreationWorkbench
          open
          initialFeeIds={['fee-A']}
          initialOrganizationId="A"
          onClose={vi.fn()}
        />
      </App>,
    );
    await waitFor(() =>
      expect(mocks.preview).toHaveBeenCalledWith(
        expect.objectContaining({ organizationId: 'A' }),
        expect.anything(),
      ),
    );

    mocks.organizationId = 'B';
    view.rerender(
      <QueryClientProvider client={view.queryClient}>
        <App>
          <BillCreationWorkbench
            open
            initialFeeIds={['fee-B']}
            initialOrganizationId="B"
            onClose={vi.fn()}
          />
        </App>
      </QueryClientProvider>,
    );
    expect(await screen.findByDisplayValue('结算单位B')).toBeInTheDocument();

    resolveA?.({
      previewToken: 'token-a',
      data: [
        {
          groupKey: 'group-a',
          settlementPartyName: '结算单位A',
          fees: [],
        },
      ],
    });
    await waitFor(() => {
      expect(screen.queryByDisplayValue('结算单位A')).not.toBeInTheDocument();
      expect(screen.getByDisplayValue('结算单位B')).toBeInTheDocument();
    });
  });

  it.each([undefined, 'B'])(
    '固定带入公司%s不能覆盖当前公司',
    async (source) => {
      renderWithClient(
        <App>
          <BillCreationWorkbench
            open
            initialFeeIds={['foreign-fee']}
            initialOrganizationId={source}
            onClose={() => undefined}
          />
        </App>,
      );
      expect(mocks.preview).not.toHaveBeenCalled();
      fireEvent.click(screen.getByRole('button', { name: '下一步' }));
      expect(mocks.preview).not.toHaveBeenCalled();
      expect(mocks.create).not.toHaveBeenCalled();
    },
  );

  it('非公司或无建账动作权限不请求候选或预览', async () => {
    mocks.canOperate = false;
    renderWithClient(
      <App>
        <BillCreationWorkbench
          open
          initialFeeIds={['fee-A']}
          initialOrganizationId="A"
          onClose={() => undefined}
        />
      </App>,
    );
    fireEvent.click(screen.getByRole('button', { name: '下一步' }));
    expect(mocks.candidates).not.toHaveBeenCalled();
    expect(mocks.preview).not.toHaveBeenCalled();
    expect(mocks.create).not.toHaveBeenCalled();
  });
});
