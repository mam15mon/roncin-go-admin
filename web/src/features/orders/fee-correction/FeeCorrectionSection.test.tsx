import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { FeeCorrectionSection } from './FeeCorrectionSection';

const api = vi.hoisted(() => ({
  list: vi.fn(),
  preview: vi.fn(),
  approve: vi.fn(),
  reject: vi.fn(),
  withdraw: vi.fn(),
}));
vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: { id: 'user-1', currentOrganization: { id: 'org-1' } },
    },
  }),
}));
vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceListOrderFeeCorrectionRequests: api.list,
  orderFeeServicePreviewOrderFeeCorrectionApproval: api.preview,
  orderFeeServiceApproveOrderFeeCorrection: api.approve,
  orderFeeServiceRejectOrderFeeCorrection: api.reject,
  orderFeeServiceWithdrawOrderFeeCorrection: api.withdraw,
}));
const request: API.OrderFeeCorrectionRequestData = {
  id: 'r1',
  orderId: 'o1',
  operation: 2,
  status: 'PENDING',
  version: '4',
  canApprove: true,
  reason: '客户优惠',
  requestedByName: '业务员',
  beforeFee: {
    feeName: '运费',
    totalAmount: '1000',
    currency: 'CNY',
    direction: 1,
  },
  afterFee: {
    feeName: '运费',
    totalAmount: '800',
    currency: 'CNY',
    direction: 1,
  },
};
const preview: API.OrderFeeCorrectionApprovalData = {
  request,
  baseCurrency: 'CNY',
  currentReceivable: '1000',
  currentPayable: '400',
  currentProfit: '600',
  projectedReceivable: '800',
  projectedPayable: '400',
  projectedProfit: '400',
  previewToken: 'actual-cash-snapshot',
  commissionImpacts: [
    {
      commissionId: 'c1',
      employeeId: 'e1',
      orderId: 'o1',
      commissionNo: 'COM-1',
      currentAmount: '30',
      targetAmount: '25',
      difference: '-5',
      baseCurrency: 'CNY',
    },
  ],
};

describe('费用更正审核', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    api.list.mockResolvedValue({ success: true, data: [request], total: 1 });
    api.preview.mockResolvedValue({ success: true, data: preview });
    api.approve.mockResolvedValue({
      success: true,
      data: { ...request, status: 'APPROVED' },
    });
  });
  it('展示实际回款重算的25元提成并提交同一预览与版本', async () => {
    const changed = vi.fn();
    renderWithClient(
      <App>
        <FeeCorrectionSection orderId="o1" onChanged={changed} />
      </App>,
    );
    fireEvent.click(await screen.findByRole('button', { name: '审核' }));
    expect(await screen.findByText('25.00')).toBeInTheDocument();
    expect(screen.getByText('-5.00')).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText('审批意见'), {
      target: { value: '优惠已确认' },
    });
    fireEvent.click(screen.getByRole('button', { name: '批准更正' }));
    await waitFor(() =>
      expect(api.approve).toHaveBeenCalledWith(
        { orderId: 'o1', id: 'r1' },
        expect.objectContaining({
          expectedVersion: '4',
          previewToken: 'actual-cash-snapshot',
          comment: '优惠已确认',
        }),
      ),
    );
    await waitFor(() => expect(changed).toHaveBeenCalledOnce());
    await waitFor(() =>
      expect(screen.getByRole('dialog').className).toContain('leave-active'),
    );
    await act(async () => {
      fireEvent.animationEnd(screen.getByRole('dialog'));
    });
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
    );
    await act(async () => {});
  });
  it('审批被拒绝后保留意见并重新获取金额快照', async () => {
    api.approve.mockResolvedValueOnce(undefined);
    renderWithClient(
      <App>
        <FeeCorrectionSection orderId="o1" />
      </App>,
    );
    fireEvent.click(await screen.findByRole('button', { name: '审核' }));
    await screen.findByText('25.00');
    fireEvent.change(screen.getByLabelText('审批意见'), {
      target: { value: '保留的意见' },
    });
    fireEvent.click(screen.getByRole('button', { name: '批准更正' }));
    await waitFor(() => expect(api.preview).toHaveBeenCalledTimes(2));
    expect(screen.getByLabelText('审批意见')).toHaveValue('保留的意见');
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    await act(async () => {});
  });
  it('刷新预览失败时禁止提交缓存中的旧金额快照', async () => {
    api.approve.mockResolvedValueOnce(undefined);
    api.preview.mockResolvedValueOnce({ success: true, data: preview });
    api.preview.mockRejectedValue(new Error('预览请求失败'));
    renderWithClient(
      <App>
        <FeeCorrectionSection orderId="o1" />
      </App>,
    );
    fireEvent.click(await screen.findByRole('button', { name: '审核' }));
    await screen.findByText('25.00');
    fireEvent.click(screen.getByRole('button', { name: '批准更正' }));
    await screen.findByRole('button', { name: '重新预览' });
    expect(screen.getByRole('button', { name: '批准更正' })).toBeDisabled();
    expect(api.approve).toHaveBeenCalledOnce();
    await act(async () => {});
  });
  it('只读历史不展示审批动作，也不请求审批预览', async () => {
    api.list.mockResolvedValue({
      success: true,
      data: [{ ...request, canApprove: false }],
      total: 1,
    });
    renderWithClient(
      <App>
        <FeeCorrectionSection orderId="o1" />
      </App>,
    );
    await screen.findByText('客户优惠');
    expect(
      screen.queryByRole('button', { name: '审核' }),
    ).not.toBeInTheDocument();
    expect(api.preview).not.toHaveBeenCalled();
    await act(async () => {});
  });
});
