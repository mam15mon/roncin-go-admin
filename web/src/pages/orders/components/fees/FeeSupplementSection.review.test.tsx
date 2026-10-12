import './feeSupplementTestSupport';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { describe, expect, it, vi } from 'vitest';
import { orderErrorReasons } from '@/errorReasons.generated';
import FeeSupplementSection from './FeeSupplementSection';
import {
  approveSupplement,
  conflictError,
  listResponse,
  listSupplements,
  makeProps,
  previewSupplement,
  rejectSupplement,
  render,
  supplementRow,
} from './feeSupplementTestSupport';

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: { currentUser: { currentOrganization: { id: 'org-1' } } },
  }),
}));
describe('费用补录审核', () => {
  it('具备审批能力时先审核毛利，再通过并刷新费用表', async () => {
    approveSupplement.mockResolvedValue({
      success: true,
      data: supplementRow(),
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

    expect(await screen.findByText('审核')).toBeInTheDocument();
    expect(screen.getByText('张发起')).toBeInTheDocument();
    expect(screen.queryByText('user-other')).not.toBeInTheDocument();

    fireEvent.click(screen.getByText('审核'));
    expect(await screen.findByText('40% → 30%')).toBeInTheDocument();
    // expectedVersion 必须与 orderId/id 同处 params 槽位（生成客户端第二参是
    // 请求配置对象，放进去会被静默丢弃，服务端按参数无效拒绝）。
    expect(previewSupplement).toHaveBeenCalledWith({
      orderId: 'order-1',
      id: 'sup-1',
      expectedVersion: '3',
    });
    fireEvent.click(
      await screen.findByRole('button', { name: '批准并确认提成影响' }),
    );

    await waitFor(() => expect(approveSupplement).toHaveBeenCalledTimes(1));
    const [, body] = approveSupplement.mock.calls[0];
    expect(body.expectedVersion).toBe('3');
    await waitFor(() => expect(onFeeTablesReload).toHaveBeenCalled());
    expect(
      await screen.findByText('补录已批准，费用与应有的提成冲减已同步保存'),
    ).toBeInTheDocument();
  });
  it('预览失败时禁止通过，但仍可按原流程驳回', async () => {
    previewSupplement.mockRejectedValue(new Error('汇率不可用'));
    rejectSupplement.mockResolvedValue({
      success: true,
      data: supplementRow(),
    } as Awaited<ReturnType<typeof rejectSupplement>>);
    listSupplements.mockResolvedValue(
      listResponse([supplementRow({ canApprove: true })]),
    );
    render(
      <App>
        <FeeSupplementSection {...makeProps('order-1')} />
      </App>,
    );
    fireEvent.click(await screen.findByText('审核'));
    expect(await screen.findByText('无法预览毛利变化')).toBeInTheDocument();
    expect(
      screen.getByText('批准并确认提成影响').closest('button'),
    ).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: /退\s*回/ }));
    const reasonInput = await screen.findByPlaceholderText(
      '请输入驳回原因（必填）',
    );
    fireEvent.change(reasonInput, { target: { value: '费用依据不足' } });
    fireEvent.click(screen.getByRole('button', { name: /^确\s*认$/ }));
    await waitFor(() => expect(rejectSupplement).toHaveBeenCalledTimes(1));
    expect(approveSupplement).not.toHaveBeenCalled();
  });
  it('零应收时显示毛利率不可计算', async () => {
    previewSupplement.mockResolvedValue({
      data: {
        baseCurrency: 'CNY',
        previewToken: 'supplement-preview',
        currentReceivable: '0.00000000',
        currentPayable: '20.00000000',
        currentProfit: '-20.00000000',
        supplementCost: '10.00000000',
        projectedReceivable: '0.00000000',
        projectedPayable: '30.00000000',
        projectedProfit: '-30.00000000',
        profitChange: '-10.00000000',
      },
    } as Awaited<ReturnType<typeof previewSupplement>>);
    listSupplements.mockResolvedValue(
      listResponse([supplementRow({ canApprove: true })]),
    );
    render(
      <App>
        <FeeSupplementSection {...makeProps('order-1')} />
      </App>,
    );
    fireEvent.click(await screen.findByText('审核'));
    expect(await screen.findByText('不可计算 → 不可计算')).toBeInTheDocument();
  });
  it('审核依据变化时提示刷新并阻止再次批准', async () => {
    approveSupplement.mockRejectedValue(
      conflictError(
        orderErrorReasons.LOCK_BASIS_CHANGED,
        '申请提交时的审核依据已变化，请刷新整票费用后重新核对',
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

    fireEvent.click(await screen.findByText('审核'));
    expect(await screen.findByText('40% → 30%')).toBeInTheDocument();
    fireEvent.click(
      await screen.findByRole('button', { name: '批准并确认提成影响' }),
    );

    expect(
      await screen.findByText(/请刷新整票费用后重新核对/),
    ).toBeInTheDocument();
    expect(
      screen.getByText('批准并确认提成影响').closest('button'),
    ).toBeDisabled();
  });
  it('驳回原因为必填并随请求提交', async () => {
    rejectSupplement.mockResolvedValue({
      success: true,
      data: supplementRow(),
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

    fireEvent.click(await screen.findByText('审核'));
    fireEvent.click(await screen.findByRole('button', { name: /退\s*回/ }));
    const reasonInput = await screen.findByPlaceholderText(
      '请输入驳回原因（必填）',
    );
    // 不填原因直接确认会被拦截。
    fireEvent.click(screen.getByRole('button', { name: /^确\s*认$/ }));
    expect(await screen.findByText('请输入驳回原因')).toBeInTheDocument();
    expect(rejectSupplement).not.toHaveBeenCalled();

    fireEvent.change(reasonInput, { target: { value: '金额与实际不符' } });
    fireEvent.click(screen.getByRole('button', { name: /^确\s*认$/ }));

    await waitFor(() => expect(rejectSupplement).toHaveBeenCalledTimes(1));
    const [, body] = rejectSupplement.mock.calls[0];
    expect(body.reason).toBe('金额与实际不符');
    expect(body.expectedVersion).toBe('3');
  });
});
