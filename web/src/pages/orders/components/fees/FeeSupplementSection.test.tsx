import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { orderErrorReasons } from '@/errorReasons.generated';
import {
  orderFeeServiceApproveOrderFeeSupplement,
  orderFeeServiceCancelApprovedOrderFeeSupplement,
  orderFeeServiceCreateOrderFeeSupplement,
  orderFeeServiceListOrderFeeSupplementRequests,
  orderFeeServiceRejectOrderFeeSupplement,
  orderFeeServiceWithdrawOrderFeeSupplement,
} from '@/services/roncin/orderFeeService';
import FeeSupplementSection from './FeeSupplementSection';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceListOrderFeeSupplementRequests: vi.fn(),
  orderFeeServiceCreateOrderFeeSupplement: vi.fn(),
  orderFeeServiceApproveOrderFeeSupplement: vi.fn(),
  orderFeeServiceRejectOrderFeeSupplement: vi.fn(),
  orderFeeServiceWithdrawOrderFeeSupplement: vi.fn(),
  orderFeeServiceCancelApprovedOrderFeeSupplement: vi.fn(),
}));

// 弹窗以桩件替换，专注测试提交后服务调用与反馈；表单校验单独用真实弹窗覆盖。
vi.mock('./FeeSupplementModal', () => ({
  default: ({ open, onSubmit, initialRequest }: any) => (
    <div data-testid="supplement-modal" data-prefill-id={initialRequest?.id}>
      {String(open)}
      <button
        type="button"
        onClick={() =>
          onSubmit({
            feeSettingId: 'fs-1',
            settlementPartyId: 'sp-1',
            billingUnitId: 'bu-1',
            quantity: '2',
            unitPrice: '50',
            currency: 'CNY',
            expenseDate: '2026-09-01',
            reason: '漏录拖车费',
          })
        }
      >
        提交补录
      </button>
    </div>
  ),
}));

const listSupplements = vi.mocked(
  orderFeeServiceListOrderFeeSupplementRequests,
);
const createSupplement = vi.mocked(orderFeeServiceCreateOrderFeeSupplement);
const approveSupplement = vi.mocked(orderFeeServiceApproveOrderFeeSupplement);
const rejectSupplement = vi.mocked(orderFeeServiceRejectOrderFeeSupplement);
const withdrawSupplement = vi.mocked(orderFeeServiceWithdrawOrderFeeSupplement);
const cancelApproved = vi.mocked(
  orderFeeServiceCancelApprovedOrderFeeSupplement,
);

function supplementRow(
  overrides: Partial<API.OrderFeeSupplementRequestData> = {},
): API.OrderFeeSupplementRequestData {
  return {
    id: 'sup-1',
    orderId: 'order-1',
    lockBasis: 'BOTH',
    status: 'PENDING',
    version: '3',
    direction: 2,
    feeCode: 'OCEAN',
    feeName: '海运费',
    quantity: '2',
    unitPrice: '50',
    totalAmount: '100',
    currency: 'CNY',
    expenseDate: '2026-09-01',
    reason: '漏录拖车费',
    requestedBy: 'user-a',
    requestedByName: '张发起',
    requestedAt: '2026-09-01T10:00:00Z',
    canApprove: false,
    canWithdraw: false,
    canCancel: false,
    approverAvailable: true,
    ...overrides,
  };
}

function listResponse(items: API.OrderFeeSupplementRequestData[]) {
  return {
    success: true,
    data: { items, total: items.length, page: 1, pageSize: 10 },
  } as Awaited<ReturnType<typeof listSupplements>>;
}

function conflictError(reason: string, message: string) {
  const error = new Error(message) as Error & {
    data?: { reason: string; message: string };
  };
  error.data = { reason, message };
  return error;
}

function makeProps(orderId: string) {
  return {
    orderId,
    canCreate: true,
    lockActive: true,
    feeSettings: [],
    settlementParties: [],
    currencies: [],
    billingUnits: [],
    onFeeTablesReload: vi.fn(),
  };
}

