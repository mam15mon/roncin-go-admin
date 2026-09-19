import { CopyOutlined, DownloadOutlined } from '@ant-design/icons';
import { App, Button, Input, Modal, QRCode, Space, Tag } from 'antd';
import dayjs from 'dayjs';
import React, { useRef } from 'react';
import { MODAL_SIZE } from '@/components/ui';
import { DingTalkInvitationKind } from '@/enums.generated';

export interface InvitationQrModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  invitation?: API.DingTalkInvitation;
  invitationUrl?: string;
}

export default function InvitationQrModal({
  open,
  onOpenChange,
  invitation,
  invitationUrl,
}: InvitationQrModalProps) {
  const { message } = App.useApp();
  const qrContainerRef = useRef<HTMLDivElement>(null);

  const fullUrl = React.useMemo(() => {
    if (invitationUrl) {
      if (invitationUrl.startsWith('http')) return invitationUrl;
      const origin =
        typeof window !== 'undefined' ? window.location.origin : '';
      return `${origin}${invitationUrl}`;
    }
    if (invitation?.token) {
      const origin =
        typeof window !== 'undefined' ? window.location.origin : '';
      return `${origin}/login?invite=${invitation.token}`;
    }
    return '';
  }, [invitationUrl, invitation?.token]);

  const isGeneric =
    invitation?.kind ===
    DingTalkInvitationKind.DING_TALK_INVITATION_KIND_GENERIC;

  const handleCopy = async () => {
    if (!fullUrl) return;
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(fullUrl);
      } else {
        const textarea = document.createElement('textarea');
        textarea.value = fullUrl;
        document.body.appendChild(textarea);
        textarea.select();
        document.execCommand('copy');
        document.body.removeChild(textarea);
      }
      message.success('邀请链接已复制到剪贴板');
    } catch {
      message.error('复制失败，请手动选择链接复制');
    }
  };

  const handleDownload = () => {
    const canvas =
      qrContainerRef.current?.querySelector<HTMLCanvasElement>('canvas');
    if (!canvas) {
      message.error('未找到二维码图片，无法下载');
      return;
    }
    const url = canvas.toDataURL('image/png');
    const a = document.createElement('a');
    const name = invitation?.organizationName
      ? `${invitation.organizationName}-邀请二维码`
      : '邀请二维码';
    a.download = `${name}.png`;
    a.href = url;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    message.success('二维码已下载');
  };

  return (
    <Modal
      title="邀请二维码与链接"
      open={open}
      onCancel={() => onOpenChange(false)}
      footer={[
        <Button
          key="download"
          icon={<DownloadOutlined />}
          onClick={handleDownload}
          disabled={!fullUrl}
        >
          下载二维码
        </Button>,
        <Button key="close" type="primary" onClick={() => onOpenChange(false)}>
          完成
        </Button>,
      ]}
      width={MODAL_SIZE.SM}
      destroyOnClose
      centered
    >
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          padding: '16px 0 8px',
          gap: 16,
        }}
      >
        <Space size={8}>
          <Tag color="blue">{invitation?.organizationName || '目标组织'}</Tag>
          {isGeneric ? (
            <Tag color="geekblue">通用入职码（多人有效）</Tag>
          ) : (
            <Tag color="cyan">定向邀请码（单人有效）</Tag>
          )}
        </Space>

        <div
          style={{
            padding: 16,
            background: '#ffffff',
            borderRadius: 8,
            border: '1px solid #f0f0f0',
            boxShadow: '0 2px 8px rgba(0, 0, 0, 0.04)',
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
          }}
        >
          {fullUrl ? (
            <div ref={qrContainerRef} data-testid="invitation-qrcode-container">
              <QRCode value={fullUrl} size={190} bordered={false} />
            </div>
          ) : (
            <div style={{ width: 190, height: 190, background: '#f5f5f5' }} />
          )}
        </div>

        <div style={{ textAlign: 'center', fontSize: 12, color: '#8c8c8c' }}>
          {isGeneric ? (
            <span>新员工使用手机钉钉扫码即可提交入职申请，由管理员审批。</span>
          ) : (
            <span>
              限目标手机号（{invitation?.mobileMasked || '-'}
              ）使用钉钉扫码，匹配后免审自动激活。
            </span>
          )}
          {invitation?.expiresAt && (
            <div style={{ marginTop: 4 }}>
              有效期至：
              {dayjs(invitation.expiresAt).format('YYYY-MM-DD HH:mm')}
            </div>
          )}
        </div>

        <div style={{ width: '100%', marginTop: 4 }}>
          <Input
            value={fullUrl}
            readOnly
            addonAfter={
              <Button
                type="link"
                size="small"
                icon={<CopyOutlined />}
                onClick={handleCopy}
                style={{ padding: '0 4px', height: 'auto' }}
              >
                复制
              </Button>
            }
          />
        </div>
      </div>
    </Modal>
  );
}
