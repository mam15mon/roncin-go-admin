import {
  DeleteOutlined,
  EditOutlined,
  KeyOutlined,
  MailOutlined,
  PlusOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  UserOutlined,
} from '@ant-design/icons';
import type { ActionType, ProColumns, ProFormInstance } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
  ProTable,
} from '@ant-design/pro-components';
import { Alert, App, Avatar, Button, Popconfirm, Space, Tag, Typography } from 'antd';
import { useAccess, useModel } from '@umijs/max';
import React, { useEffect, useRef, useState } from 'react';
import {
  adminServiceAuthorizeDingTalkUser,
  adminServiceAuthorizeWeComUser,
  adminServiceCreateUser,
  adminServiceDeleteUser,
  adminServiceListOrganizationRoles,
  adminServiceListOrganizations,
  adminServiceListRoles,
  adminServiceListUsers,
  adminServiceResetUserPassword,
  adminServiceUpdateUser,
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

function pendingExternalProvider(
  user?: API.AdminUser,
): 'wecom' | 'dingtalk' | undefined {
  if (!user || user.enabled) return undefined;
  if (user.wecomUserid) return 'wecom';
  if (user.dingtalkUnionid) return 'dingtalk';
  return undefined;
}

function hasExternalIdentity(user: API.AdminUser): boolean {
  return Boolean(user.wecomUserid || user.dingtalkUnionid);
}

export default function UsersPanel() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const { message } = App.useApp();
  const access = useAccess();
  const { initialState } = useModel('@@initialState');
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<API.AdminUser>();
  const [resetting, setResetting] = useState<API.AdminUser>();
  const [roles, setRoles] = useState<API.AdminRole[]>([]);
  const [approvalRoles, setApprovalRoles] = useState<API.AdminRole[]>([]);
  const [organizations, setOrganizations] = useState<API.AdminOrganization[]>([]);
  const pendingProvider = pendingExternalProvider(editing);

  useEffect(() => {
    adminServiceListRoles().then((response) => setRoles(response.data ?? []));
    adminServiceListOrganizations().then((response) => setOrganizations(response.data ?? []));
  }, []);

  const openCreate = () => {
    setEditing(undefined);
    formRef.current?.resetFields();
    setModalOpen(true);
  };

  const openEdit = (user: API.AdminUser) => {
    setEditing(user);
    if (pendingExternalProvider(user)) {
      setApprovalRoles(roles);
    }
    setModalOpen(true);
  };

  const columns: ProColumns<API.AdminUser>[] = [
    {
      title: '用户',
      dataIndex: 'displayName',
      width: 220,
      render: (_, record) => {
        const initial = record.displayName ? record.displayName.charAt(0).toUpperCase() : 'U';
        return (
          <Space size={10} align="center">
            <Avatar
              size={32}
              style={{
                backgroundColor: record.enabled ? '#1677ff' : '#94a3b8',
                fontSize: 14,
                fontWeight: 600,
                flexShrink: 0,
              }}
            >
              {initial}
            </Avatar>
            <div style={{ lineHeight: 1.3 }}>
              <div style={{ fontWeight: 600, fontSize: 13, color: 'rgba(0, 0, 0, 0.88)' }}>
                {record.displayName || '-'}
              </div>
              <Text
                copyable={{ text: record.username }}
                type="secondary"
                style={{ fontSize: 11, fontFamily: 'monospace' }}
              >
                @{record.username}
              </Text>
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
          <Space size={4} style={{ color: 'rgba(0, 0, 0, 0.65)', fontSize: 12 }}>
            <MailOutlined style={{ color: 'rgba(0, 0, 0, 0.45)', fontSize: 12 }} />
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
            <div style={{ fontSize: 13, fontWeight: 600 }}>{record.wecomName || '-'}</div>
            <Text type="secondary" style={{ fontSize: 11, fontFamily: 'monospace' }}>
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
              type="secondary"
              style={{ fontSize: 11, fontFamily: 'monospace' }}
            >
              {record.dingtalkUnionid}
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
                  bordered={false}
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
                  <SafetyCertificateOutlined style={{ marginRight: 3, fontSize: 11 }} />
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
      dataIndex: 'enabled',
      width: 100,
      valueEnum: {
        true: { text: '启用' },
        false: { text: '停用' },
      },
      render: (_, record) =>
        record.enabled ? (
          <Tag color="success">启用</Tag>
        ) : hasExternalIdentity(record) ? (
          <Tag color="warning">待授权</Tag>
        ) : (
          <Tag color="default">停用</Tag>
        ),
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
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            style={{ padding: 0 }}
            onClick={() => openEdit(record)}
          >
            编辑
          </Button>
          <Button
            type="link"
            size="small"
            icon={<KeyOutlined />}
            style={{ padding: 0, color: '#f59e0b' }}
            onClick={() => setResetting(record)}
          >
            重置密码
          </Button>
          {access.canDeleteUsers &&
            record.id !== initialState?.currentUser?.id && (
              <Popconfirm
                title={`确定删除员工“${record.displayName || record.username}”？`}
                description="将从当前组织移除该员工并撤销其组织会话，账号及历史业务记录仍会保留。"
                okText="删除"
                cancelText="取消"
                okButtonProps={{ danger: true }}
                onConfirm={async () => {
                  if (!record.id) return;
                  await adminServiceDeleteUser({ id: record.id });
                  message.success('员工已从当前组织删除');
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
                  删除
                </Button>
              </Popconfirm>
            )}
        </Space>
      ),
    },
  ];

  return (
    <>
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
            keyword: params.keyword || params.username || params.displayName,
          });
          return {
            data: response.data ?? [],
            success: response.success ?? true,
            total: response.total ?? 0,
          };
        }}
        search={{
          labelWidth: 'auto',
          defaultCollapsed: false,
          span: 8,
        }}
        toolBarRender={() => [
          <Button
            key="refresh"
            icon={<ReloadOutlined />}
            onClick={() => actionRef.current?.reload()}
          >
            刷新
          </Button>,
          <Button
            key="create"
            type="primary"
            icon={<PlusOutlined />}
            onClick={openCreate}
          >
            新增用户
          </Button>,
        ]}
      />

      {/* Create / Edit User Modal */}
      <ModalForm<UserFormValues>
        title={editing ? `编辑用户：${editing.displayName || editing.username}` : '新增用户'}
        open={modalOpen}
        formRef={formRef}
        initialValues={
          editing
            ? {
                ...editing,
                organizationId: pendingProvider ? initialState?.currentUser?.currentOrganization?.id : undefined,
              }
            : undefined
        }
        modalProps={{
          destroyOnClose: true,
          width: 560,
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
                enabled: values.enabled ?? true,
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
            message={`${pendingProvider === 'wecom' ? '企业微信' : '钉钉'}成员 ${pendingProvider === 'wecom' ? editing?.wecomName || editing?.displayName : editing?.dingtalkName || editing?.displayName} 已完成身份登记`}
            description="请分配至少一个角色并启用账号。启用后，该成员再次扫码即可登录。"
            style={{ marginBottom: 16 }}
          />
        )}
        {pendingProvider && (
          <ProFormSelect
            name="organizationId"
            label="所属组织"
            placeholder="请选择公司、部门或组"
            options={organizations.map((organization) => ({
              label: `${organization.name} (${organization.code})`,
              value: organization.id,
            }))}
            rules={[{ required: true, message: '请选择所属组织' }]}
            fieldProps={{
              onChange: async (organizationId: string) => {
                formRef.current?.setFieldValue('roleIds', []);
                const response = await adminServiceListOrganizationRoles({ organizationId });
                setApprovalRoles(response.data ?? []);
              },
            }}
          />
        )}
        <ProFormText
          name="username"
          label="用户名（登录账号）"
          placeholder="例如：zhangsan 或 logistics_op"
          disabled={Boolean(editing)}
          rules={[
            { required: true, message: '请输入用户名' },
            { pattern: /^[A-Za-z0-9_.-]+$/, message: '用户名仅支持英文字母、数字、点号、下划线及连字符' },
          ]}
        />
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
        <ProFormSelect
          name="roleIds"
          label="分配角色"
          mode="multiple"
          placeholder="请选择角色"
          options={(pendingProvider ? approvalRoles : roles).map((role) => ({
            label: `${role.name} (${role.code})`,
            value: role.id,
          }))}
          rules={
            pendingProvider
              ? [{ required: true, message: '请至少分配一个角色' }]
              : undefined
          }
        />
        {editing && !pendingProvider && (
          <ProFormSwitch
            name="enabled"
            label="账号状态"
            extra="停用后用户将无法登录系统或调用业务接口"
          />
        )}
      </ModalForm>

      {/* Reset Password Modal */}
      <ModalForm<{ password?: string }>
        title={`重置登录密码：${resetting?.displayName || resetting?.username || ''}`}
        open={Boolean(resetting)}
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
            { id: resetting.id, password: values.password ?? '' },
          );
          message.success('密码已重置，该用户现有在线会话已全部失效');
          setResetting(undefined);
          return true;
        }}
      >
        <Alert
          showIcon
          type="warning"
          message="重置密码安全须知"
          description="密码重置成功后，该用户的旧密码将立即失效，当前所有在线登录会话将被强制退出。"
          style={{ marginBottom: 16 }}
        />
        <div style={{ marginBottom: 12 }}>
          <Text type="secondary" style={{ fontSize: 12 }}>
            目标账号：
          </Text>
          <Text strong style={{ marginLeft: 4, fontFamily: 'monospace' }}>
            {resetting?.username}
          </Text>
        </div>
        <ProFormText
          name="password"
          label="新登录密码"
          placeholder="请输入至少 12 位新密码"
          fieldProps={{ type: 'password' }}
          extra="新密码至少 12 位，重置后请及时通知用户"
          rules={[{ required: true, min: 12, message: '新密码至少 12 位' }]}
        />
      </ModalForm>
    </>
  );
}
