import { ApartmentOutlined, PlusOutlined } from '@ant-design/icons';
import type { ProFormInstance } from '@ant-design/pro-components';
import { ModalForm, ProFormText } from '@ant-design/pro-components';
import { Alert, App, Button, Popconfirm, Space, Table, Tag, Typography } from 'antd';
import React, { useEffect, useState } from 'react';
import { ProFormSearchableSelect } from '@/components/ui';
import {
  adminServiceAuthorizeDingTalkUser,
  adminServiceAuthorizeWeComUser,
  adminServiceCreateUser,
  adminServiceDeleteUserMembership,
  adminServiceListOrganizationRoles,
  adminServiceListUserMemberships,
  adminServiceUpdateUser,
} from '@/services/roncin/adminService';
import UserMembershipModal from './UserMembershipModal';
import {
  organizationKindLabels,
  pendingExternalProvider,
  type UserFormValues,
} from './userConstants';

const { Text } = Typography;

interface UserFormModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  editing?: API.AdminUser;
  formRef: React.RefObject<ProFormInstance | undefined>;
  roles: API.AdminRole[];
  organizations: API.AdminOrganization[];
  canReadAllUserMemberships: boolean;
  canManageUserMemberships: boolean;
  currentUserId?: string;
  defaultOrganizationId?: string;
  onReload: () => void;
}

