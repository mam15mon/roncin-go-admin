import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { OrderBusinessType } from '@/enums.generated';
import { orderLockServiceLockOrder } from '@/services/roncin/orderLockService';
import OrderLockControl from './OrderLockControl';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceGetOrderFeeReview: vi.fn().mockResolvedValue({
    success: true,
    data: { orderId: 'air-order', contentVersion: '8', fees: [] },
  }),
}));
vi.mock('@/services/roncin/orderLockService', () => ({
  orderLockServiceLockOrder: vi.fn(),
  orderLockServiceRequestOrderUnlock: vi.fn(),
}));
vi.mock('./UnlockRequestHistoryDrawer', () => ({ default: () => null }));

const state: API.OrderLockStateData = {
  orderId: 'air-order',
  businessType: OrderBusinessType.BUSINESS_TYPE_AE,
  orderVersion: '17',
  feeContentVersion: '8',
  feeContentFingerprint: 'fees-8',
  airDocumentFingerprint: 'master-house-link-1',
  canLock: true,
};
function Control({
  snapshot,
  loading = false,
  error = null,
}: {
  snapshot: API.OrderLockStateData;
  loading?: boolean;
  error?: Error | null;
}) {
  return (
    <OrderLockControl
      orderId="air-order"
      state={snapshot}
      loading={loading}
      error={error}
      onRetry={vi.fn()}
      onSynchronize={vi.fn()}
    />
  );
}

describe('航空审核单证基线', () => {
  it('缺少航空单证指纹时不允许打开审核', () => {
    renderWithApp(
      <Control snapshot={{ ...state, airDocumentFingerprint: undefined }} />,
    );
    expect(screen.getByRole('button', { name: /锁定订单/ })).toBeDisabled();
  });

  it('订单与费用版本没变但主单改变时阻止旧弹窗确认，不借新指纹提交', async () => {
    const view = renderWithApp(<Control snapshot={state} />);
    fireEvent.click(screen.getByRole('button', { name: /锁定订单/ }));
    const confirm = await screen.findByRole('button', { name: '确认锁定' });
    await waitFor(() => expect(confirm).toBeEnabled());
    view.rerender(
      <Control
        snapshot={{ ...state, airDocumentFingerprint: 'master-house-link-2' }}
      />,
    );
    await waitFor(() => expect(confirm).toBeDisabled());
    expect(
      screen.getByText('订单或费用已变化，请刷新后重新锁定'),
    ).toBeInTheDocument();
    fireEvent.click(confirm);
    expect(orderLockServiceLockOrder).not.toHaveBeenCalled();
  });

  it('版本和指纹未变但已知锁定资格撤销时阻止旧弹窗确认', async () => {
    const view = renderWithApp(<Control snapshot={state} />);
    fireEvent.click(screen.getByRole('button', { name: /锁定订单/ }));
    const confirm = await screen.findByRole('button', { name: '确认锁定' });
    await waitFor(() => expect(confirm).toBeEnabled());
    view.rerender(<Control snapshot={{ ...state, canLock: false }} />);
    await waitFor(() => expect(confirm).toBeDisabled());
    fireEvent.click(confirm);
    expect(orderLockServiceLockOrder).not.toHaveBeenCalled();
  });

  it('重查在途或失败时禁用确认，恢复成功后仍提交原单证指纹', async () => {
    const view = renderWithApp(<Control snapshot={state} />);
    fireEvent.click(screen.getByRole('button', { name: /锁定订单/ }));
    const confirm = await screen.findByRole('button', { name: '确认锁定' });
    await waitFor(() => expect(confirm).toBeEnabled());
    view.rerender(<Control snapshot={state} loading />);
    await waitFor(() => expect(confirm).toBeDisabled());
    view.rerender(<Control snapshot={state} error={new Error('重查失败')} />);
    await waitFor(() => expect(confirm).toBeDisabled());
    view.rerender(<Control snapshot={state} />);
    await waitFor(() => expect(confirm).toBeEnabled());
    fireEvent.click(confirm);
    await waitFor(() => expect(orderLockServiceLockOrder).toHaveBeenCalled());
    expect(orderLockServiceLockOrder).toHaveBeenLastCalledWith(
      { orderId: 'air-order' },
      expect.objectContaining({
        expectedOrderVersion: '17',
        expectedFeeContentVersion: '8',
        airDocumentFingerprint: 'master-house-link-1',
      }),
    );
  });
});
