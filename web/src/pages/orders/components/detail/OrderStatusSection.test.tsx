import { fireEvent, render, screen } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { describe, expect, it, vi } from 'vitest';
import {
  OrderClosureStatus,
  OrderFlowStatus,
  OrderTerminationStatus,
} from '@/enums.generated';
import { formatDate } from '@/utils/format';
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
  it('当前节点明确标识，下一步保持可流转且不冒充当前节点', () => {
    renderStatusSection(
      {
        flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_BOOKED,
        allowedTargetFlowStatuses: [
          OrderFlowStatus.ORDER_FLOW_STATUS_SPACE_ALLOCATED,
        ],
        statusRecords: [{ dimension: 'FLOW', status: 'BOOKED' }],
      },
      { transitionFlow: vi.fn() },
    );

    expect(screen.getByText('当前：已订舱')).toBeInTheDocument();
    expect(screen.getByText('当前').parentElement).toHaveAttribute(
      'aria-current',
      'step',
    );
    expect(screen.getByText('当前').parentElement).toHaveTextContent('已订舱');
    expect(
      screen.getByRole('button', { name: '流转到已配舱' }),
    ).toHaveTextContent('可流转');
    expect(
      screen
        .getByRole('button', { name: '流转到已配舱' })
        .querySelector('[aria-current]'),
    ).toBeNull();
  });

  it('草稿不将已订舱标为当前，跳过的拖车节点也不显示已完成', () => {
    const { unmount } = renderStatusSection({
      flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_DRAFT,
    });
    expect(screen.getByText('当前：草稿')).toBeInTheDocument();
    expect(screen.queryByText('当前')).not.toBeInTheDocument();
    unmount();

    renderStatusSection({
      flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_DOCUMENT_CUTOFF,
      statusRecords: [
        { dimension: 'FLOW', status: 'BOOKED' },
        { dimension: 'FLOW', status: 'SPACE_ALLOCATED' },
        { dimension: 'FLOW', status: 'DOCUMENT_CUTOFF' },
      ],
    });
    expect(screen.getByText('拖车已安排').parentElement).toHaveTextContent(
      '未完成',
    );
    expect(screen.getAllByText('已完成')).toHaveLength(2);
  });

  it('已完成节点悬停显示真实操作人、本地时间和多行备注', async () => {
    const changedAt = '2026-10-23T02:03:04Z';
    const transitionFlow = vi.fn();
    renderStatusSection(
      {
        flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_SPACE_ALLOCATED,
        statusRecords: [
          {
            dimension: 'FLOW',
            status: 'BOOKED',
            operatorName: '张操作',
            changedAt,
            reason: '舱位已确认\n客户已知悉',
          },
        ],
      },
      { transitionFlow },
    );

    fireEvent.mouseOver(screen.getByText('已订舱'));
    expect(await screen.findByText('操作人：张操作')).toBeInTheDocument();
    expect(
      screen.getByText(`操作时间：${formatDate(changedAt, 'datetime')}`),
    ).toBeInTheDocument();
    expect(screen.getByText(/备注：舱位已确认/)).toHaveTextContent(
      '客户已知悉',
    );
    expect(transitionFlow).not.toHaveBeenCalled();
  });

  it('悬停无备注的当前节点明确显示无备注，未操作节点不伪造人员', async () => {
    const { unmount } = renderStatusSection({
      flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_BOOKED,
      statusRecords: [
        {
          dimension: 'FLOW',
          status: 'BOOKED',
          operatorName: '李操作',
          reason: '  ',
        },
      ],
    });
    fireEvent.mouseOver(screen.getByText('已订舱'));
    expect(await screen.findByText('备注：无备注')).toBeInTheDocument();
    unmount();

    renderStatusSection({
      flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_BOOKED,
    });
    fireEvent.mouseOver(screen.getByText('已配舱'));
    expect(await screen.findByText('暂无操作记录')).toBeInTheDocument();
    expect(screen.queryByText(/操作人：/)).not.toBeInTheDocument();
  });

  it('退关与完结徽标也使用对应维度的实际记录', async () => {
    const { unmount } = renderStatusSection({
      terminationStatus:
        OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED,
      statusRecords: [
        {
          dimension: 'TERMINATION',
          status: 'TERMINATED',
          operatorName: '王操作',
          reason: '客户取消出运',
        },
      ],
    });
    fireEvent.mouseOver(screen.getByText('已退关'));
    expect(await screen.findByText('备注：客户取消出运')).toBeInTheDocument();
    unmount();

    renderStatusSection({
      closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED,
      statusRecords: [
        {
          dimension: 'CLOSURE',
          status: 'CLOSED',
          operatorName: '赵操作',
          reason: '业务已完成',
        },
      ],
    });
    fireEvent.mouseOver(screen.getByText('已完结'));
    expect(await screen.findByText('操作人：赵操作')).toBeInTheDocument();
    expect(screen.getByText('备注：业务已完成')).toBeInTheDocument();
  });

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
