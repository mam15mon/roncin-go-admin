import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import OrderDetailHeader from './OrderDetailHeader';

function renderHeader(canReadAbnormal: boolean) {
  const onOpenAbnormalCase = vi.fn();
  renderWithApp(
    <OrderDetailHeader
      kind="sea-export"
      navigationTitle="海运出口"
      orderId="order-1"
      order={{ id: 'order-1', orderNo: 'SE001', organizationId: 'org-1' }}
      saving={false}
      canManageFee={false}
      canCreatePod={false}
      canReadAbnormal={canReadAbnormal}
      moreMenuItems={[]}
      hasAction={() => false}
      onSave={vi.fn()}
      onOpenReleasePod={vi.fn()}
      onOpenAbnormalCase={onOpenAbnormalCase}
      lockState={{ isLocked: true }}
      lockStateLoading={false}
      lockStateError={null}
      businessWritesDisabled
      businessWriteBlockedReason="订单已锁定"
      onRetryLockState={vi.fn().mockResolvedValue(null)}
      onSynchronizeLockChange={vi.fn().mockResolvedValue(undefined)}
    />,
  );
  return onOpenAbnormalCase;
}

describe('订单详情异常查看入口', () => {
  it('仅查看权限且订单锁定时仍可打开面板', () => {
    const open = renderHeader(true);
    const button = screen.getByRole('button', { name: /异常情况/ });
    expect(button).toBeEnabled();
    fireEvent.click(button);
    expect(open).toHaveBeenCalledTimes(1);
  });

  it('没有查看权限则隐藏入口', () => {
    renderHeader(false);
    expect(
      screen.queryByRole('button', { name: /异常情况/ }),
    ).not.toBeInTheDocument();
  });
});