describe('FeeSupplementSection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    listSupplements.mockResolvedValue(listResponse([]));
  });

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
    const props = { ...makeProps('order-1'), lockActive: false };
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
      await screen.findByText(/当前没有任何具备直接解锁资格的审批人/),
    ).toBeInTheDocument();
  });

  it('发起人可撤回本人 PENDING 申请，携带 expectedVersion', async () => {
    withdrawSupplement.mockResolvedValue({
      success: true,
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
    withdrawSupplement.mockResolvedValue({ success: true } as Awaited<
      ReturnType<typeof withdrawSupplement>
    >);
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
        <FeeSupplementSection {...makeProps('order-1')} lockActive={false} />
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
    resolveWithdraw({ success: true });
    await waitFor(() =>
      expect(screen.getByTestId('supplement-modal')).toHaveTextContent('false'),
    );
    expect(screen.getByTestId('supplement-modal')).not.toHaveAttribute(
      'data-prefill-id',
      'sup-1',
    );
    expect(createSupplement).not.toHaveBeenCalled();
  });

  it('具备审批能力时展示通过/驳回；通过后生成费用并刷新费用表', async () => {
    approveSupplement.mockResolvedValue({
      success: true,
    } as Awaited<ReturnType<typeof approveSupplement>>);
    listSupplements.mockResolvedValue(
      listResponse([
        supplementRow({ canApprove: true, requestedBy: 'user-other' }),
      ]),
    );
    const onFeeTablesReload = vi.fn();
    const props = { ...makeProps('order-1'), onFeeTablesReload };
    render(
      <App>
        <FeeSupplementSection {...props} />
      </App>,
    );

    expect(await screen.findByText('通过')).toBeInTheDocument();
    expect(screen.getByText('驳回')).toBeInTheDocument();
    expect(screen.getByText('张发起')).toBeInTheDocument();
    expect(screen.queryByText('user-other')).not.toBeInTheDocument();

    fireEvent.click(screen.getByText('通过'));
    fireEvent.click(await screen.findByRole('button', { name: /通\s*过/ }));

    await waitFor(() => expect(approveSupplement).toHaveBeenCalledTimes(1));
    const [, body] = approveSupplement.mock.calls[0];
    expect(body.expectedVersion).toBe('3');
    await waitFor(() => expect(onFeeTablesReload).toHaveBeenCalled());
    expect(
      await screen.findByText('补录申请已通过，费用已生成'),
    ).toBeInTheDocument();
  });

  it('锁依据全部失效时按服务端 next_action 引导提示', async () => {
    approveSupplement.mockRejectedValue(
      conflictError(
        orderErrorReasons.LOCK_BASIS_CHANGED,
        '申请提交时的锁依据已全部失效，订单当前已无任何锁，请改走普通费用新增入口',
      ),
    );
    listSupplements.mockResolvedValue(
      listResponse([supplementRow({ canApprove: true })]),
    );
    const props = makeProps('order-1');
    render(
      <App>
        <FeeSupplementSection {...props} />
      </App>,
    );

    fireEvent.click(await screen.findByText('通过'));
    fireEvent.click(await screen.findByRole('button', { name: /通\s*过/ }));

    expect(
      await screen.findByText(/请改走普通费用新增入口/),
    ).toBeInTheDocument();
  });

  it('驳回原因为必填并随请求提交', async () => {
    rejectSupplement.mockResolvedValue({
      success: true,
    } as Awaited<ReturnType<typeof rejectSupplement>>);
    listSupplements.mockResolvedValue(
      listResponse([supplementRow({ canApprove: true })]),
    );
    const props = makeProps('order-1');
    render(
      <App>
        <FeeSupplementSection {...props} />
      </App>,
    );

    fireEvent.click(await screen.findByText('驳回'));
    const reasonInput = await screen.findByPlaceholderText(
      '请输入驳回原因（必填）',
    );
    // 不填原因直接确认会被拦截。
    fireEvent.click(screen.getByRole('button', { name: /确\s*认/ }));
    expect(await screen.findByText('请输入驳回原因')).toBeInTheDocument();
    expect(rejectSupplement).not.toHaveBeenCalled();

    fireEvent.change(reasonInput, { target: { value: '金额与实际不符' } });
    fireEvent.click(screen.getByRole('button', { name: /确\s*认/ }));

    await waitFor(() => expect(rejectSupplement).toHaveBeenCalledTimes(1));
    const [, body] = rejectSupplement.mock.calls[0];
    expect(body.reason).toBe('金额与实际不符');
    expect(body.expectedVersion).toBe('3');
  });

  it('APPROVED 申请按能力投影展示作废入口与稳定阻断原因', async () => {
    listSupplements.mockResolvedValue(
      listResponse([
        supplementRow({
          id: 'sup-cancelable',
          status: 'APPROVED',
          decidedBy: 'user-approver',
          decidedByName: '李审批',
          feeId: 'fee-1',
          feeStatus: 'CONFIRMED',
          canCancel: true,
        }),
        supplementRow({
          id: 'sup-blocked',
          status: 'APPROVED',
          feeId: 'fee-2',
          feeStatus: 'BILLED',
          canCancel: false,
          cancelBlockedReason: '补录费用已建账，需先取消账单后再作废',
        }),
      ]),
    );
    const props = makeProps('order-1');
    render(
      <App>
        <FeeSupplementSection {...props} />
      </App>,
    );

    expect(await screen.findByText('作废补录费用')).toBeInTheDocument();
    expect(screen.getByText('不可作废')).toBeInTheDocument();
    expect(screen.getByText('已进账单')).toBeInTheDocument();
    expect(screen.getByText('李审批')).toBeInTheDocument();
    expect(screen.queryByText('user-approver')).not.toBeInTheDocument();
    // 阻断原因只展示服务端文案，不引导普通删除。
    expect(screen.queryByText(/^普通删除/)).not.toBeInTheDocument();
    expect(cancelApproved).not.toHaveBeenCalled();
  });

  it('作废补录费用必须填写原因并携带费用版本', async () => {
    cancelApproved.mockResolvedValue({
      success: true,
    } as Awaited<ReturnType<typeof cancelApproved>>);
    listSupplements.mockResolvedValue(
      listResponse([
        supplementRow({
          status: 'APPROVED',
          feeId: 'fee-1',
          feeStatus: 'CONFIRMED',
          canCancel: true,
        }),
      ]),
    );
    const props = makeProps('order-1');
    render(
      <App>
        <FeeSupplementSection {...props} />
      </App>,
    );

    fireEvent.click(await screen.findByText('作废补录费用'));
    // 确认弹窗明确提示已建账与冲减已确认的前置条件。
    expect(
      await screen.findByText(/已建账需先按现有链路取消账单/),
    ).toBeInTheDocument();
    const reasonInput = await screen.findByPlaceholderText(
      '请输入作废原因（必填）',
    );
    fireEvent.click(screen.getByRole('button', { name: /确\s*认\s*作\s*废/ }));
    expect(await screen.findByText('请输入作废原因')).toBeInTheDocument();
    expect(cancelApproved).not.toHaveBeenCalled();

    fireEvent.change(reasonInput, { target: { value: '重复补录' } });
    fireEvent.click(screen.getByRole('button', { name: /确\s*认\s*作\s*废/ }));

    await waitFor(() => expect(cancelApproved).toHaveBeenCalledTimes(1));
    const [params, body] = cancelApproved.mock.calls[0];
    expect(params).toEqual({ orderId: 'order-1', id: 'sup-1' });
    expect(body.expectedVersion).toBe('3');
    expect(body.reason).toBe('重复补录');
  });

  it('提交后审批人全部失效的 PENDING 申请展示暂无审批人提示', async () => {
    listSupplements.mockResolvedValue(
      listResponse([supplementRow({ approverAvailable: false })]),
    );
    const props = makeProps('order-1');
    render(
      <App>
        <FeeSupplementSection {...props} />
      </App>,
    );

    expect(await screen.findByText('暂无可用审批人')).toBeInTheDocument();
  });

  it('订单 A 的申请列表迟到响应不得覆盖订单 B', async () => {
    let resolveA!: (value: unknown) => void;
    const deferredA = new Promise<any>((resolve) => {
      resolveA = resolve;
    });
    listSupplements.mockImplementation(({ orderId }) =>
      orderId === 'order-A' ? deferredA : Promise.resolve(listResponse([])),
    );
    const { rerender } = render(
      <App>
        <FeeSupplementSection {...makeProps('order-A')} />
      </App>,
    );
    await waitFor(() =>
      expect(listSupplements).toHaveBeenCalledWith({
        orderId: 'order-A',
        page: 1,
        pageSize: 10,
      }),
    );

    rerender(
      <App>
        <FeeSupplementSection {...makeProps('order-B')} />
      </App>,
    );
    await waitFor(() =>
      expect(listSupplements).toHaveBeenCalledWith({
        orderId: 'order-B',
        page: 1,
        pageSize: 10,
      }),
    );

    resolveA(listResponse([supplementRow({ feeName: 'A 订单独有费用' })]));
    await waitFor(() =>
      expect(screen.queryByText('A 订单独有费用')).not.toBeInTheDocument(),
    );
  });
});
