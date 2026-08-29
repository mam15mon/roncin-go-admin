import type { ProFormInstance } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormSwitch,
} from '@ant-design/pro-components';
import { Alert, App } from 'antd';
import React, { useRef } from 'react';
import { ProFormSearchableSelect } from '@/components/ui';
import {
  adminServiceCreateUserMembership,
  adminServiceListOrganizationRoles,
  adminServiceUpdateUserMembership,
} from '@/services/roncin/adminService';
import type { UserMembershipFormValues } from './userConstants';

interface UserMembershipModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  userId?: string;
  membershipEditing?: API.AdminUserMembership;
  memberships: API.AdminUserMembership[];
  organizations: API.AdminOrganization[];
  membershipRoles: API.AdminRole[];
  onMembershipRolesChange: (roles: API.AdminRole[]) => void;
  currentUserId?: string;
  onSaved: () => Promise<void> | void;
}

export default function UserMembershipModal({
  open,
  onOpenChange,
  userId,
  membershipEditing,
  memberships,
  organizations,
  membershipRoles,
  onMembershipRolesChange,
  currentUserId,
  onSaved,
}: UserMembershipModalProps) {
  const membershipFormRef = useRef<ProFormInstance | undefined>(undefined);
  const { message } = App.useApp();

  return (
    <ModalForm<UserMembershipFormValues>
      title={membershipEditing ? '编辑组织成员关系' : '加入其他组织'}
      open={open}
      formRef={membershipFormRef}
      initialValues={
        membershipEditing
          ? {
              organizationId: membershipEditing.organizationId,
              roleIds: membershipEditing.roleIds ?? [],
              enabled: membershipEditing.enabled ?? true,
              primary: membershipEditing.primary ?? false,
            }
          : { enabled: true, primary: memberships.length === 0 }
      }
      modalProps={{
        destroyOnClose: true,
        width: 520,
        onCancel: () => onOpenChange(false),
      }}
      onOpenChange={onOpenChange}
      onFinish={async (values) => {
        if (!userId) return false;
        if (membershipEditing?.id) {
          await adminServiceUpdateUserMembership(
            { userId, id: membershipEditing.id },
            {
              userId,
              id: membershipEditing.id,
              roleIds: values.roleIds ?? [],
              enabled: values.enabled ?? true,
              primary: values.primary ?? false,
            },
          );
          message.success('组织成员关系已更新');
        } else {
          await adminServiceCreateUserMembership(
            { userId },
            {
              userId,
              organizationId: values.organizationId ?? '',
              roleIds: values.roleIds ?? [],
              primary: values.primary ?? false,
            },
          );
          message.success('用户已加入组织');
        }
        onOpenChange(false);
        await onSaved();
        return true;
      }}
    >
      <Alert
        showIcon
        type="info"
        title="各组织的角色和状态相互独立"
        description="设为主要组织后，用户下次登录将默认进入该组织；停用会立即撤销该组织中的在线会话。"
        style={{ marginBottom: 16 }}
      />
      <ProFormSearchableSelect
        name="organizationId"
        label="所属组织"
        placeholder="请选择公司、部门或组"
        disabled={Boolean(membershipEditing)}
        options={organizations
          .filter(
            (organization) =>
              membershipEditing?.organizationId === organization.id ||
              !memberships.some(
                (membership) => membership.organizationId === organization.id,
              ),
          )
          .map((organization) => ({
            label: `${organization.name} (${organization.code})`,
            value: organization.id,
            code: organization.code,
            name: organization.name,
          }))}
        rules={[{ required: true, message: '请选择所属组织' }]}
        fieldProps={{
          onChange: async (organizationId: string) => {
            membershipFormRef.current?.setFieldValue('roleIds', []);
            const response = await adminServiceListOrganizationRoles({
              organizationId,
            });
            onMembershipRolesChange(response.data ?? []);
          },
        }}
      />
      <ProFormSearchableSelect
        name="roleIds"
        label="组织角色"
        mode="multiple"
        placeholder="请选择该组织中的角色"
        options={membershipRoles.map((role) => ({
          label: `${role.name} (${role.code})`,
          value: role.id,
          code: role.code,
          name: role.name,
        }))}
      />
      <ProFormSwitch
        name="primary"
        label="主要组织"
        extra={
          membershipEditing?.primary
            ? '如需更换主要组织，请在另一条启用的成员关系中将其设为主要组织。'
            : '开启后会自动取消原主要组织，用户下次登录将默认进入这里。'
        }
        disabled={membershipEditing?.primary}
      />
      {membershipEditing && (
        <ProFormSwitch
          name="enabled"
          label="成员关系状态"
          extra="停用后用户不能进入该组织，但不会影响其在其他组织中的访问。"
          disabled={userId === currentUserId && membershipEditing.enabled}
        />
      )}
    </ModalForm>
  );
}
