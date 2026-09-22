import { fireEvent, render, screen } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { describe, expect, it, vi } from 'vitest';
import {
  OrderClosureStatus,
  OrderFlowStatus,
  OrderTerminationStatus,
} from '@/enums.generated';
import {
  buildOrderStatusSection,
  type OrderStatusSectionActions,
} from './OrderStatusSection';

function renderStatusSection(
  order: Partial<API.Order>,
  actions: OrderStatusSectionActions = {},
) {
  const section = buildOrderStatusSection(order as API.Order, actions);
  return render(
    <App>
      {section.extra}
      {section.content}
    </App>,
  );
}

describe('OrderStatusSection', () => {
  it('仅将服务端允许的草稿下一节点开放为按钮', () => {
    const transitionFlow = vi.fn();
    renderStatusSection(
      {
        flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_DRAFT,
        allowedTargetFlowStatuses: [OrderFlowStatus.ORDER_FLOW_STATUS_BOOKED],
      },
      { transitionFlow },
    );

    fireEvent.click(screen.getByRole('button', { name: '流转到已订舱' }));

    expect(transitionFlow).toHaveBeenCalledWith(
      OrderFlowStatus.ORDER_FLOW_STATUS_BOOKED,
    );
    expect(
      screen.queryByRole('button', { name: '流转到已配舱' }),
    ).not.toBeInTheDocument();
  });

  it('完整保留已配舱阶段服务端给出的两个合法分支', () => {
    const transitionFlow = vi.fn();
    renderStatusSection(
      {
        flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_SPACE_ALLOCATED,
        allowedTargetFlowStatuses: [
          OrderFlowStatus.ORDER_FLOW_STATUS_TRUCKING_ARRANGED,
          OrderFlowStatus.ORDER_FLOW_STATUS_DOCUMENT_CUTOFF,
        ],
      },
      { transitionFlow },
    );

    fireEvent.click(screen.getByRole('button', { name: '流转到拖车已安排' }));
    fireEvent.click(screen.getByRole('button', { name: '流转到已截单' }));

    expect(transitionFlow).toHaveBeenNthCalledWith(
      1,
      OrderFlowStatus.ORDER_FLOW_STATUS_TRUCKING_ARRANGED,
    );
    expect(transitionFlow).toHaveBeenNthCalledWith(
      2,
      OrderFlowStatus.ORDER_FLOW_STATUS_DOCUMENT_CUTOFF,
    );
  });

  it('业务写门禁关闭时保留目标提示但禁止流转', () => {
    renderStatusSection(
      {
        flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_DRAFT,
        allowedTargetFlowStatuses: [OrderFlowStatus.ORDER_FLOW_STATUS_BOOKED],
      },
      {
        transitionFlow: vi.fn(),
        flowDisabled: true,
        flowDisabledReason: '订单已锁定',
      },
    );

    expect(screen.getByRole('button', { name: '流转到已订舱' })).toBeDisabled();
  });

  it('退关状态分别提供发起、完成取消菜单和恢复订单入口', async () => {
    const start = vi.fn();
    const complete = vi.fn();
    const cancel = vi.fn();
    const actions = {
      startTermination: { onClick: start },
      completeTermination: { onClick: complete },
      cancelTermination: { onClick: cancel },
    };
    const { rerender } = renderStatusSection(
      {
        terminationStatus:
          OrderTerminationStatus.ORDER_TERMINATION_STATUS_ACTIVE,
      },
      actions,
    );

    fireEvent.click(screen.getByRole('button', { name: '发起退关' }));
    expect(start).toHaveBeenCalledOnce();

    let section = buildOrderStatusSection(
      {
        terminationStatus:
          OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATING,
      } as API.Order,
      actions,
    );
    rerender(
      <App>
        {section.extra}
        {section.content}
      </App>,
    );
    fireEvent.click(screen.getByRole('button', { name: '选择退关操作' }));
    fireEvent.click(await screen.findByRole('menuitem', { name: '完成退关' }));
    expect(complete).toHaveBeenCalledOnce();

    section = buildOrderStatusSection(
      {
        terminationStatus:
          OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED,
      } as API.Order,
      actions,
    );
    rerender(
      <App>
        {section.extra}
        {section.content}
      </App>,
    );
    fireEvent.click(screen.getByRole('button', { name: '恢复订单' }));
    expect(cancel).toHaveBeenCalledOnce();
  });

  it('未完结无动作时只读，有动作和已完结时开放对应入口', () => {
    const close = vi.fn();
    const reopen = vi.fn();
    const { rerender } = renderStatusSection({
      closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_OPEN,
    });

    expect(screen.getByRole('button', { name: '完结订单' })).toBeDisabled();
    expect(screen.getByText('未完结')).toBeInTheDocument();

    let section = buildOrderStatusSection(
      {
        closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_OPEN,
      } as API.Order,
      { close: { onClick: close } },
    );
    rerender(
      <App>
        {section.extra}
        {section.content}
      </App>,
    );
    fireEvent.click(screen.getByRole('button', { name: '完结订单' }));
    expect(close).toHaveBeenCalledOnce();

    section = buildOrderStatusSection(
      {
        closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED,
      } as API.Order,
      { reopen: { onClick: reopen } },
    );
    rerender(
      <App>
        {section.extra}
        {section.content}
      </App>,
    );
    fireEvent.click(screen.getByRole('button', { name: '反结案' }));
    expect(reopen).toHaveBeenCalledOnce();
  });
});
