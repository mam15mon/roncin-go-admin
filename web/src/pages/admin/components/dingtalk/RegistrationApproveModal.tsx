import type { ProFormInstance } from '@ant-design/pro-components';
import { ModalForm } from '@ant-design/pro-components';
import { Alert, App, Avatar, Space, Typography } from 'antd';
import React, { useEffect, useState } from 'react';
import { ProFormSearchableSelect } from '@/components/ui';
import { adminServiceApproveDingTalkRegistration } from '@/services/roncin/adminService';
import { formatDate } from '@/utils/format';
import {
  fetchRolesForOrganization,
  resolveRootOrganizationId,
  roleSelectOptions,
} from './constants';

const { Text } = Typography;

export type ApproveFormValues = {
  roleIds: string[];
};

interface RegistrationApproveModalProps {
  registration?: API.DingTalkRegistration;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  formRef: React.RefObject<ProFormInstance | undefined>;
  /** 全量组织列表（总部兜底注册需从中解析根组织以加载角色）。 */
  organizations: API.AdminOrganization[];
  currentOrganizationId?: string;
  canReadRoles: boolean;
  onReload: () => void;
}

/**
 * 注册审批「同意」模态：一站式完成启用账号 + 目标组织成员资格 + 初始角色 +
 * 通知本人；路由组织为自选目标组织，未自选时按总部兜底（组织树根）。
 */
export default function RegistrationApproveModal({
  registration,
  open,
  onOpenChange,
  formRef,
  organizations,
  currentOrganizationId,
  canReadRoles,
  onReload,
}: RegistrationApproveModalProps) {
  const { message } = App.useApp();
  const [roles, setRoles] = useState<API.AdminRole[]>([]);

  const routingOrganizationId =
    registration?.requestedOrganizationId ??
    resolveRootOrganizationId(organizations) ??
    currentOrganizationId;

  useEffect(() => {
    if (!open) return;
    formRef.current?.resetFields();
    setRoles([]);
    if (canReadRoles && routingOrganizationId) {
      fetchRolesForOrganization(routingOrganizationId, currentOrganizationId)
        .then(setRoles)
        .catch(() => setRoles([]));
    }
  }, [
    open,
    canReadRoles,
    routingOrganizationId,
    currentOrganizationId,
    formRef,
  ]);

  if (!registration) return null;

  return (
    <ModalForm<ApproveFormValues>
      title={`同意注册：${registration.displayName || '未知成员'}`}
      width={520}
      open={open}
      formRef={formRef}
      modalProps={{
        destroyOnClose: true,
        onCancel: () => onOpenChange(false),
      }}
      onOpenChange={onOpenChange}
      onFinish={async (values) => {
        const userId = registration.userId ?? '';
        await adminServiceApproveDingTalkRegistration(
          { id: userId },
          { id: userId, roleIds: values.roleIds ?? [] },
        );
        message.success('已同意注册：账号启用、成员资格与初始角色已生效');
        onOpenChange(false);
        onReload();
        return true;
      }}
    >
      <Space size={10} align="center" style={{ marginBottom: 12 }}>
        <Avatar
          size={40}
          src={registration.avatarUrl}
          style={{ backgroundColor: '#1677ff', fontWeight: 600 }}
        >
          {(registration.displayName || '?').charAt(0).toUpperCase()}
        </Avatar>
        <div>
          <div style={{ fontWeight: 600 }}>
            {registration.displayName || '-'}
          </div>
          <Text type="secondary" style={{ fontSize: 12 }}>
            注册时间：{formatDate(registration.registeredAt)}
          </Text>
        </div>
      </Space>
      <Alert
        showIcon
        type="info"
        title={
          registration.requestedOrganizationId
            ? `自选目标组织：${registration.requestedOrganizationName || '未知组织'}`
            : '未自选目标组织：按总部兜底处理'
        }
        description="同意后将启用账号、创建目标组织成员资格、授予下列初始角色，并通过钉钉通知本人。"
        style={{ marginBottom: 16 }}
      />
      <ProFormSearchableSelect
        name="roleIds"
        label="初始角色"
        mode="multiple"
        placeholder="请选择授予的初始角色"
        options={roleSelectOptions(roles)}
        rules={[{ required: true, message: '请至少选择一个初始角色' }]}
      />
    </ModalForm>
  );
}
