import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { App } from 'antd';
import { useState } from 'react';
import { beforeEach, expect, it, vi } from 'vitest';
import { getAnyDirtyTabKey } from '@/components/layout/tabCloseGuard';
import { useModalOwnerSignal } from '@/components/layout/useModalOwnerSignal';
import {
  OrderClosureStatus,
  OrderFlowStatus,
  OrderTerminationStatus,
} from '@/enums.generated';
import {
  orderServiceTransitionOrderClosure,
  orderServiceTransitionOrderStatus,
  orderServiceTransitionOrderTermination,
} from '@/services/roncin/orderService';
import {
  confirmOrderClosure,
  confirmOrderFlow,
  confirmOrderTermination,
} from './order-detail-transitions';

vi.mock('@/services/roncin/orderService', () => ({
  orderServiceTransitionOrderClosure: vi.fn(),
  orderServiceTransitionOrderStatus: vi.fn(),
  orderServiceTransitionOrderTermination: vi.fn(),
}));

beforeEach(() => vi.clearAllMocks());

it('完结订单不填写原因也能提交并关闭确认框', async () => {
  vi.mocked(orderServiceTransitionOrderClosure).mockResolvedValueOnce({
    success: true,
    data: { id: 'order-1' },
  });
  const completed = vi.fn();
  function Entry() {
    const app = App.useApp();
    return (
      <button
        type="button"
        onClick={() =>
          confirmOrderClosure(
            app,
            { id: 'order-1', version: '7' },
            OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED,
            completed,
          )
        }
      >
        直接完结
      </button>
    );
  }
  renderWithApp(<Entry />);
  fireEvent.click(screen.getByText('直接完结'));
  const dialog = await screen.findByRole('dialog');
  expect(within(dialog).queryByRole('textbox')).not.toBeInTheDocument();
  fireEvent.click(within(dialog).getByRole('button', { name: /确\s*定$/ }));
  await waitFor(() => expect(completed).toHaveBeenCalledOnce());
  expect(orderServiceTransitionOrderClosure).toHaveBeenCalledWith(
    { id: 'order-1' },
    {
      id: 'order-1',
      expectedVersion: '7',
      targetStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED,
    },
  );
  await waitFor(() => expect(dialog).toHaveClass('ant-zoom-leave'));
});

it.each([
  {
    name: '主流程',
    confirm: confirmOrderFlow,
    api: orderServiceTransitionOrderStatus,
    target: OrderFlowStatus.ORDER_FLOW_STATUS_BOOKED,
    required: false,
  },
  {
    name: '退关',
    confirm: confirmOrderTermination,
    api: orderServiceTransitionOrderTermination,
    target: OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATING,
    required: true,
  },
  {
    name: '结案',
    confirm: confirmOrderClosure,
    api: orderServiceTransitionOrderClosure,
    target: OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED,
    required: false,
  },
])(
  '$name真实Modal失败留窗，成功重试关闭；仅退关收集原因',
  async ({ confirm, api, target, required }) => {
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
      return (
        <button
          type="button"
          onClick={() =>
            confirm(
              app,
              {
                id: 'order-1',
                version: '7',
                flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_DRAFT,
              },
              target,
              completed,
            )
          }
        >
          打开确认
        </button>
      );
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
    const input = within(dialog).queryByRole('textbox');
    if (required) {
      fireEvent.change(within(dialog).getByRole('textbox'), {
        target: { value: '保留本次操作原因' },
      });
    } else {
      expect(input).not.toBeInTheDocument();
    }
    for (let attempt = 1; attempt <= 4; attempt++) {
      fireEvent.click(ok());
      await waitFor(() => expect(submit).toHaveBeenCalledTimes(attempt));
      await waitFor(() => expect(ok()).not.toHaveClass('ant-btn-loading'));
      expect(dialog).not.toHaveClass('ant-zoom-leave');
      if (required) expect(input).toHaveValue('保留本次操作原因');
      expect(completed).not.toHaveBeenCalled();
    }
    fireEvent.click(ok());
    await waitFor(() => expect(completed).toHaveBeenCalledOnce());
    await waitFor(() => expect(dialog).toHaveClass('ant-zoom-leave'));
    expect(submit).toHaveBeenCalledTimes(5);
  },
);

it('原因输入标脏；所属订单页卸载销毁 App modal，不能在新工作区提交旧订单', async () => {
  window.history.replaceState({}, '', '/orders/sea-export/order-1');
  function Entry() {
    const app = App.useApp();
    const getOwnerSignal = useModalOwnerSignal();
    return (
      <button
        type="button"
        onClick={() =>
          confirmOrderTermination(
            app,
            { id: 'order-1', version: '7' },
            OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATING,
            vi.fn(),
            { signal: getOwnerSignal() },
          )
        }
      >
        打开订单退关
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
  fireEvent.click(screen.getByText('打开订单退关'));
  expect(getAnyDirtyTabKey()).toBeUndefined();
  fireEvent.change(screen.getByPlaceholderText('请输入原因（必填）'), {
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
  expect(orderServiceTransitionOrderTermination).not.toHaveBeenCalled();
});
