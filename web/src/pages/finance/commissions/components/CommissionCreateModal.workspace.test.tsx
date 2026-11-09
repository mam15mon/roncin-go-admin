import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, expect, it, vi } from 'vitest';
import {
  CommissionCreateModal,
  modalState,
  resetCommissionModalMocks,
  serviceMocks,
  workspace,
} from './CommissionCreateModal.fixtures';

beforeEach(resetCommissionModalMocks);

it.each(['missing-user', 'missing-company', 'non-company', 'no-manage'])(
  '%s 不加载候选、预览或创建',
  async (scenario) => {
    if (scenario === 'missing-user') workspace.userId = undefined;
    if (scenario === 'missing-company') workspace.organizationId = undefined;
    if (scenario === 'non-company') workspace.canOperate = false;
    if (scenario === 'no-manage') workspace.canManage = false;
    renderWithClient(
      <App>
        <CommissionCreateModal
          open
          onOpenChange={vi.fn()}
          onSuccess={vi.fn()}
        />
      </App>,
    );
    await act(async () => {});
    expect(
      serviceMocks.listCommissionVerificationCandidates,
    ).not.toHaveBeenCalled();
    expect(serviceMocks.listCommissionNettingCandidates).not.toHaveBeenCalled();
    expect(serviceMocks.listCommissionCandidates).not.toHaveBeenCalled();
    expect(
      screen.getByRole('button', { name: '计算并核对预览' }),
    ).toBeDisabled();
    await act(async () => {
      expect(
        await modalState.props?.onFinish(modalState.dependencyValues),
      ).toBe(false);
    });
    expect(serviceMocks.previewCommission).not.toHaveBeenCalled();
    expect(serviceMocks.createCommission).not.toHaveBeenCalled();
  },
);

it.each([undefined, 'org-b'])(
  '核销来源公司为%s时明确拒绝，不产生候选、预览和创建',
  async (organizationId) => {
    serviceMocks.listCommissionVerificationCandidates.mockResolvedValue({
      data: [
        {
          id: 'verification-1',
          organizationId,
          verificationNo: 'INVALID',
          amount: '1',
          currency: 'CNY',
        },
      ],
    });
    renderWithClient(
      <App>
        <CommissionCreateModal
          open
          onOpenChange={vi.fn()}
          onSuccess={vi.fn()}
        />
      </App>,
    );
    await screen.findByText(/INVALID｜/);
    fireEvent.click(screen.getByRole('button', { name: '选择核销单' }));
    expect(
      await screen.findByText('只能选择当前公司工作区的核销单'),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '计算并核对预览' }));
    await act(async () => {
      expect(
        await modalState.props?.onFinish(modalState.dependencyValues),
      ).toBe(false);
    });
    expect(serviceMocks.listCommissionCandidates).not.toHaveBeenCalled();
    expect(serviceMocks.previewCommission).not.toHaveBeenCalled();
    expect(serviceMocks.createCommission).not.toHaveBeenCalled();
  },
);

it('选中核销后搜索不包含它的新候选，仍以已选来源快照预览和创建', async () => {
  serviceMocks.listCommissionVerificationCandidates.mockImplementation(
    ({ keyword }) =>
      Promise.resolve({
        data: keyword
          ? []
          : [
              {
                id: 'verification-1',
                organizationId: 'org-a',
                verificationNo: 'VR-SELECTED',
                amount: '1',
                currency: 'CNY',
              },
            ],
      }),
  );
  serviceMocks.createCommission.mockResolvedValue({ data: { id: 'created' } });
  renderWithClient(
    <App>
      <CommissionCreateModal open onOpenChange={vi.fn()} onSuccess={vi.fn()} />
    </App>,
  );
  await screen.findByText(/VR-SELECTED｜/);
  fireEvent.click(screen.getByRole('button', { name: '选择核销单' }));
  await waitFor(() =>
    expect(serviceMocks.listCommissionCandidates).toHaveBeenCalledOnce(),
  );
  expect(screen.getByTestId('candidate-disabled')).toHaveTextContent('false');
  fireEvent.change(screen.getByLabelText('搜索有效应收核销'), {
    target: { value: '不匹配' },
  });
  await waitFor(() =>
    expect(
      serviceMocks.listCommissionVerificationCandidates,
    ).toHaveBeenLastCalledWith(
      expect.objectContaining({ keyword: '不匹配', organizationId: 'org-a' }),
    ),
  );
  fireEvent.click(screen.getByRole('button', { name: '计算并核对预览' }));
  await screen.findByText('400.00 CNY');
  await act(async () => {
    expect(
      await modalState.props?.onFinish({
        ...modalState.dependencyValues,
        organizationId: 'org-b',
      }),
    ).toBe(true);
  });
  expect(serviceMocks.previewCommission.mock.calls[0][0]).toMatchObject({
    organizationId: 'org-a',
    verificationId: 'verification-1',
  });
  expect(serviceMocks.createCommission.mock.calls[0][0]).toMatchObject({
    organizationId: 'org-a',
    verificationId: 'verification-1',
  });
});

it('关闭重开后迟到预览不能恢复为可提交预览', async () => {
  let resolvePreview: (value: {
    data: { cnyCommissionAmount: string; baseCurrency: string };
  }) => void = () => {};
  serviceMocks.previewCommission.mockImplementation(
    () =>
      new Promise((resolve) => {
        resolvePreview = resolve;
      }),
  );
  const { rerender, queryClient } = renderWithClient(
    <App>
      <CommissionCreateModal open onOpenChange={vi.fn()} onSuccess={vi.fn()} />
    </App>,
  );
  await screen.findByText('VR-A｜A单位｜200.00 CNY');
  fireEvent.click(screen.getByRole('button', { name: '选择核销单' }));
  fireEvent.click(screen.getByRole('button', { name: '计算并核对预览' }));
  await waitFor(() =>
    expect(serviceMocks.previewCommission).toHaveBeenCalledOnce(),
  );
  rerender(
    <QueryClientProvider client={queryClient}>
      <App>
        <CommissionCreateModal
          open={false}
          onOpenChange={vi.fn()}
          onSuccess={vi.fn()}
        />
      </App>
    </QueryClientProvider>,
  );
  rerender(
    <QueryClientProvider client={queryClient}>
      <App>
        <CommissionCreateModal
          open
          onOpenChange={vi.fn()}
          onSuccess={vi.fn()}
        />
      </App>
    </QueryClientProvider>,
  );
  await act(async () => {
    resolvePreview({
      data: { cnyCommissionAmount: '999', baseCurrency: 'CNY' },
    });
  });
  expect(screen.queryByText('999.00 CNY')).not.toBeInTheDocument();
  expect(modalState.props?.submitter.submitButtonProps.disabled).toBe(true);
  await act(async () => {
    expect(await modalState.props?.onFinish(modalState.dependencyValues)).toBe(
      false,
    );
  });
  expect(serviceMocks.createCommission).not.toHaveBeenCalled();
});
