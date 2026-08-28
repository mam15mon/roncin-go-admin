import {
  ApartmentOutlined,
  DeleteOutlined,
  EditOutlined,
  KeyOutlined,
  MailOutlined,
  PlusOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  UserOutlined,
} from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormSwitch,
  ProFormText,
  ProTable,
} from '@ant-design/pro-components';
import { useAccess, useModel } from '@umijs/max';
import {
  Alert,
  App,
  Avatar,
  Button,
  Popconfirm,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import React, { useEffect, useRef, useState } from 'react';
import { ProFormSearchableSelect, SearchFilterTemplate } from '@/components/ui';
import {
  adminServiceAuthorizeDingTalkUser,
  adminServiceAuthorizeWeComUser,
  adminServiceCreateUser,
  adminServiceCreateUserMembership,
  adminServiceDeleteUserMembership,
  adminServiceListOrganizationRoles,
  adminServiceListOrganizations,
  adminServiceListRoles,
  adminServiceListUserMemberships,
  adminServiceListUsers,
  adminServiceResetUserPassword,
  adminServiceTerminateUser,
  adminServiceUpdateUser,
  adminServiceUpdateUserMembership,
} from '@/services/roncin/adminService';

const { Text } = Typography;

type UserFormValues = {
  username?: string;
  displayName?: string;
  password?: string;
  email?: string;
  enabled?: boolean;
  roleIds?: string[];
  organizationId?: string;
};

type UserMembershipFormValues = {
  organizationId?: string;
  roleIds?: string[];
  enabled?: boolean;
  primary?: boolean;
};

const organizationKindLabels: Record<number, string> = {
  1: '总部',
  2: '公司',
  3: '部门',
  4: '组',
};

function pendingExternalProvider(
  user?: API.AdminUser,
): 'wecom' | 'dingtalk' | undefined {
  if (user?.status !== 2) return undefined;
  if (user.wecomUserid) return 'wecom';
  if (user.dingtalkUnionid) return 'dingtalk';
  return undefined;
}

const userStatusLabels: Record<number, { text: string; color?: string }> = {
  1: { text: '在职', color: 'success' },
  2: { text: '待授权', color: 'warning' },
  3: { text: '已离职', color: 'default' },
  4: { text: '已移出本组织', color: 'default' },
  5: { text: '已停用', color: 'default' },
};

export default function UsersPanel() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const membershipFormRef = useRef<ProFormInstance | undefined>(undefined);
  const { message } = App.useApp();
  const access = useAccess();
  const { initialState } = useModel('@@initialState');
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<API.AdminUser>();
  const [resetting, setResetting] = useState<API.AdminUser>();
  const [roles, setRoles] = useState<API.AdminRole[]>([]);
  const [approvalRoles, setApprovalRoles] = useState<API.AdminRole[]>([]);
  const [organizations, setOrganizations] = useState<API.AdminOrganization[]>(
    [],
  );
  const [memberships, setMemberships] = useState<API.AdminUserMembership[]>([]);
  const [membershipsLoading, setMembershipsLoading] = useState(false);
  const [membershipModalOpen, setMembershipModalOpen] = useState(false);
  const [membershipEditing, setMembershipEditing] =
    useState<API.AdminUserMembership>();
  const [membershipRoles, setMembershipRoles] = useState<API.AdminRole[]>([]);
  const pendingProvider = pendingExternalProvider(editing);

  useEffect(() => {
    adminServiceListRoles().then((response) => setRoles(response.data ?? []));
    adminServiceListOrganizations().then((response) =>
      setOrganizations(response.data ?? []),
    );
  }, []);

  const openCreate = () => {
    setEditing(undefined);
    setMemberships([]);
    formRef.current?.resetFields();
    setModalOpen(true);
  };

  const loadMemberships = async (userId: string) => {
    setMembershipsLoading(true);
    try {
      const response = await adminServiceListUserMemberships({ userId });
      setMemberships(response.data ?? []);
    } finally {
      setMembershipsLoading(false);
    }
  };

  const openEdit = (user: API.AdminUser) => {
    setEditing(user);
    const provider = pendingExternalProvider(user);
    if (provider) {
      setApprovalRoles(roles);
      setMemberships([]);
    } else if (user.id && access.canReadAllUserMemberships) {
      void loadMemberships(user.id);
    } else {
      setMemberships([]);
    }
    setModalOpen(true);
  };

  const openCreateMembership = () => {
    setMembershipEditing(undefined);
    setMembershipRoles([]);
    membershipFormRef.current?.resetFields();
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

  const columns: ProColumns<API.AdminUser>[] = [
    {
      title: '用户',
      dataIndex: 'displayName',
      width: 220,
      render: (_, record) => {
        const initial = record.displayName
          ? record.displayName.charAt(0).toUpperCase()
          : 'U';
        return (
          <Space size={10} align="center">
            <Avatar
              size={32}
              src={record.avatarUrl}
              style={{
                backgroundColor: record.status === 1 ? '#1677ff' : '#94a3b8',
                fontSize: 14,
                fontWeight: 600,
                flexShrink: 0,
              }}
            >
              {initial}
            </Avatar>
            <div style={{ lineHeight: 1.3 }}>
              <div
                style={{
                  fontWeight: 600,
                  fontSize: 13,
                  color: 'rgba(0, 0, 0, 0.88)',
                }}
              >
                {record.displayName || '-'}
              </div>
              {record.username ? (
                <Text
                  copyable={{ text: record.username }}
                  type="secondary"
                  style={{ fontSize: 11, fontFamily: 'monospace' }}
                >
                  @{record.username}
                </Text>
              ) : (
                <Text type="secondary" style={{ fontSize: 11 }}>
                  {record.dingtalkUnionid ? '钉钉账号' : '无密码账号'}
                </Text>
              )}
            </div>
          </Space>
        );
      },
    },
    {
      title: '用户名',
      dataIndex: 'username',
      hideInTable: true,
      fieldProps: {
        placeholder: '搜索用户名',
      },
    },
    {
      title: '邮箱地址',
      dataIndex: 'email',
      width: 220,
      ellipsis: true,
      render: (_, record) =>
        record.email ? (
          <Space
            size={4}
            style={{ color: 'rgba(0, 0, 0, 0.65)', fontSize: 12 }}
          >
            <MailOutlined
              style={{ color: 'rgba(0, 0, 0, 0.45)', fontSize: 12 }}
            />
            <span>{record.email}</span>
          </Space>
        ) : (
          <Text type="secondary" style={{ fontSize: 12 }}>
            -
          </Text>
        ),
    },
    {
      title: '企业微信',
      dataIndex: 'wecomName',
      width: 210,
      search: false,
      render: (_, record) =>
        record.wecomUserid ? (
          <div style={{ lineHeight: 1.4 }}>
            <div style={{ fontSize: 13, fontWeight: 600 }}>
              {record.wecomName || '-'}
            </div>
            <Text
              type="secondary"
              style={{ fontSize: 11, fontFamily: 'monospace' }}
            >
              {record.wecomUserid}
            </Text>
          </div>
        ) : (
          <Text type="secondary" style={{ fontSize: 12 }}>
            未绑定
          </Text>
        ),
    },
    {
      title: '钉钉',
      dataIndex: 'dingtalkName',
      width: 210,
      search: false,
      render: (_, record) =>
        record.dingtalkUnionid ? (
          <div style={{ lineHeight: 1.4 }}>
            <div style={{ fontSize: 13, fontWeight: 600 }}>
              {record.dingtalkName || '-'}
            </div>
            <Text
              copyable={
                record.dingtalkUserid ? { text: record.dingtalkUserid } : false
              }
              type={record.dingtalkUserid ? undefined : 'warning'}
              style={{
                display: 'block',
                fontSize: 11,
                fontFamily: 'monospace',
              }}
            >
              {record.dingtalkUserid || '待重新登录绑定 userId'}
            </Text>
            <Text
              type="secondary"
              ellipsis={{ tooltip: record.dingtalkUnionid }}
              style={{
                display: 'block',
                maxWidth: 190,
                fontSize: 10,
                fontFamily: 'monospace',
              }}
            >
              unionId: {record.dingtalkUnionid}
            </Text>
          </div>
        ) : (
          <Text type="secondary" style={{ fontSize: 12 }}>
            未绑定
          </Text>
        ),
    },
    {
      title: '已分配角色',
      dataIndex: 'roleCodes',
      width: 240,
      search: false,
      render: (_, record) => {
        const codes = record.roleCodes ?? [];
        if (codes.length === 0) {
          return (
            <Text type="secondary" style={{ fontSize: 12 }}>
              未分配角色
            </Text>
          );
        }
        return (
          <Space wrap size={[4, 4]}>
            {codes.map((code) => {
              const matchedRole = roles.find((r) => r.code === code);
              const label = matchedRole ? matchedRole.name : code;
              return (
                <Tag
                  key={code}
                  variant="filled"
                  style={{
                    margin: 0,
                    fontSize: 11,
                    lineHeight: '20px',
                    padding: '0 6px',
                    backgroundColor: '#eff6ff',
                    color: '#1d4ed8',
                    border: '1px solid #dbeafe',
                  }}
                >
                  <SafetyCertificateOutlined
                    style={{ marginRight: 3, fontSize: 11 }}
                  />
                  {label}
                </Tag>
              );
            })}
          </Space>
        );
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 120,
      search: false,
      render: (_, record) => {
        const status = userStatusLabels[record.status ?? 0] ?? {
          text: '未知',
          color: 'default',
        };
        return <Tag color={status.color}>{status.text}</Tag>;
      },
    },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      valueType: 'dateTime',
      width: 170,
      search: false,
    },
    {
      title: '操作',
      valueType: 'option',
      width: 230,
      fixed: 'right',
      render: (_, record) => (
        <Space size={8}>
          {access.canUpdateUsers &&
            record.status !== 3 &&
            record.status !== 4 && (
              <Button
                type="link"
                size="small"
                icon={<EditOutlined />}
                style={{ padding: 0 }}
                onClick={() => openEdit(record)}
              >
                编辑
              </Button>
            )}
          {access.canResetUserPasswords && record.status === 1 && (
            <Button
              type="link"
              size="small"
              icon={<KeyOutlined />}
              style={{
                padding: 0,
                color: record.hasPassword ? '#f59e0b' : '#1677ff',
              }}
              onClick={() => setResetting(record)}
            >
              {record.hasPassword ? '重置密码' : '设置密码'}
            </Button>
          )}
          {access.canTerminateUsers &&
            record.status === 1 &&
            record.currentMembershipEnabled &&
            record.id !== initialState?.currentUser?.id && (
              <Popconfirm
                title={`确定为“${record.displayName || record.username}”办理离职？`}
                description="将停用账号和全部组织权限、撤销所有在线会话；历史业务记录与钉钉绑定会保留，返聘时需重新审批角色。"
                okText="确认离职"
                cancelText="取消"
                okButtonProps={{ danger: true }}
                onConfirm={async () => {
                  if (!record.id) return;
                  await adminServiceTerminateUser(
                    { id: record.id },
                    { id: record.id },
                  );
                  message.success('离职办理完成，账号和历史记录已保留');
                  actionRef.current?.reload();
                }}
              >
                <Button
                  type="link"
                  danger
                  size="small"
                  icon={<DeleteOutlined />}
                  style={{ padding: 0 }}
                >
                  办理离职
                </Button>
              </Popconfirm>
            )}
        </Space>
      ),
    },
  ];

  const [searchParams, setSearchParams] = useState<{ keyword?: string }>({});

  return (
    <>
      <SearchFilterTemplate
        layout="bar"
        keywordPlaceholder="搜索用户名、姓名、拼音或邮箱..."
        onSearch={(values) => {
          setSearchParams(values);
          actionRef.current?.reload();
        }}
        onReset={() => {
          setSearchParams({});
          actionRef.current?.reload();
        }}
        extraRight={
          <Space size={8}>
            <Button
              key="refresh"
              icon={<ReloadOutlined />}
              onClick={() => actionRef.current?.reload()}
            >
              刷新
            </Button>
            {access.canCreateUsers && (
              <Button
                key="create"
                type="primary"
                icon={<PlusOutlined />}
                onClick={openCreate}
              >
                新增用户
              </Button>
            )}
          </Space>
        }
      />
      <ProTable<API.AdminUser>
        headerTitle={
          <Space size={8}>
            <UserOutlined style={{ color: '#1677ff' }} />
            <span>成员账号列表</span>
          </Space>
        }
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        bordered
        pagination={{
          defaultPageSize: 20,
          showSizeChanger: true,
          showQuickJumper: true,
        }}
        request={async (params) => {
          const response = await adminServiceListUsers({
            page: params.current,
            pageSize: params.pageSize,
            keyword: searchParams.keyword,
          });
          return {
            data: response.data ?? [],
            success: response.success ?? true,
            total: response.total ?? 0,
          };
        }}
        search={false}
        toolBarRender={false}
      />

      {/* Create / Edit User Modal */}
      <ModalForm<UserFormValues>
        title={
          editing
            ? `编辑用户：${editing.displayName || editing.username}`
            : '新增用户'
        }
        open={modalOpen}
        formRef={formRef}
        initialValues={
          editing
            ? {
                ...editing,
                organizationId: pendingProvider
                  ? initialState?.currentUser?.currentOrganization?.id
                  : undefined,
              }
            : undefined
        }
        modalProps={{
          destroyOnClose: true,
          width:
            editing && !pendingProvider && access.canReadAllUserMemberships
              ? 880
              : 560,
          onCancel: () => setModalOpen(false),
        }}
        onOpenChange={setModalOpen}
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
          setModalOpen(false);
          actionRef.current?.reload();
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
        {editing && !pendingProvider && access.canReadAllUserMemberships && (
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
              {access.canManageUserMemberships && (
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
                    access.canManageUserMemberships ? (
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
                          disabled={
                            editing.id === initialState?.currentUser?.id
                          }
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
                            disabled={
                              editing.id === initialState?.currentUser?.id
                            }
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
      </ModalForm>

      <ModalForm<UserMembershipFormValues>
        title={membershipEditing ? '编辑组织成员关系' : '加入其他组织'}
        open={membershipModalOpen}
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
          onCancel: () => setMembershipModalOpen(false),
        }}
        onOpenChange={setMembershipModalOpen}
        onFinish={async (values) => {
          if (!editing?.id) return false;
          if (membershipEditing?.id) {
            await adminServiceUpdateUserMembership(
              { userId: editing.id, id: membershipEditing.id },
              {
                userId: editing.id,
                id: membershipEditing.id,
                roleIds: values.roleIds ?? [],
                enabled: values.enabled ?? true,
                primary: values.primary ?? false,
              },
            );
            message.success('组织成员关系已更新');
          } else {
            await adminServiceCreateUserMembership(
              { userId: editing.id },
              {
                userId: editing.id,
                organizationId: values.organizationId ?? '',
                roleIds: values.roleIds ?? [],
                primary: values.primary ?? false,
              },
            );
            message.success('用户已加入组织');
          }
          setMembershipModalOpen(false);
          await loadMemberships(editing.id);
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
              setMembershipRoles(response.data ?? []);
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
            disabled={
              editing?.id === initialState?.currentUser?.id &&
              membershipEditing.enabled
            }
          />
        )}
      </ModalForm>

      {/* Set / Reset Password Modal */}
      <ModalForm<{ username?: string; password?: string }>
        title={
          resetting?.hasPassword
            ? `重置登录密码：${resetting?.displayName || resetting?.username || ''}`
            : `设置登录账密：${resetting?.displayName || resetting?.dingtalkName || resetting?.wecomName || ''}`
        }
        open={Boolean(resetting)}
        initialValues={
          resetting?.username ? { username: resetting.username } : undefined
        }
        modalProps={{
          destroyOnClose: true,
          width: 500,
          onCancel: () => setResetting(undefined),
        }}
        onOpenChange={(open) => {
          if (!open) setResetting(undefined);
        }}
        onFinish={async (values) => {
          if (!resetting?.id) return false;
          await adminServiceResetUserPassword(
            { id: resetting.id },
            {
              id: resetting.id,
              password: values.password ?? '',
              username: values.username?.trim() || undefined,
            },
          );
          message.success(
            resetting.hasPassword
              ? '密码已重置，该用户现有在线会话已全部失效'
              : '登录账密已设置，该用户现可使用用户名与密码登录',
          );
          setResetting(undefined);
          actionRef.current?.reload();
          return true;
        }}
      >
        <Alert
          showIcon
          type={resetting?.hasPassword ? 'warning' : 'info'}
          title={resetting?.hasPassword ? '重置密码安全须知' : '设置登录账密'}
          description={
            resetting?.hasPassword
              ? '密码重置成功后，该用户的旧密码将立即失效，当前所有在线登录会话将被强制退出。'
              : '为该第三方账号设置专属登录用户名与密码后，用户可在未携带手机或扫码不便时使用账密备用登录。'
          }
          style={{ marginBottom: 16 }}
        />
        {(!resetting?.username || !resetting.hasPassword) && (
          <ProFormText
            name="username"
            label="登录用户名"
            placeholder="例如：zhangsan 或 logistics_op"
            fieldProps={{ maxLength: 64 }}
            rules={[
              { required: !resetting?.username, message: '请输入登录用户名' },
              { min: 3, max: 64, message: '用户名长度需在 3 至 64 个字符之间' },
              {
                pattern: /^[a-z0-9_.-]+$/,
                message: '用户名仅支持小写字母、数字、点号、下划线及连字符',
              },
            ]}
          />
        )}
        {resetting?.username && resetting.hasPassword && (
          <div style={{ marginBottom: 12 }}>
            <Text type="secondary" style={{ fontSize: 12 }}>
              目标账号：
            </Text>
            <Text strong style={{ marginLeft: 4, fontFamily: 'monospace' }}>
              {resetting?.username}
            </Text>
          </div>
        )}
        <ProFormText
          name="password"
          label={resetting?.hasPassword ? '新登录密码' : '初始登录密码'}
          placeholder="请输入至少 12 位新密码"
          fieldProps={{ type: 'password' }}
          extra="密码至少 12 位，设置后请及时通知用户"
          rules={[{ required: true, min: 12, message: '密码至少 12 位' }]}
        />
      </ModalForm>
    </>
  );
}
