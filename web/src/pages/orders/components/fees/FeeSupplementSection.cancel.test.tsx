import './feeSupplementTestSupport';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { describe, expect, it } from 'vitest';
import FeeSupplementSection from './FeeSupplementSection';
import {
  cancelApproved,
  conflictError,
  listResponse,
  listSupplements,
  makeProps,
  render,
  supplementRow,
} from './feeSupplementTestSupport';

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: { currentUser: { currentOrganization: { id: 'org-1' } } },
  }),
}));
describe('费用补录撤销与状态', () => {
  it('APPROVED 申请按能力投影展示撤销入口、稳定阻断原因与生成费用删除结果', async () => {
    listSupplements.mockResolvedValue(
      listResponse([
        supplementRow({
          id: 'sup-cancelable',
          status: 'APPROVED',
          decidedBy: 'user-approver',
          decidedByName: '李审批',
          feeId: 'fee-1',
          canCancel: true,
        }),
        supplementRow({
          id: 'sup-blocked',
          status: 'APPROVED',
          feeId: 'fee-2',
          canCancel: false,
          cancelBlockedReason:
            '补录生成的费用已建账，需先经现有财务链路取消账单',
        }),
        supplementRow({
          id: 'sup-deleted',
          status: 'APPROVED',
          canCancel: false,
          cancelBlockedReason: '生成费用已删除',
        }),
      ]),
    );
    const props = makeProps('order-1');
    render(
      <App>
        <FeeSupplementSection {...props} />
      </App>,
    );

    expect(await screen.findByText('撤销补录费用')).toBeInTheDocument();
    // 两行不可撤销：已建账阻断与生成费用已删除。
    expect(screen.getAllByText('不可撤销')).toHaveLength(2);
    // 生成结果列：有 fee_id 展示已生成；fee_id 缺失明确展示已删除，不显示空白。
    expect(screen.getAllByText('已生成').length).toBeGreaterThan(0);
    expect(screen.getAllByText('生成费用已删除').length).toBeGreaterThan(0);
    expect(screen.getByText('李审批')).toBeInTheDocument();
    expect(screen.queryByText('user-approver')).not.toBeInTheDocument();
    // 阻断原因只展示服务端文案，不引导普通删除。
    expect(screen.queryByText(/^普通删除/)).not.toBeInTheDocument();
    expect(cancelApproved).not.toHaveBeenCalled();
  });
  it('撤销补录费用必须填写原因并携带费用版本', async () => {
    cancelApproved.mockResolvedValue({
      success: true,
      data: supplementRow(),
    } as Awaited<ReturnType<typeof cancelApproved>>);
    listSupplements.mockResolvedValue(
      listResponse([
        supplementRow({
          status: 'APPROVED',
          feeId: 'fee-1',
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

    fireEvent.click(await screen.findByText('撤销补录费用'));
    // 确认弹窗明确提示已建账与冲减已确认的前置条件。
    expect(
      await screen.findByText(/已建账需先按现有链路取消账单/),
    ).toBeInTheDocument();
    const reasonInput = await screen.findByPlaceholderText(
      '请输入撤销原因（必填）',
    );
    fireEvent.click(screen.getByRole('button', { name: /确\s*认\s*撤\s*销/ }));
    expect(await screen.findByText('请输入撤销原因')).toBeInTheDocument();
    expect(cancelApproved).not.toHaveBeenCalled();

    fireEvent.change(reasonInput, { target: { value: '重复补录' } });
    fireEvent.click(screen.getByRole('button', { name: /确\s*认\s*撤\s*销/ }));

    await waitFor(() => expect(cancelApproved).toHaveBeenCalledTimes(1));
    const [params, body] = cancelApproved.mock.calls[0];
    expect(params).toEqual({ orderId: 'order-1', id: 'sup-1' });
    expect(body.expectedVersion).toBe('3');
    expect(body.reason).toBe('重复补录');
  });
  it('重复撤销返回服务端明确拒绝提示且不重复处理', async () => {
    cancelApproved.mockRejectedValue(
      conflictError(
        'FEE_SUPPLEMENT_CANCEL_BLOCKED',
        '补录生成的费用已删除，无需重复撤销',
      ),
    );
    listSupplements.mockResolvedValue(
      listResponse([
        supplementRow({
          status: 'APPROVED',
          feeId: 'fee-1',
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

    fireEvent.click(await screen.findByText('撤销补录费用'));
    const reasonInput = await screen.findByPlaceholderText(
      '请输入撤销原因（必填）',
    );
    fireEvent.change(reasonInput, { target: { value: '误操作撤销' } });
    fireEvent.click(screen.getByRole('button', { name: /确\s*认\s*撤\s*销/ }));

    expect(await screen.findByText(/已删除，无需重复撤销/)).toBeInTheDocument();
    expect(cancelApproved).toHaveBeenCalledTimes(1);
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
