import './feeSupplementTestSupport';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { describe, expect, it } from 'vitest';
import { orderErrorReasons } from '@/errorReasons.generated';
import FeeSupplementSection from './FeeSupplementSection';
import {
  conflictError,
  createSupplement,
  listResponse,
  listSupplements,
  makeProps,
  render,
  supplementRow,
  withdrawSupplement,
} from './feeSupplementTestSupport';

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: { currentUser: { currentOrganization: { id: 'org-1' } } },
  }),
}));
describe('费用补录申请与撤回', () => {
  it('锁定且具备 fee.create 时显示补录入口并固定应付方向发起申请', async () => {
    createSupplement.mockResolvedValue({
      success: true,
      data: supplementRow({ status: 'PENDING', approverAvailable: true }),
    } as Awaited<ReturnType<typeof createSupplement>>);
    const props = makeProps('order-1');

    render(
      <App>
        <FeeSupplementSection {...props} />
      </App>,
    );
    await waitFor(() => expect(listSupplements).toHaveBeenCalled());

    fireEvent.click(screen.getByRole('button', { name: '补录费用' }));
    expect(screen.getByTestId('supplement-modal')).toHaveTextContent('true');
    fireEvent.click(screen.getByRole('button', { name: '提交补录' }));

    await waitFor(() => expect(createSupplement).toHaveBeenCalledTimes(1));
    const [params, body] = createSupplement.mock.calls[0];
    expect(params.orderId).toBe('order-1');
    expect(body.direction).toBe(2);
    expect(body.reason).toBe('漏录拖车费');
    expect(body.expenseDate).toBe('2026-09-01');
    expect(body.idempotencyKey).toBeTruthy();
    await waitFor(() => expect(listSupplements).toHaveBeenCalledTimes(2));
  });
  it('无锁订单不显示补录入口，但申请列表仍可读取（不以 fee.read 隐藏申请区）', async () => {
    const props = { ...makeProps('order-1'), feeReviewApproved: false };
    render(
      <App>
        <FeeSupplementSection {...props} />
      </App>,
    );

    await waitFor(() => expect(listSupplements).toHaveBeenCalled());
    expect(
      screen.queryByRole('button', { name: '补录费用' }),
    ).not.toBeInTheDocument();
  });
  it('不具备 fee.create 时不显示补录入口', async () => {
    const props = { ...makeProps('order-1'), canCreate: false };
    render(
      <App>
        <FeeSupplementSection {...props} />
      </App>,
    );

    await waitFor(() => expect(listSupplements).toHaveBeenCalled());
    expect(
      screen.queryByRole('button', { name: '补录费用' }),
    ).not.toBeInTheDocument();
  });
  it('提交时无合格审批人被稳定拒绝并展示配置引导', async () => {
    createSupplement.mockRejectedValue(
      conflictError(
        orderErrorReasons.FEE_SUPPLEMENT_APPROVER_UNAVAILABLE,
        '当前没有可用审批人',
      ),
    );
    const props = makeProps('order-1');
    render(
      <App>
        <FeeSupplementSection {...props} />
      </App>,
    );
    await waitFor(() => expect(listSupplements).toHaveBeenCalled());

    fireEvent.click(screen.getByRole('button', { name: '补录费用' }));
    fireEvent.click(screen.getByRole('button', { name: '提交补录' }));

    await waitFor(() => expect(createSupplement).toHaveBeenCalled());
    expect(
      await screen.findByText(/当前没有任何具备费用审核权限的审批人/),
    ).toBeInTheDocument();
  });
  it('发起人可撤回本人 PENDING 申请，携带 expectedVersion', async () => {
    withdrawSupplement.mockResolvedValue({
      success: true,
      data: supplementRow(),
    } as Awaited<ReturnType<typeof withdrawSupplement>>);
    listSupplements.mockResolvedValue(
      listResponse([supplementRow({ canWithdraw: true })]),
    );
    const props = makeProps('order-1');
    render(
      <App>
        <FeeSupplementSection {...props} />
      </App>,
    );

    fireEvent.click(await screen.findByText('撤回申请'));
    fireEvent.click(await screen.findByRole('button', { name: /撤\s*回/ }));

    await waitFor(() => expect(withdrawSupplement).toHaveBeenCalledTimes(1));
    const [params, body] = withdrawSupplement.mock.calls[0];
    expect(params).toEqual({ orderId: 'order-1', id: 'sup-1' });
    expect(body.expectedVersion).toBe('3');
    expect(await screen.findByText('补录申请已撤回')).toBeInTheDocument();
  });
  it('撤回与审批竞争失败时展示后端冲突提示并刷新', async () => {
    withdrawSupplement.mockRejectedValue(
      conflictError(orderErrorReasons.FEE_SUPPLEMENT_TRANSITION, '状态冲突'),
    );
    listSupplements.mockResolvedValue(
      listResponse([supplementRow({ canWithdraw: true })]),
    );
    const props = makeProps('order-1');
    render(
      <App>
        <FeeSupplementSection {...props} />
      </App>,
    );

    fireEvent.click(await screen.findByText('撤回申请'));
    fireEvent.click(await screen.findByRole('button', { name: /撤\s*回/ }));

    expect(
      (await screen.findAllByText('申请状态已变化')).length,
    ).toBeGreaterThan(0);
    await waitFor(() => expect(listSupplements).toHaveBeenCalledTimes(2));
  });
  it('仅本人可撤回且当前可创建、订单有锁时显示重提入口；成功后预填原申请', async () => {
    withdrawSupplement.mockResolvedValue({
      success: true,
      data: supplementRow(),
    } as Awaited<ReturnType<typeof withdrawSupplement>>);
    listSupplements.mockResolvedValue(
      listResponse([
        supplementRow({
          canWithdraw: true,
          feeSettingId: 'fs-1',
          settlementPartyId: 'sp-1',
        }),
      ]),
    );
    render(
      <App>
        <FeeSupplementSection {...makeProps('order-1')} />
      </App>,
    );

    fireEvent.click(await screen.findByText('撤回并重提'));
    expect(
      screen.getByText(/原申请撤回后将成为不可修改的历史记录/),
    ).toBeInTheDocument();
    expect(screen.getByTestId('supplement-modal')).toHaveTextContent('false');
    fireEvent.click(screen.getByRole('button', { name: /撤s*回s*并s*重s*提/ }));

    await waitFor(() => expect(withdrawSupplement).toHaveBeenCalledTimes(1));
    expect(withdrawSupplement.mock.calls[0][1].expectedVersion).toBe('3');
    await waitFor(() =>
      expect(screen.getByTestId('supplement-modal')).toHaveTextContent('true'),
    );
    expect(screen.getByTestId('supplement-modal')).toHaveAttribute(
      'data-prefill-id',
      'sup-1',
    );
    expect(createSupplement).not.toHaveBeenCalled();
  });
  it('撤回重提失败时不打开表单；无创建资格或锁时不展示重提', async () => {
    withdrawSupplement.mockRejectedValue(
      conflictError(orderErrorReasons.FEE_SUPPLEMENT_TRANSITION, '状态冲突'),
    );
    listSupplements.mockResolvedValue(
      listResponse([supplementRow({ canWithdraw: true })]),
    );
    const { unmount } = render(
      <App>
        <FeeSupplementSection {...makeProps('order-1')} />
      </App>,
    );

    fireEvent.click(await screen.findByText('撤回并重提'));
    fireEvent.click(screen.getByRole('button', { name: /撤s*回s*并s*重s*提/ }));
    await waitFor(() => expect(withdrawSupplement).toHaveBeenCalledTimes(1));
    expect(screen.getByTestId('supplement-modal')).toHaveTextContent('false');
    unmount();
    const withoutCreate = render(
      <App>
        <FeeSupplementSection {...makeProps('order-1')} canCreate={false} />
      </App>,
    );
    await waitFor(() => expect(listSupplements).toHaveBeenCalled());
    expect(screen.queryByText('撤回并重提')).not.toBeInTheDocument();
    withoutCreate.unmount();
    render(
      <App>
        <FeeSupplementSection
          {...makeProps('order-1')}
          feeReviewApproved={false}
        />
      </App>,
    );
    await screen.findByText('撤回申请');
    expect(screen.queryByText('撤回并重提')).not.toBeInTheDocument();
    expect(screen.getByText('撤回申请')).toBeInTheDocument();
  });
  it('订单切换后旧订单撤回的迟到成功不得打开新订单表单', async () => {
    let resolveWithdraw!: (
      value: API.WithdrawOrderFeeSupplementResponse,
    ) => void;
    withdrawSupplement.mockReturnValue(
      new Promise((resolve) => {
        resolveWithdraw = resolve;
      }) as ReturnType<typeof withdrawSupplement>,
    );
    listSupplements.mockResolvedValue(
      listResponse([supplementRow({ canWithdraw: true })]),
    );
    const { rerender } = render(
      <App>
        <FeeSupplementSection {...makeProps('order-1')} />
      </App>,
    );
    fireEvent.click(await screen.findByText('撤回并重提'));
    fireEvent.click(screen.getByRole('button', { name: /撤s*回s*并s*重s*提/ }));
    await waitFor(() => expect(withdrawSupplement).toHaveBeenCalled());

    rerender(
      <App>
        <FeeSupplementSection {...makeProps('order-2')} />
      </App>,
    );
    resolveWithdraw({ success: true, data: supplementRow() });
    await waitFor(() =>
      expect(screen.getByTestId('supplement-modal')).toHaveTextContent('false'),
    );
    expect(screen.getByTestId('supplement-modal')).not.toHaveAttribute(
      'data-prefill-id',
      'sup-1',
    );
    expect(createSupplement).not.toHaveBeenCalled();
  });
});
