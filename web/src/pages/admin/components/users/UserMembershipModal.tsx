import type { ProFormInstance } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDependency,
  ProFormSwitch,
} from '@ant-design/pro-components';
import { useQuery } from '@tanstack/react-query';
import { Alert, App } from 'antd';
import React, { useRef } from 'react';
import { ProFormSearchableSelect } from '@/components/ui';
import {
  adminServiceCreateUserMemberships,
  adminServiceListOrganizationRoles,
  adminServiceUpdateUserMembership,
} from '@/services/roncin/adminService';
import { ensureListResponse, unwrapList } from '@/utils/api';
import {
  formatOrganizationHierarchyName,
  type UserMembershipFormValues,
} from './userConstants';

interface UserMembershipModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  userId?: string;
  membershipEditing?: API.AdminUserMembership;
  memberships: API.AdminUserMembership[];
  organizations: API.AdminOrganization[];
  currentUserId?: string;
  onSaved: () => Promise<void> | void;
}

function OrganizationRolesField({
  organizationId,
  label,
  name,
  open,
}: {
  organizationId: string;
  label: string;
  name: string | string[];
  open: boolean;
}) {
  const rolesQuery = useQuery({
    queryKey: ['admin-users', 'organization-roles', organizationId],
    enabled: open && Boolean(organizationId),
    queryFn: async () =>
      unwrapList(
        ensureListResponse(
          await adminServiceListOrganizationRoles({ organizationId }),
          '读取组织角色失败',
        ),
      ),
  });
  return (
    <ProFormSearchableSelect
      name={name}
      label={label}
      mode="multiple"
      preserve={false}
      placeholder="选填，可不分配角色"
      extra={
        rolesQuery.isError
          ? '角色候选读取失败，可保留空角色或重新打开后重试'
          : undefined
      }
      options={(rolesQuery.data ?? [])
        .filter((role) => role.enabled)
        .map((role) => ({
          label: `${role.name} (${role.code})`,
          value: role.id,
          code: role.code,
          name: role.name,
        }))}
      fieldProps={{ loading: rolesQuery.isFetching }}
    />
  );
}

export default function UserMembershipModal({
  open,
  onOpenChange,
  userId,
  membershipEditing,
  memberships,
  organizations,
  currentUserId,
  onSaved,
}: UserMembershipModalProps) {
  const membershipFormRef = useRef<ProFormInstance | undefined>(undefined);
  const { message } = App.useApp();
  const availableOrganizations = organizations.filter(
    (organization) =>
      membershipEditing?.organizationId === organization.id ||
      (organization.enabled &&
        !memberships.some(
          (membership) => membership.organizationId === organization.id,
        )),
  );

  return (
    <ModalForm<UserMembershipFormValues>
      key={membershipEditing?.id ?? 'create'}
      title={membershipEditing ? '编辑组织成员关系' : '加入其他组织'}
      open={open}
      formRef={membershipFormRef}
      initialValues={
        membershipEditing
          ? {
              organizationId: membershipEditing.organizationId,
              roleIds: membershipEditing.roleIds ?? [],
              enabled: membershipEditing.enabled ?? true,
            }
          : { organizationIds: [], rolesByOrganization: {} }
      }
      modalProps={{
        destroyOnHidden: true,
        width: 600,
        onCancel: () => onOpenChange(false),
      }}
      onOpenChange={onOpenChange}
      onFinish={async (values) => {
        if (!userId) return false;
        if (membershipEditing?.id) {
          const response = await adminServiceUpdateUserMembership(
            { userId, id: membershipEditing.id },
            {
              userId,
              id: membershipEditing.id,
              roleIds: values.roleIds ?? [],
              enabled: values.enabled ?? true,
            },
          );
          if (!response?.success || !response.data?.id) return false;
          message.success('组织成员关系已更新');
        } else {
          const items = (values.organizationIds ?? []).map(
            (organizationId) => ({
              organizationId,
              roleIds: values.rolesByOrganization?.[organizationId] ?? [],
            }),
          );
          if (items.length === 0) return false;
          const response = await adminServiceCreateUserMemberships(
            { userId },
            { userId, items },
          );
          if (
            !response?.success ||
            response.data?.length !== items.length ||
            response.data.some((item) => !item.id)
          )
            return false;
          message.success(`用户已加入 ${items.length} 个组织`);
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
        description={
          membershipEditing
            ? '角色仅在所属工作台生效；停用会立即撤销该组织中的在线会话。'
            : '每个组织可单独选填角色；一次保存全部组织，任一项失败时保留本次输入。'
        }
        style={{ marginBottom: 16 }}
      />
      <ProFormSearchableSelect
        name={membershipEditing ? 'organizationId' : 'organizationIds'}
        label="所属组织"
        mode={membershipEditing ? undefined : 'multiple'}
        placeholder="请选择公司、部门或组"
        disabled={Boolean(membershipEditing)}
        options={availableOrganizations.map((organization) => ({
          label: `${formatOrganizationHierarchyName(organization, organizations)} (${organization.code})`,
          value: organization.id,
          code: organization.code,
          name: organization.name,
        }))}
        rules={[{ required: true, message: '请选择所属组织' }]}
        fieldProps={{
          onChange: (organizationIds: string[]) => {
            if (membershipEditing) return;
            const previous =
              membershipFormRef.current?.getFieldValue('rolesByOrganization') ??
              {};
            membershipFormRef.current?.setFieldValue(
              'rolesByOrganization',
              Object.fromEntries(
                organizationIds.map((id) => [id, previous[id] ?? []]),
              ),
            );
          },
        }}
      />
      {membershipEditing ? (
        <OrganizationRolesField
          organizationId={membershipEditing.organizationId ?? ''}
          label="组织角色"
          name="roleIds"
          open={open}
        />
      ) : (
        <ProFormDependency name={['organizationIds']}>
          {({ organizationIds }) =>
            (organizationIds ?? []).map((organizationId: string) => (
              <OrganizationRolesField
                key={organizationId}
                organizationId={organizationId}
                label={`${formatOrganizationHierarchyName(organizationId, organizations)} · 角色（选填）`}
                name={['rolesByOrganization', organizationId]}
                open={open}
              />
            ))
          }
        </ProFormDependency>
      )}
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
