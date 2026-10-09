import { ApartmentOutlined, PlusOutlined } from '@ant-design/icons';
import type { ProFormInstance } from '@ant-design/pro-components';
import { ModalForm, ProFormText } from '@ant-design/pro-components';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
  App,
  Button,
  Popconfirm,
  Segmented,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import React, { useEffect, useState } from 'react';
import { ProFormSearchableSelect } from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import {
  adminServiceCreateUser,
  adminServiceDeleteUserMembership,
  adminServiceListOrganizationRoles,
  adminServiceListUserMemberships,
  adminServiceUpdateUser,
} from '@/services/roncin/adminService';
import { ensureListResponse, unwrapList } from '@/utils/api';
import UserMembershipModal from './UserMembershipModal';
import {
  formatOrganizationHierarchyName,
  organizationKindLabels,
  resolveAnchorOrganizationId,
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
  canReadUserMemberships: boolean;
  canManageUserMemberships: boolean;
  canUpdateUserProfile: boolean;
  currentUserId?: string;
  onReload: () => void;
}

export default function UserFormModal({
  open,
  onOpenChange,
  editing,
  formRef,
  roles,
  organizations,
  canReadUserMemberships,
  canManageUserMemberships,
  canUpdateUserProfile,
  currentUserId,
  onReload,
}: UserFormModalProps) {
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const [membershipView, setMembershipView] = useState<'active' | 'inactive'>(
    'active',
  );
  const [membershipModalOpen, setMembershipModalOpen] = useState(false);
  const [membershipEditing, setMembershipEditing] =
    useState<API.AdminUserMembership>();
  const [membershipRoles, setMembershipRoles] = useState<API.AdminRole[]>([]);
  useEffect(() => {
    if (open) setMembershipView('active');
  }, [open, editing?.id]);

  const membershipsQuery = useQuery({
    queryKey: ['admin-users', 'memberships', editing?.id],
    enabled: Boolean(open && editing?.id && canReadUserMemberships),
    queryFn: async () =>
      unwrapList(
        ensureListResponse(
          await adminServiceListUserMemberships({ userId: editing?.id ?? '' }),
          '读取组织成员关系失败',
        ),
      ),
  });
  const memberships = membershipsQuery.data ?? [];
  const membershipsLoading = membershipsQuery.isLoading;
  const activeMemberships = memberships.filter(
    (membership) => membership.enabled,
  );
  const inactiveMemberships = memberships.filter(
    (membership) => !membership.enabled,
  );

  const anchorOrganizationId = resolveAnchorOrganizationId(memberships);
  const anchorRolesQuery = useQuery({
    queryKey: ['admin-users', 'organization-roles', anchorOrganizationId],
    enabled: Boolean(open && anchorOrganizationId),
    queryFn: async () =>
      unwrapList(
        ensureListResponse(
          await adminServiceListOrganizationRoles({
            organizationId: anchorOrganizationId ?? '',
          }),
          '读取组织角色失败',
        ),
      ),
  });
  const anchorRoles = anchorRolesQuery.data ?? [];
  const loadMemberships = async (_userId: string) => {
    await queryClient.invalidateQueries({ queryKey: ['admin-users'] });
  };
  useEffect(() => {
    if (!open || !membershipsQuery.data) return;
    const anchor = memberships.find(
      (membership) => membership.organizationId === anchorOrganizationId,
    );
    formRef.current?.setFieldValue('roleIds', anchor?.roleIds ?? []);
  }, [open, membershipsQuery.data, anchorOrganizationId, formRef]);

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
      setMembershipRoles(
        unwrapList(ensureListResponse(response, '读取组织角色失败')),
      );
    }
  };

  const membershipColumns: ColumnsType<API.AdminUserMembership> = [
    {
      title: '组织',
      key: 'organization',
      render: (_, membership) => {
        const org = organizations.find(
          (item) => item.id === membership.organizationId,
        );
        const fullName =
          formatOrganizationHierarchyName(org, organizations) ||
          membership.organizationName ||
          '-';
        return (
          <div>
            <Space size={6}>
              <Text strong>{fullName}</Text>
              {membership.primary && <Tag color="blue">主要</Tag>}
            </Space>
            <div>
              <Text type="secondary" style={{ fontSize: 11 }}>
                {organizationKindLabels[membership.organizationKind ?? 0] ??
                  '组织'}{' '}
                · {membership.organizationCode || '-'}
              </Text>
            </div>
          </div>
        );
      },
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
            {membership.enabled && (
              <Popconfirm
                title={`确定从“${membership.organizationName || '该组织'}”移除？`}
                description="移出后会停用并保留该组织关系、清除该组织角色并撤销在线会话，其他组织不受影响。在职用户不能移出最后一个有效组织。"
                okText="移除"
                cancelText="取消"
                okButtonProps={{ danger: true }}
                disabled={editing?.id === currentUserId}
                onConfirm={async () => {
                  if (!editing?.id || !membership.id) return;
                  const response = await adminServiceDeleteUserMembership({
                    userId: editing.id,
                    id: membership.id,
                  });
                  if (!response) return;
                  message.success('已从组织移除该用户');
                  await loadMemberships(editing.id);
                  onReload();
                }}
              >
                <Button
                  type="link"
                  danger
                  size="small"
                  disabled={editing?.id === currentUserId}
                >
                  移除
                </Button>
              </Popconfirm>
            )}
          </Space>
        ) : null,
    },
  ];

  const columnSettings = useColumnSettings<
    ColumnsType<API.AdminUserMembership>[number]
  >({
    tableKey: 'admin:user-memberships',
    columns: membershipColumns,
    structuralKeys: ['actions'],
  });

  return (
    <ModalForm<UserFormValues>
      title={
        editing
          ? `编辑用户：${editing.displayName || editing.username}`
          : '新增用户'
      }
      submitter={
        editing && !canUpdateUserProfile
          ? false
          : {
              submitButtonProps: {
                disabled:
                  membershipsQuery.isFetching ||
                  membershipsQuery.isError ||
                  anchorRolesQuery.isFetching ||
                  anchorRolesQuery.isError,
              },
            }
      }
      open={open}
      formRef={formRef}
      initialValues={editing}
      modalProps={{
        destroyOnHidden: true,
        width: editing && canReadUserMemberships ? 880 : 560,
        onCancel: () => onOpenChange(false),
      }}
      onOpenChange={onOpenChange}
      onFinish={async (values) => {
        if (editing && !canUpdateUserProfile) return false;
        if (editing?.id) {
          const response = await adminServiceUpdateUser(
            { id: editing.id },
            {
              id: editing.id,
              ...(!editing.dingtalkUnionid
                ? {
                    displayName: values.displayName?.trim() ?? '',
                    email: values.email?.trim() || undefined,
                  }
                : {}),
              enabled: true,
              roleIds: values.roleIds ?? [],
            },
          );
          if (!response) return false;
          message.success('用户已成功更新');
        } else {
          const response = await adminServiceCreateUser({
            username: values.username?.trim() ?? '',
            displayName: values.displayName?.trim() ?? '',
            password: values.password ?? '',
            email: values.email?.trim() || undefined,
            roleIds: values.roleIds ?? [],
          });
          if (!response) return false;
          message.success('用户已成功创建');
        }
        onOpenChange(false);
        onReload();
        return true;
      }}
    >
      {editing?.dingtalkUnionid && (
        <Alert
          type="info"
          showIcon
          title="钉钉基础资料由通讯录维护"
          description="姓名、邮箱、头像、联系方式及来源部门在钉钉维护；组织成员关系和角色在本系统配置。"
          style={{ marginBottom: 16 }}
        />
      )}
      {editing && !canUpdateUserProfile && (
        <Alert
          type="info"
          showIcon
          title={
            editing.dingtalkUnionid
              ? '仅管理本公司成员关系和角色；基础资料由钉钉维护，密码及全局状态由系统管理员维护。'
              : '仅管理本公司成员关系和角色；账号姓名、邮箱、密码及全局状态由系统管理员维护。'
          }
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
        disabled={
          Boolean(editing?.dingtalkUnionid) ||
          (Boolean(editing) && !canUpdateUserProfile)
        }
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
        disabled={
          Boolean(editing?.dingtalkUnionid) ||
          (Boolean(editing) && !canUpdateUserProfile)
        }
        name="email"
        label="邮箱地址"
        placeholder="例如：user@roncin.com"
        rules={[{ type: 'email', message: '请输入有效的邮箱地址' }]}
      />
      <ProFormSearchableSelect
        disabled={Boolean(editing) && !canUpdateUserProfile}
        name="roleIds"
        label={editing ? '当前组织角色' : '分配角色'}
        mode="multiple"
        placeholder="请选择角色"
        options={(anchorOrganizationId ? anchorRoles : roles).map((role) => ({
          label: `${role.name} (${role.code})`,
          value: role.id,
          code: role.code,
          name: role.name,
        }))}
      />
      {editing && canReadUserMemberships && (
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
            <Space size={8}>
              {columnSettings.entry}
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
          </Space>
          <Segmented<'active' | 'inactive'>
            value={membershipView}
            onChange={setMembershipView}
            options={[
              { label: `当前 (${activeMemberships.length})`, value: 'active' },
              {
                label: `已停用 (${inactiveMemberships.length})`,
                value: 'inactive',
              },
            ]}
            style={{ marginBottom: 12 }}
          />
          <Table<API.AdminUserMembership>
            rowKey="id"
            size="small"
            loading={membershipsLoading}
            pagination={false}
            dataSource={
              membershipView === 'active'
                ? activeMemberships
                : inactiveMemberships
            }
            columns={columnSettings.columns}
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
          onReload();
        }}
      />
    </ModalForm>
  );
}
