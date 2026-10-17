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
