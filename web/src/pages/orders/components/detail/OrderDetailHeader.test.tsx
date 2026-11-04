import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, within } from '@testing-library/react';
import type { ComponentProps } from 'react';
import { describe, expect, it, vi } from 'vitest';
import OrderDetailHeader from './OrderDetailHeader';

vi.mock('@/app/access', () => ({
  useAccess: () => ({ canOperateOrganization: () => true }),
}));
vi.mock('./UnlockRequestHistoryDrawer', () => ({
  default: ({ open }: { open: boolean }) =>
    open ? (
      <div role="dialog" aria-label="撤销审核记录">
        撤销审核历史
      </div>
    ) : null,
}));

function renderHeader(
  canReadAbnormal: boolean,
  overrides: Partial<ComponentProps<typeof OrderDetailHeader>> = {},
) {
  const onOpenAbnormalCase = vi.fn();
  renderWithApp(
    <OrderDetailHeader
      kind="sea-export"
      navigationTitle="海运出口"
      orderId="order-1"
      order={{ id: 'order-1', orderNo: 'SE001', organizationId: 'org-1' }}
      saving={false}
      canManageFee={false}
      canReadAttachments={false}
      canCreatePod={false}
      canReadAbnormal={canReadAbnormal}
      moreMenuItems={[]}
      hasAction={() => false}
      onSave={vi.fn()}
      onOpenReleasePod={vi.fn()}
      onOpenAttachments={vi.fn()}
      onOpenAbnormalCase={onOpenAbnormalCase}
      lockState={{ isLocked: true }}
      lockStateLoading={false}
      lockStateError={null}
      businessWritesDisabled
      businessWriteBlockedReason="订单已锁定"
      onRetryLockState={vi.fn().mockResolvedValue(null)}
      onSynchronizeLockChange={vi.fn().mockResolvedValue(undefined)}
      {...overrides}
    />,
  );
  return onOpenAbnormalCase;
}

describe('订单详情异常查看入口', () => {
  it('仅查看权限且订单锁定时仍可打开面板', () => {
    const open = renderHeader(true);
    fireEvent.click(screen.getByRole('button', { name: /更多操作/ }));
    const button = screen.getByRole('menuitem', { name: /异常情况/ });
    expect(button).not.toHaveAttribute('aria-disabled', 'true');
    fireEvent.click(button);
    expect(open).toHaveBeenCalledTimes(1);
  });

  it('没有查看权限则隐藏入口', () => {
    renderHeader(false);
    fireEvent.click(screen.getByRole('button', { name: /更多操作/ }));
    expect(
      screen.queryByRole('menuitem', { name: /异常情况/ }),
    ).not.toBeInTheDocument();
  });
});

describe('订单详情页头操作层级', () => {
  it('桌面动作与标题同行，仅保存使用主按钮，低频操作收进分组菜单', () => {
    const onSave = vi.fn();
    const onReset = vi.fn();
    const openPod = vi.fn();
    const openAttachments = vi.fn();
    renderHeader(true, {
      canManageFee: true,
      canReadAttachments: true,
      canCreatePod: true,
      hasAction: () => true,
      lockState: {
        isLocked: false,
        canLock: true,
        orderVersion: '1',
        feeContentVersion: '1',
        feeContentFingerprint: 'snapshot',
      },
      businessWritesDisabled: false,
      businessWriteBlockedReason: undefined,
      onSave,
      onReset,
      onOpenReleasePod: openPod,
      onOpenAttachments: openAttachments,
      businessMenuItems: [{ key: 'split', label: '拆票', disabled: true }],
    });
    const area = document.querySelector(
      '.roncin-page-header-extra',
    ) as HTMLElement;
    expect(
      within(area)
        .getAllByRole('button')
        .map((button) => button.textContent?.trim()),
    ).toEqual(['费用录入', '附件', '审核订单', '保存', '更多操作']);
    expect(area.querySelectorAll('.ant-btn-primary')).toHaveLength(1);
    expect(
      document.querySelector('.roncin-page-header-actions-row'),
    ).toBeNull();
    expect(screen.getByText(/未审核/).closest('.ant-tag')).not.toHaveClass(
      'ant-tag-success',
    );
    fireEvent.click(screen.getByRole('button', { name: /保存/ }));
    expect(onSave).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByRole('button', { name: /附件/ }));
    expect(openAttachments).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByRole('button', { name: /更多操作/ }));
    expect(screen.getByRole('menuitem', { name: '拆票' })).toHaveAttribute(
      'aria-disabled',
      'true',
    );
    expect(
      document.querySelectorAll('.ant-dropdown-menu-item-divider'),
    ).toHaveLength(2);
    fireEvent.click(screen.getByRole('menuitem', { name: /放货凭证/ }));
    expect(openPod).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByRole('button', { name: /更多操作/ }));
    fireEvent.click(screen.getByRole('menuitem', { name: /重置修改/ }));
    expect(onReset).toHaveBeenCalledOnce();
  });

  it('锁定时保存、导出和重置保持禁用，异常查看及撤销审核记录可用', () => {
    const openAttachments = vi.fn();
    renderHeader(true, {
      hasAction: () => true,
      canCreatePod: true,
      canReadAttachments: true,
      onOpenAttachments: openAttachments,
      onReset: vi.fn(),
    });
    expect(screen.getByRole('button', { name: /保存/ })).toBeDisabled();
    expect(screen.getByRole('button', { name: /附件/ })).toBeEnabled();
    fireEvent.click(screen.getByRole('button', { name: /附件/ }));
    expect(openAttachments).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByRole('button', { name: /更多操作/ }));
    expect(screen.getByRole('menuitem', { name: /放货凭证/ })).toHaveAttribute(
      'aria-disabled',
      'true',
    );
    expect(screen.getByRole('menuitem', { name: /重置修改/ })).toHaveAttribute(
      'aria-disabled',
      'true',
    );
    fireEvent.click(screen.getByRole('menuitem', { name: /撤销审核记录/ }));
    expect(
      screen.getByRole('dialog', { name: '撤销审核记录' }),
    ).toBeInTheDocument();
  });

  it('没有附件读取权限时隐藏附件入口', () => {
    renderHeader(false);
    expect(
      screen.queryByRole('button', { name: /附件/ }),
    ).not.toBeInTheDocument();
  });

  it('类型扩展贡献的读取入口在审核锁定时仍可使用', () => {
    const open = vi.fn();
    renderHeader(false, {
      businessMenuItems: [
        { key: 'hbl-production', label: 'HBL 制作', onClick: open },
      ],
    });
    fireEvent.click(screen.getByRole('button', { name: /更多操作/ }));
    const item = screen.getByRole('menuitem', { name: 'HBL 制作' });
    expect(item).not.toHaveAttribute('aria-disabled', 'true');
    fireEvent.click(item);
    expect(open).toHaveBeenCalledOnce();
  });
});
