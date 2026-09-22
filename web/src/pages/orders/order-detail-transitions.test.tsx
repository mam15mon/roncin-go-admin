import { fireEvent, render, screen } from '@testing-library/react';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
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

const transitionStatus = vi.mocked(orderServiceTransitionOrderStatus);
const transitionTermination = vi.mocked(orderServiceTransitionOrderTermination);
const transitionClosure = vi.mocked(orderServiceTransitionOrderClosure);

function createApp() {
  let config: any;
  return {
    app: {
      modal: {
        confirm: vi.fn((nextConfig) => {
          config = nextConfig;
          return {};
        }),
      },
      message: {
        success: vi.fn(),
        error: vi.fn(),
      },
    } as any,
    getConfig: () => config,
  };
}

describe('订单详情状态确认函数', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('固定用户点击的目标状态并携带最新版本提交一次', async () => {
    const { app, getConfig } = createApp();
    const onCompleted = vi.fn();
    transitionStatus.mockResolvedValue({ data: { id: 'order-1' } } as any);

    confirmOrderFlow(
      app,
      {
        id: 'order-1',
        version: '7',
        flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_DRAFT,
      } as API.Order,
      OrderFlowStatus.ORDER_FLOW_STATUS_BOOKED,
      onCompleted,
    );

    render(getConfig().content);
    expect(screen.getByText(/当前状态：草稿/)).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText('流转原因'), {
      target: { value: '已确认舱位' },
    });
    await getConfig().onOk();

    expect(transitionStatus).toHaveBeenCalledOnce();
    expect(transitionStatus).toHaveBeenCalledWith(
      { id: 'order-1' },
      {
        id: 'order-1',
        expectedVersion: '7',
        targetFlowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_BOOKED,
        reason: '已确认舱位',
      },
    );
    expect(onCompleted).toHaveBeenCalledOnce();
  });

  it('空响应不会误报主流程成功或刷新', async () => {
    const { app, getConfig } = createApp();
    const onCompleted = vi.fn();
    transitionStatus.mockResolvedValue(undefined as any);

    confirmOrderFlow(
      app,
      {
        id: 'order-1',
        version: '7',
        flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_DRAFT,
      } as API.Order,
      OrderFlowStatus.ORDER_FLOW_STATUS_BOOKED,
      onCompleted,
    );

    await expect(getConfig().onOk()).rejects.toBeUndefined();
    expect(app.message.success).not.toHaveBeenCalled();
    expect(onCompleted).not.toHaveBeenCalled();
  });

  it('已退关恢复使用恢复订单标题且空响应不报成功', async () => {
    const { app, getConfig } = createApp();
    transitionTermination.mockResolvedValue(undefined as any);

    confirmOrderTermination(
      app,
      {
        id: 'order-1',
        version: '8',
        terminationStatus:
          OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED,
      } as API.Order,
      OrderTerminationStatus.ORDER_TERMINATION_STATUS_ACTIVE,
      vi.fn(),
    );

    expect(getConfig().title).toBe('恢复订单');
    render(getConfig().content);
    fireEvent.change(screen.getByPlaceholderText('请输入原因（必填）'), {
      target: { value: '重新承接业务' },
    });
    await expect(getConfig().onOk()).rejects.toBeUndefined();
    expect(app.message.success).not.toHaveBeenCalled();
  });

  it.each([
    [OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED, '请输入完结原因'],
    [OrderClosureStatus.ORDER_CLOSURE_STATUS_OPEN, '请输入反结案原因'],
  ])('完结与反结案原因均为必填', async (targetStatus, errorMessage) => {
    const { app, getConfig } = createApp();
    confirmOrderClosure(
      app,
      { id: 'order-1', version: '9' } as API.Order,
      targetStatus,
      vi.fn(),
    );

    await expect(getConfig().onOk()).rejects.toBeUndefined();

    expect(app.message.error).toHaveBeenCalledWith(errorMessage);
    expect(transitionClosure).not.toHaveBeenCalled();
  });
});
