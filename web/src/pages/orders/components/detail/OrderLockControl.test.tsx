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
  getOrderLockTriggerTypeText,
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
    ).toBe('锁定后将冻结订单业务资料和费用。如需修改必须先解锁。');
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
      screen.getByText(/空运进口 · 已锁定 · 测试用户/),
    ).toBeInTheDocument();
  });

  it('动作完全使用服务端 can 字段，并以锁状态版本发起锁定', async () => {
    lockOrder.mockResolvedValue({ success: true } as Awaited<
      ReturnType<typeof lockOrder>
    >);
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
          }}
          loading={false}
          error={null}
          onRetry={vi.fn().mockResolvedValue(null)}
          onSynchronize={onSynchronize}
        />
      </App>,
    );

    expect(screen.queryByRole('button', { name: '直接解锁' })).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: /锁定订单/ }));
    expect(
      (await screen.findAllByText('锁定空运进口订单')).length,
    ).toBeGreaterThan(0);
    expect(screen.getByText(/冻结订单业务资料和费用/)).toBeInTheDocument();
    expect(screen.queryByText(/MBL\/HBL/)).toBeNull();

    fireEvent.click(screen.getByRole('button', { name: '确认锁定' }));
    await waitFor(() => expect(lockOrder).toHaveBeenCalledTimes(1));
    expect(lockOrder.mock.calls[0][1].expectedOrderVersion).toBe('17');
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

  it('人工锁定展示实际锁定人，不因 locked_by 缺失猜测来源', () => {
    expect(
      getOrderLockSourceDisplay({
        lockSource: 'MANUAL',
        lockedBy: 'user-1',
        lockedByName: '张三',
      }),
    ).toEqual({ actorText: '张三', isAuto: false, triggerLines: [] });

    // 无锁定人时回落中性文案，绝不因 locked_by 为空推断为系统自动锁定。
    expect(
      getOrderLockSourceDisplay({
        lockSource: 'MANUAL',
        lockedByName: undefined,
      }).actorText,
    ).toBe('人工锁定');
    expect(
      getOrderLockSourceDisplay({
        lockSource: undefined,
        lockedByName: undefined,
      }).isAuto,
    ).toBe(false);
  });

  it('自动锁定固定展示系统自动锁定并携带触发审计详情', () => {
    render(
      <OrderLockStatusTag
        state={{
          businessType: OrderBusinessType.BUSINESS_TYPE_SE,
          isLocked: true,
          lockSource: 'AUTO_SETTLEMENT',
          currentLockRecord: {
            lockSource: 'AUTO_SETTLEMENT',
            triggerType: 'VERIFICATION',
            triggerResourceId: 'VR-2026-001',
            triggeredByName: '李四',
          },
        }}
        loading={false}
        error={null}
      />,
    );

    expect(screen.getByText(/系统自动锁定/)).toBeInTheDocument();
    expect(screen.queryByText(/测试用户/)).not.toBeInTheDocument();

    expect(
      getOrderLockSourceDisplay({
        lockSource: 'AUTO_SETTLEMENT',
        currentLockRecord: {
          lockSource: 'AUTO_SETTLEMENT',
          triggerType: 'VERIFICATION',
          triggerResourceId: 'VR-2026-001',
          triggeredByName: '李四',
        },
      }).triggerLines,
    ).toEqual([
      '触发类型：应收核销',
      '触发单据：VR-2026-001',
      '触发操作人：李四',
    ]);
    expect(getOrderLockTriggerTypeText('FEE_BILLED')).toBe('费用进入账单');
    expect(getOrderLockTriggerTypeText('FEE_CANCEL')).toBe('未建账费用删除');
    expect(getOrderLockTriggerTypeText('UNKNOWN_KIND')).toBe('UNKNOWN_KIND');
    expect(getOrderLockTriggerTypeText(undefined)).toBe('-');
  });

  it('自动锁定后显式解锁入口保持可用', () => {
    render(
      <App>
        <OrderLockControl
          orderId="order-se-auto"
          state={{
            businessType: OrderBusinessType.BUSINESS_TYPE_SE,
            isLocked: true,
            lockSource: 'AUTO_SETTLEMENT',
            orderVersion: '9',
            canRoleDirectUnlock: true,
            canRequestUnlock: true,
          }}
          loading={false}
          error={null}
          onRetry={vi.fn().mockResolvedValue(null)}
          onSynchronize={vi.fn().mockResolvedValue(undefined)}
        />
      </App>,
    );

    expect(screen.getByRole('button', { name: /直接解锁/ })).toBeEnabled();
    expect(screen.getByRole('button', { name: /申请解锁/ })).toBeEnabled();
  });
});
