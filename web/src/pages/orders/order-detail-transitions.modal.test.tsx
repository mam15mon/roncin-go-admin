import { useState } from 'react';
import { useModalOwnerSignal } from '@/components/layout/useModalOwnerSignal';
import { getAnyDirtyTabKey } from '@/components/layout/tabCloseGuard';
import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { App } from 'antd';
import { beforeEach, expect, it, vi } from 'vitest';
import { OrderClosureStatus, OrderFlowStatus, OrderTerminationStatus } from '@/enums.generated';
import {
  orderServiceTransitionOrderClosure,
  orderServiceTransitionOrderStatus,
  orderServiceTransitionOrderTermination,
} from '@/services/roncin/orderService';
import { confirmOrderClosure, confirmOrderFlow, confirmOrderTermination } from './order-detail-transitions';

vi.mock('@/services/roncin/orderService', () => ({
  orderServiceTransitionOrderClosure: vi.fn(),
  orderServiceTransitionOrderStatus: vi.fn(),
  orderServiceTransitionOrderTermination: vi.fn(),
}));

beforeEach(() => vi.clearAllMocks());

it.each([
  { name: '主流程', confirm: confirmOrderFlow, api: orderServiceTransitionOrderStatus, target: OrderFlowStatus.ORDER_FLOW_STATUS_BOOKED, reason: '请输入流转原因（选填）', required: false },
  { name: '退关', confirm: confirmOrderTermination, api: orderServiceTransitionOrderTermination, target: OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATING, reason: '请输入原因（必填）', required: true },
  { name: '结案', confirm: confirmOrderClosure, api: orderServiceTransitionOrderClosure, target: OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED, reason: '请输入完结原因（必填）', required: true },
])('$name真实Modal失败留窗保留原因，成功重试关闭且不产生未处理拒绝', async ({ confirm, api, target, reason, required }) => {
  const submit = vi.mocked(api);
  submit
    .mockRejectedValueOnce(new Error('请求已拒绝'))
    .mockResolvedValueOnce(undefined as never)
    .mockResolvedValueOnce({ success: false, data: { id: 'order-1' } })
    .mockResolvedValueOnce({ success: true, data: {} })
    .mockResolvedValueOnce({ success: true, data: { id: 'order-1' } });
  const completed = vi.fn();
  function Entry() {
    const app = App.useApp();
    return <button type="button" onClick={() => confirm(app, { id: 'order-1', version: '7', flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_DRAFT }, target, completed)}>打开确认</button>;
  }
  renderWithApp(<Entry />);
  fireEvent.click(screen.getByText('打开确认'));
  const dialog = await screen.findByRole('dialog');
  const ok = () => within(dialog).getByRole('button', { name: /确\s*定$/ });
  if (required) {
    fireEvent.click(ok());
    await waitFor(() => expect(ok()).not.toHaveClass('ant-btn-loading'));
    expect(submit).not.toHaveBeenCalled();
    expect(dialog).not.toHaveClass('ant-zoom-leave');
  }
  const input = screen.getByPlaceholderText(reason);
  fireEvent.change(input, { target: { value: '保留本次操作原因' } });
  for (let attempt = 1; attempt <= 4; attempt++) {
    fireEvent.click(ok());
    await waitFor(() => expect(submit).toHaveBeenCalledTimes(attempt));
    await waitFor(() => expect(ok()).not.toHaveClass('ant-btn-loading'));
    expect(dialog).not.toHaveClass('ant-zoom-leave');
    expect(input).toHaveValue('保留本次操作原因');
    expect(completed).not.toHaveBeenCalled();
  }
  fireEvent.click(ok());
  await waitFor(() => expect(completed).toHaveBeenCalledOnce());
  await waitFor(() => expect(dialog).toHaveClass('ant-zoom-leave'));
  expect(submit).toHaveBeenCalledTimes(5);
});

it('原因输入标脏；所属订单页卸载销毁 App modal，不能在新工作区提交旧订单', async () => {
  window.history.replaceState({}, '', '/orders/sea-export/order-1');
  function Entry() {
    const app = App.useApp();
    const getOwnerSignal = useModalOwnerSignal();
    return (
      <button
        type="button"
        onClick={() =>
          confirmOrderClosure(
            app,
            { id: 'order-1', version: '7' },
            OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED,
            vi.fn(),
            { signal: getOwnerSignal() },
          )
        }
      >
        打开订单结案
      </button>
    );
  }
  function Shell() {
    const [visible, setVisible] = useState(true);
    return (
      <>
        <button type="button" onClick={() => setVisible(false)}>
          销毁旧工作区
        </button>
        {visible && <Entry />}
      </>
    );
  }
  renderWithApp(<Shell />);
  fireEvent.click(screen.getByText('打开订单结案'));
  expect(getAnyDirtyTabKey()).toBeUndefined();
  fireEvent.change(screen.getByPlaceholderText('请输入完结原因（必填）'), {
    target: { value: '北京订单待提交原因' },
  });
  expect(getAnyDirtyTabKey()).toBe('/orders/sea-export');
  fireEvent.click(screen.getByText('销毁旧工作区'));
  expect(getAnyDirtyTabKey()).toBeUndefined();
  await waitFor(() =>
    expect(screen.getByRole('dialog')).toHaveClass('ant-zoom-leave'),
  );
  fireEvent.click(
    within(screen.getByRole('dialog')).getByRole('button', {
      name: /确\s*定$/,
    }),
  );
  expect(orderServiceTransitionOrderClosure).not.toHaveBeenCalled();
});
