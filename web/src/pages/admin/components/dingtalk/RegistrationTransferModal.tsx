import type { ProFormInstance } from '@ant-design/pro-components';
import { ModalForm, ProFormTextArea } from '@ant-design/pro-components';
import { Alert, App, Avatar, Space, Typography } from 'antd';
import React, { useEffect } from 'react';
import { MODAL_SIZE, ProFormSearchableSelect } from '@/components/ui';
import { adminServiceTransferDingTalkRegistration } from '@/services/roncin/adminService';
import { organizationSelectOptions } from './constants';

const { Text } = Typography;

export type TransferFormValues = {
  targetOrganizationId: string;
  reason: string;
};

interface RegistrationTransferModalProps {
  registration?: API.DingTalkRegistration;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  formRef: React.RefObject<ProFormInstance | undefined>;
  organizations: API.AdminOrganization[];
  onReload: () => void;
}

/**
 * 待审批注册「一键转派」模态：将误选本组织的申请转派至正确的兄弟分公司。
 * 转派后原审批人不再收到待办，新分公司审批人（带向上追溯兜底）将收到通知。
 */
export default function RegistrationTransferModal({
  registration,
  open,
  onOpenChange,
  formRef,
  organizations,
  onReload,
}: RegistrationTransferModalProps) {
  const { message } = App.useApp();

  useEffect(() => {
    if (!open) return;
    formRef.current?.resetFields();
  }, [open, formRef]);

  if (!registration) return null;

  // 过滤排除当前申请所在的组织（禁止原地转派）
  const candidateOrganizations = organizations.filter(
    (org) => org.id && org.id !== registration.requestedOrganizationId,
  );

  return (
    <ModalForm<TransferFormValues>
      title={`转派待审批注册：${registration.displayName || '未提供姓名'}`}
      width={MODAL_SIZE.SM}
      open={open}
      formRef={formRef}
      modalProps={{
        destroyOnClose: true,
        onCancel: () => onOpenChange(false),
      }}
      onOpenChange={onOpenChange}
      onFinish={async (values) => {
        if (!registration.userId) return false;
        await adminServiceTransferDingTalkRegistration(
          { userId: registration.userId },
          {
            userId: registration.userId,
            targetOrganizationId: values.targetOrganizationId,
            reason: values.reason.trim(),
          },
        );
        message.success('已转派至目标分公司审批');
        onOpenChange(false);
        onReload();
        return true;
      }}
    >
      <Alert
        type="info"
        showIcon
        message={
          <Space direction="vertical" size={2}>
            <span>
              转派后将把该人员的申请组织更新为目标分公司，并向目标分公司的审批管理员发送钉钉通知。
            </span>
            <span style={{ fontSize: 12, color: '#8c8c8c' }}>
              当前自选组织：
              {registration.requestedOrganizationName || '总部收口'}
            </span>
          </Space>
        }
        style={{ marginBottom: 16 }}
      />

      <Space size={12} align="center" style={{ marginBottom: 20 }}>
        <Avatar
          size={40}
          src={registration.avatarUrl}
          style={{
            backgroundColor: '#1677ff',
            fontSize: 16,
            fontWeight: 600,
          }}
        >
          {registration.displayName
            ? registration.displayName.charAt(0).toUpperCase()
            : '?'}
        </Avatar>
        <div>
          <div style={{ fontWeight: 600, fontSize: 15 }}>
            {registration.displayName || '未提供姓名'}
          </div>
          <Text type="secondary" style={{ fontSize: 12 }}>
            当前归属：
            {registration.requestedOrganizationName || '未指定（总部收口）'}
          </Text>
        </div>
      </Space>

      <ProFormSearchableSelect
        name="targetOrganizationId"
        label="目标分公司"
        placeholder="请选择要转派到的分公司"
        options={organizationSelectOptions(candidateOrganizations)}
        rules={[{ required: true, message: '请选择目标分公司' }]}
      />

      <ProFormTextArea
        name="reason"
        label="转派原因"
        placeholder="请输入转派原因（例如：该员工属于天津港现场调度，由天津分公司审核入职）"
        fieldProps={{ maxLength: 200, showCount: true, rows: 3 }}
        rules={[
          { required: true, message: '请输入转派原因' },
          { max: 200, message: '转派原因最多 200 个字' },
        ]}
      />
    </ModalForm>
  );
}
