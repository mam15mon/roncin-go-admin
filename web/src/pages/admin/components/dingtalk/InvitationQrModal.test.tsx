import { fireEvent, render, screen } from '@testing-library/react';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { DingTalkInvitationKind } from '@/enums.generated';

const messageSuccessMock = vi.fn();

vi.mock('antd', async (importOriginal) => ({
  // 透传真实 antd：order-list-template 在模块顶层解构 DatePicker，
  // 局部 mock 缺导出会导致整个测试套件加载失败。
  ...(await importOriginal<typeof import('antd')>()),
  App: {
    useApp: () => ({
      message: { success: messageSuccessMock, error: vi.fn() },
    }),
  },
  Button: ({
    children,
    onClick,
    disabled,
  }: {
    children?: React.ReactNode;
    onClick?: () => void;
    disabled?: boolean;
  }) => (
    <button type="button" onClick={onClick} disabled={disabled}>
      {children}
    </button>
  ),
  Input: ({
    value,
    addonAfter,
  }: {
    value?: string;
    addonAfter?: React.ReactNode;
  }) => (
    <div>
      <input readOnly value={value} />
      {addonAfter}
    </div>
  ),
  Modal: ({
    open,
    children,
    footer,
  }: {
    open?: boolean;
    children?: React.ReactNode;
    footer?: React.ReactNode;
  }) =>
    open ? (
      <div>
        {children}
        {footer}
      </div>
    ) : null,
  QRCode: ({ value }: { value: string }) => (
    <div data-testid="qrcode">
      <canvas />
      {value}
    </div>
  ),
  Space: ({ children }: { children?: React.ReactNode }) => (
    <div>{children}</div>
  ),
  Tag: ({ children }: { children?: React.ReactNode }) => (
    <span>{children}</span>
  ),
  Typography: {
    Text: ({ children }: { children?: React.ReactNode }) => (
      <span>{children}</span>
    ),
  },
}));

import InvitationQrModal from './InvitationQrModal';

describe('InvitationQrModal', () => {
  beforeEach(() => {
    messageSuccessMock.mockReset();
  });

  it('展示通用入职码信息与完整链接', () => {
    const onOpenChange = vi.fn();
    render(
      <InvitationQrModal
        open
        onOpenChange={onOpenChange}
        invitation={{
          organizationName: '天津分公司',
          kind: DingTalkInvitationKind.DING_TALK_INVITATION_KIND_GENERIC,
          token: 'token-generic-12345',
          expiresAt: '2026-09-20T12:00:00Z',
        }}
        invitationUrl="/login?invite=token-generic-12345"
      />,
    );

    expect(screen.getByText('天津分公司')).toBeInTheDocument();
    expect(screen.getByText('通用入职码（多人有效）')).toBeInTheDocument();
    expect(
      screen.getByText(/新员工使用手机钉钉扫码即可提交入职申请/),
    ).toBeInTheDocument();

    const input = screen.getByRole('textbox') as HTMLInputElement;
    expect(input.value).toContain('/login?invite=token-generic-12345');
    expect(screen.getByTestId('qrcode')).toHaveTextContent(
      '/login?invite=token-generic-12345',
    );

    // 点击完成按钮触发关闭
    fireEvent.click(screen.getByText('完成'));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it('展示定向邀请码并提示目标手机号', () => {
    render(
      <InvitationQrModal
        open
        onOpenChange={() => {}}
        invitation={{
          organizationName: '成都分公司',
          kind: DingTalkInvitationKind.DING_TALK_INVITATION_KIND_TARGETED,
          mobileMasked: '138****8000',
          token: 'token-targeted-56789',
        }}
      />,
    );

    expect(screen.getByText('成都分公司')).toBeInTheDocument();
    expect(screen.getByText('定向邀请码（单人有效）')).toBeInTheDocument();
    expect(screen.getByText(/138\*\*\*\*8000/)).toBeInTheDocument();
  });

  it('支持点击下载二维码按钮', () => {
    HTMLCanvasElement.prototype.toDataURL = vi
      .fn()
      .mockReturnValue('data:image/png;base64,mock');
    render(
      <InvitationQrModal
        open
        onOpenChange={() => {}}
        invitation={{
          organizationName: '青岛分公司',
          kind: DingTalkInvitationKind.DING_TALK_INVITATION_KIND_GENERIC,
          token: 'token-generic-999',
        }}
        invitationUrl="/login?invite=token-generic-999"
      />,
    );

    const downloadBtn = screen.getByText('下载二维码');
    expect(downloadBtn).toBeInTheDocument();
    expect(downloadBtn).not.toBeDisabled();
    fireEvent.click(downloadBtn);
    expect(messageSuccessMock).toHaveBeenCalledWith('二维码已下载');
  });
});
