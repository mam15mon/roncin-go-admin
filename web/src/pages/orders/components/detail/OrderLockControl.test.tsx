import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { OrderBusinessType } from '@/enums.generated';
import {
  orderLockServiceLockOrder,
  orderLockServiceRequestOrderUnlock,
} from '@/services/roncin/orderLockService';
import OrderLockControl, {
  getOrderLockConfirmationDescription,
  getOrderLockSourceDisplay,
  OrderLockStatusTag,
} from './OrderLockControl';

vi.mock('@/services/roncin/orderLockService', () => ({
  orderLockServiceLockOrder: vi.fn(),
  orderLockServiceRequestOrderUnlock: vi.fn(),
  orderLockServiceListOrderUnlockRequests: vi.fn(),
}));

vi.mock('./UnlockRequestHistoryDrawer', () => ({
  default: () => null,
}));

const lockOrder = vi.mocked(orderLockServiceLockOrder);
const requestUnlock = vi.mocked(orderLockServiceRequestOrderUnlock);

describe('OrderLockControl', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('按生成业务类型枚举区分 SE 与非 SE 锁定提示', () => {
    expect(
      getOrderLockConfirmationDescription(OrderBusinessType.BUSINESS_TYPE_SE),
    ).toContain('MBL/HBL');
    expect(
      getOrderLockConfirmationDescription(OrderBusinessType.BUSINESS_TYPE_AI),
    ).toContain('费用变更须申请补录并补审');
  });

  it('状态标签显示服务端返回的业务类型与锁状态', () => {
    render(
      <OrderLockStatusTag
        state={{
          businessType: OrderBusinessType.BUSINESS_TYPE_AI,
          isLocked: true,
          lockedByName: '测试用户',
          lockGeneration: '2',
        }}
        loading={false}
        error={null}
      />,
    );

    expect(
      screen.getByText(/空运进口 · 已审核锁定 · 测试用户/),
    ).toBeInTheDocument();
  });

  it('动作完全使用服务端 can 字段，并以锁状态版本发起锁定', async () => {
    lockOrder.mockResolvedValue({
      success: true,
      data: { state: { orderId: 'order-ai' } },
    } as Awaited<ReturnType<typeof lockOrder>>);
    const onSynchronize = vi.fn().mockResolvedValue(undefined);

    render(
      <App>
        <OrderLockControl
          orderId="order-ai"
          orderNo="AI20260905001"
          state={{
            orderId: 'order-ai',
            orderNo: 'AI20260905001',
            businessType: OrderBusinessType.BUSINESS_TYPE_AI,
            isLocked: false,
            canLock: true,
            orderVersion: '17',
            feeContentVersion: '8',
            feeContentFingerprint: 'server-fingerprint-8',
          }}
          loading={false}
          error={null}
          onRetry={vi.fn().mockResolvedValue(null)}
          onSynchronize={onSynchronize}
        />
      </App>,
    );

    expect(screen.queryByRole('button', { name: '直接解锁' })).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: /审核并锁定/ }));
    expect(
      (await screen.findAllByText('审核并锁定空运进口订单')).length,
    ).toBeGreaterThan(0);
    expect(screen.getByText(/订单业务资料和费用进入锁定/)).toBeInTheDocument();
    expect(screen.queryByText(/MBL\/HBL/)).toBeNull();

    fireEvent.click(screen.getByRole('button', { name: /确.*认/ }));
    await waitFor(() => expect(lockOrder).toHaveBeenCalledTimes(1));
    expect(lockOrder.mock.calls[0][1]).toMatchObject({
      expectedOrderVersion: '17',
      expectedFeeContentVersion: '8',
      feeContentFingerprint: 'server-fingerprint-8',
    });
    await waitFor(() => expect(onSynchronize).toHaveBeenCalledTimes(1));
  });

  it('只呈现服务端允许的审批解锁动作，版本缺失时禁用', () => {
    render(
      <App>
        <OrderLockControl
          orderId="order-si"
          state={{
            businessType: OrderBusinessType.BUSINESS_TYPE_SI,
            isLocked: true,
            canRoleDirectUnlock: false,
            canAdminEmergencyUnlock: false,
            canRequestUnlock: true,
          }}
          loading={false}
          error={null}
          onRetry={vi.fn().mockResolvedValue(null)}
          onSynchronize={vi.fn().mockResolvedValue(undefined)}
        />
      </App>,
    );

    expect(screen.queryByRole('button', { name: '直接解锁' })).toBeNull();
    expect(screen.queryByRole('button', { name: '紧急解锁' })).toBeNull();
    expect(screen.getByRole('button', { name: /申请解锁/ })).toBeDisabled();
    expect(requestUnlock).not.toHaveBeenCalled();
  });

  it('审核来源只展示实际审核人', () => {
    expect(
      getOrderLockSourceDisplay({ lockedBy: 'user-1', lockedByName: '张三' }),
    ).toEqual({ actorText: '张三' });
    expect(getOrderLockSourceDisplay({}).actorText).toBe('审核人待确认');
  });

  it('亏损时意见必填，失败保留审核快照和意见供重试', async () => {
    lockOrder.mockResolvedValueOnce(undefined as never).mockResolvedValueOnce({
      success: true,
      data: { state: { orderId: 'loss-order' } },
    });
    render(
      <App>
        <OrderLockControl
          orderId="loss-order"
          state={{
            canLock: true,
            orderVersion: '3',
            feeContentVersion: '9',
            feeContentFingerprint: 'loss-fingerprint',
            feeSummary: {
              baseCurrency: 'CNY',
              receivable: '100',
              payable: '150',
              profit: '-50',
            },
          }}
          loading={false}
          error={null}
          onRetry={vi.fn()}
          onSynchronize={vi.fn().mockResolvedValue(undefined)}
        />
      </App>,
    );
    fireEvent.click(screen.getByRole('button', { name: /审核并锁定/ }));
    fireEvent.click(screen.getByRole('button', { name: /确.*认/ }));
    expect(await screen.findByText('请填写亏损原因')).toBeInTheDocument();
    expect(lockOrder).not.toHaveBeenCalled();
    fireEvent.change(screen.getByRole('textbox', { name: '审核意见' }), {
      target: { value: '已核实完整成本' },
    });
    fireEvent.click(screen.getByRole('button', { name: /确.*认/ }));
    await waitFor(() => expect(lockOrder).toHaveBeenCalledOnce());
    expect(screen.getByRole('textbox', { name: '审核意见' })).toHaveValue(
      '已核实完整成本',
    );
    fireEvent.click(screen.getByRole('button', { name: /确.*认/ }));
    await waitFor(() => expect(lockOrder).toHaveBeenCalledTimes(2));
    expect(lockOrder.mock.calls[1][1]).toEqual(lockOrder.mock.calls[0][1]);
  });
  it('服务端未返回费用审核快照时禁止锁定', () => {
    render(
      <App>
        <OrderLockControl
          orderId="missing-snapshot"
          state={{ canLock: true, orderVersion: '1' }}
          loading={false}
          error={null}
          onRetry={vi.fn()}
          onSynchronize={vi.fn()}
        />
      </App>,
    );
    expect(screen.getByRole('button', { name: /审核并锁定/ })).toBeDisabled();
    expect(lockOrder).not.toHaveBeenCalled();
  });
});