export default function UserFormModal({
  open,
  onOpenChange,
  editing,
  formRef,
  roles,
  organizations,
  canReadAllUserMemberships,
  canManageUserMemberships,
  currentUserId,
  defaultOrganizationId,
  onReload,
}: UserFormModalProps) {
  const { message } = App.useApp();
  const [approvalRoles, setApprovalRoles] = useState<API.AdminRole[]>([]);
  const [memberships, setMemberships] = useState<API.AdminUserMembership[]>([]);
  const [membershipsLoading, setMembershipsLoading] = useState(false);
  const [membershipModalOpen, setMembershipModalOpen] = useState(false);
  const [membershipEditing, setMembershipEditing] =
    useState<API.AdminUserMembership>();
  const [membershipRoles, setMembershipRoles] = useState<API.AdminRole[]>([]);
  const pendingProvider = pendingExternalProvider(editing);

  const loadMemberships = async (userId: string) => {
    setMembershipsLoading(true);
    try {
      const response = await adminServiceListUserMemberships({ userId });
      setMemberships(response.data ?? []);
    } finally {
      setMembershipsLoading(false);
    }
  };

  useEffect(() => {
    if (!open) return;
    if (!editing) {
      setMemberships([]);
      return;
    }
    const provider = pendingExternalProvider(editing);
    if (provider) {
      setApprovalRoles(roles);
      setMemberships([]);
    } else if (editing.id && canReadAllUserMemberships) {
      void loadMemberships(editing.id);
    } else {
      setMemberships([]);
    }
  }, [open, editing]);

  const openCreateMembership = () => {
    setMembershipEditing(undefined);
    setMembershipRoles([]);
    setMembershipModalOpen(true);
  };

  const openEditMembership = async (membership: API.AdminUserMembership) => {
    setMembershipEditing(membership);
    setMembershipRoles([]);
    setMembershipModalOpen(true);
    if (membership.organizationId) {
      const response = await adminServiceListOrganizationRoles({
        organizationId: membership.organizationId,
      });
      setMembershipRoles(response.data ?? []);
    }
  };

  return (
    <ModalForm<UserFormValues>
      title={
        editing
          ? `编辑用户：${editing.displayName || editing.username}`
          : '新增用户'
      }
      open={open}
      formRef={formRef}
      initialValues={
        editing
          ? {
              ...editing,
              organizationId: pendingProvider
                ? defaultOrganizationId
                : undefined,
            }
          : undefined
      }
      modalProps={{
        destroyOnClose: true,
        width:
          editing && !pendingProvider && canReadAllUserMemberships
            ? 880
            : 560,
        onCancel: () => onOpenChange(false),
      }}
      onOpenChange={onOpenChange}
      onFinish={async (values) => {
        if (editing?.id && pendingProvider) {
          const authorize =
            pendingProvider === 'wecom'
              ? adminServiceAuthorizeWeComUser
              : adminServiceAuthorizeDingTalkUser;
          await authorize(
            { id: editing.id },
            {
              id: editing.id,
              organizationId: values.organizationId ?? '',
              displayName: values.displayName?.trim() ?? '',
              email: values.email?.trim() || undefined,
              roleIds: values.roleIds ?? [],
            },
          );
          message.success(
            `${pendingProvider === 'wecom' ? '企业微信' : '钉钉'}成员已完成组织授权并启用`,
          );
        } else if (editing?.id) {
          await adminServiceUpdateUser(
            { id: editing.id },
            {
              id: editing.id,
              displayName: values.displayName?.trim() ?? '',
              email: values.email?.trim() || undefined,
              enabled: true,
              roleIds: values.roleIds ?? [],
            },
          );
          message.success('用户已成功更新');
        } else {
          await adminServiceCreateUser({
            username: values.username?.trim() ?? '',
            displayName: values.displayName?.trim() ?? '',
            password: values.password ?? '',
            email: values.email?.trim() || undefined,
            roleIds: values.roleIds ?? [],
          });
          message.success('用户已成功创建');
        }
        onOpenChange(false);
        onReload();
        return true;
      }}
    >
      {pendingProvider && (
        <Alert
          showIcon
          type="info"
          title={`${pendingProvider === 'wecom' ? '企业微信' : '钉钉'}成员 ${pendingProvider === 'wecom' ? editing?.wecomName || editing?.displayName : editing?.dingtalkName || editing?.displayName} 已完成身份登记`}
          description="请分配至少一个角色并启用账号。启用后，该成员再次扫码即可登录。"
          style={{ marginBottom: 16 }}
        />
      )}
      {pendingProvider && (
        <ProFormSearchableSelect
          name="organizationId"
          label="所属组织"
          placeholder="请选择公司、部门或组"
          options={organizations.map((organization) => ({
            label: `${organization.name} (${organization.code})`,
            value: organization.id,
            code: organization.code,
            name: organization.name,
          }))}
          rules={[{ required: true, message: '请选择所属组织' }]}
          fieldProps={{
            onChange: async (organizationId: string) => {
              formRef.current?.setFieldValue('roleIds', []);
              const response = await adminServiceListOrganizationRoles({
                organizationId,
              });
              setApprovalRoles(response.data ?? []);
            },
          }}
        />
      )}
      {(!editing || editing.username) && (
        <ProFormText
          name="username"
          label="用户名（密码登录账号）"
          placeholder="例如：zhangsan 或 logistics_op"
          fieldProps={{ maxLength: 64 }}
          disabled={Boolean(editing)}
          rules={[
            { required: !editing, message: '请输入用户名' },
            { min: 3, max: 64, message: '用户名长度需在 3 至 64 个字符之间' },
            {
              pattern: /^[a-z0-9_.-]+$/,
              message: '用户名仅支持小写字母、数字、点号、下划线及连字符',
            },
          ]}
        />
      )}
      <ProFormText
        name="displayName"
        label="显示名称（用户姓名）"
        placeholder="例如：张三 / 操作一组"
        rules={[{ required: true, message: '请输入显示名称' }]}
      />
      {!editing && (
        <ProFormText
          name="password"
          label="初始登录密码"
          placeholder="请输入至少 12 位初始密码"
          fieldProps={{ type: 'password' }}
          extra="密码长度至少 12 位，建议包含大小写字母、数字与特殊字符"
          rules={[{ required: true, min: 12, message: '初始密码至少 12 位' }]}
        />
      )}
      <ProFormText
        name="email"
        label="邮箱地址"
        placeholder="例如：user@roncin.com"
        rules={[{ type: 'email', message: '请输入有效的邮箱地址' }]}
      />
      <ProFormSearchableSelect
        name="roleIds"
        label={editing && !pendingProvider ? '当前组织角色' : '分配角色'}
        mode="multiple"
        placeholder="请选择角色"
        options={(pendingProvider ? approvalRoles : roles).map((role) => ({
          label: `${role.name} (${role.code})`,
          value: role.id,
          code: role.code,
          name: role.name,
        }))}
        rules={
          pendingProvider
            ? [{ required: true, message: '请至少分配一个角色' }]
            : undefined
        }
      />
      {editing && !pendingProvider && canReadAllUserMemberships && (
        <div style={{ marginTop: 8 }}>
          <Space
            align="center"
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              marginBottom: 12,
            }}
          >
            <div>
              <Space size={6}>
                <ApartmentOutlined style={{ color: '#1677ff' }} />
                <Text strong>组织成员关系</Text>
              </Space>
              <div>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  同一账号可加入多个组织，并在每个组织中独立配置角色和状态。
                </Text>
              </div>
            </div>
            {canManageUserMemberships && (
              <Button
                size="small"
                icon={<PlusOutlined />}
                onClick={openCreateMembership}
              >
                加入组织
              </Button>
            )}
          </Space>
          <Table<API.AdminUserMembership>
            rowKey="id"
            size="small"
            loading={membershipsLoading}
            pagination={false}
            dataSource={memberships}
            columns={[
              {
                title: '组织',
                key: 'organization',
                render: (_, membership) => (
                  <div>
                    <Space size={6}>
                      <Text strong>{membership.organizationName || '-'}</Text>
                      {membership.primary && <Tag color="blue">主要</Tag>}
                    </Space>
                    <div>
                      <Text type="secondary" style={{ fontSize: 11 }}>
                        {organizationKindLabels[
                          membership.organizationKind ?? 0
                        ] ?? '组织'}{' '}
                        · {membership.organizationCode || '-'}
                      </Text>
                    </div>
                  </div>
                ),
              },
              {
                title: '角色',
                dataIndex: 'roleNames',
                render: (_, membership) =>
                  membership.roleNames?.length ? (
                    <Space wrap size={[4, 4]}>
                      {membership.roleNames.map((name) => (
                        <Tag key={name}>{name}</Tag>
                      ))}
                    </Space>
                  ) : (
                    <Text type="secondary">未分配</Text>
                  ),
              },
              {
                title: '状态',
                dataIndex: 'enabled',
                width: 72,
                render: (enabled) =>
                  enabled ? <Tag color="success">启用</Tag> : <Tag>停用</Tag>,
              },
              {
                title: '操作',
                key: 'actions',
                width: 110,
                align: 'right',
                render: (_, membership) =>
                  canManageUserMemberships ? (
                    <Space size={4}>
                      <Button
                        type="link"
                        size="small"
                        onClick={() => void openEditMembership(membership)}
                      >
                        编辑
                      </Button>
                      <Popconfirm
                        title={`确定从“${membership.organizationName || '该组织'}”移除？`}
                        description="移出后会停用并保留该组织关系、清除该组织角色并撤销在线会话，其他组织不受影响。在职用户不能移出最后一个有效组织。"
                        okText="移除"
                        cancelText="取消"
                        okButtonProps={{ danger: true }}
                        disabled={editing.id === currentUserId}
                        onConfirm={async () => {
                          if (!editing.id || !membership.id) return;
                          await adminServiceDeleteUserMembership({
                            userId: editing.id,
                            id: membership.id,
                          });
                          message.success('已从组织移除该用户');
                          await loadMemberships(editing.id);
                        }}
                      >
                        <Button
                          type="link"
                          danger
                          size="small"
                          disabled={editing.id === currentUserId}
                        >
                          移除
                        </Button>
                      </Popconfirm>
                    </Space>
                  ) : null,
              },
            ]}
          />
        </div>
      )}
      <UserMembershipModal
        open={membershipModalOpen}
        onOpenChange={setMembershipModalOpen}
        userId={editing?.id}
        membershipEditing={membershipEditing}
        memberships={memberships}
        organizations={organizations}
        membershipRoles={membershipRoles}
        onMembershipRolesChange={setMembershipRoles}
        currentUserId={currentUserId}
        onSaved={async () => {
          if (editing?.id) await loadMemberships(editing.id);
        }}
      />
    </ModalForm>
  );
}
