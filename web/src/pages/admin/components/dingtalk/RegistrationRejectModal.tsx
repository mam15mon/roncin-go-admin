import type { ProFormInstance } from '@ant-design/pro-components';
import { ModalForm, ProFormTextArea } from '@ant-design/pro-components';
import { App, Typography } from 'antd';
import React from 'react';
import { adminServiceRejectDingTalkRegistration } from '@/services/roncin/adminService';

const { Text } = Typography;

export type RejectFormValues = {
  reason: string;
};

interface RegistrationRejectModalProps {
  registration?: API.DingTalkRegistration;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  formRef: React.RefObject<ProFormInstance | undefined>;
  onReload: () => void;
}

/** 注册审批「拒绝」模态：拒绝原因必填，留痕并通过钉钉通知本人。 */
export default function RegistrationRejectModal({
  registration,
  open,
  onOpenChange,
  formRef,
  onReload,
}: RegistrationRejectModalProps) {
  const { message } = App.useApp();

  if (!registration) return null;

  return (
    <ModalForm<RejectFormValues>
      title={`拒绝注册：${registration.displayName || '未知成员'}`}
      width={520}
      open={open}
      formRef={formRef}
      modalProps={{
        destroyOnHidden: true,
        onCancel: () => onOpenChange(false),
      }}
      onOpenChange={onOpenChange}
      onFinish={async (values) => {
        const userId = registration.userId ?? '';
        await adminServiceRejectDingTalkRegistration(
          { id: userId },
          { id: userId, reason: values.reason?.trim() ?? '' },
        );
        message.success('已拒绝该注册，账号将保持禁用并通知本人');
        onOpenChange(false);
        onReload();
        return true;
      }}
    >
      <Text
        type="secondary"
        style={{ display: 'block', marginBottom: 16, fontSize: 12 }}
      >
        拒绝原因将写入审计留痕，并通过钉钉工作通知发送给本人；账号保持禁用，
        如需再次加入需重新扫码注册或使用手机号邀请。
      </Text>
      <ProFormTextArea
        name="reason"
        label="拒绝原因"
        placeholder="请填写拒绝原因，例如：非本企业在职人员"
        fieldProps={{ maxLength: 200, showCount: true, rows: 3 }}
        rules={[
          { required: true, message: '请填写拒绝原因' },
          { max: 200, message: '拒绝原因不能超过 200 字' },
        ]}
      />
    </ModalForm>
  );
}
