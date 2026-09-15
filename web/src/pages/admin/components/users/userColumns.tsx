import {
  BankOutlined,
  DeleteOutlined,
  EditOutlined,
  KeyOutlined,
  MailOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons';
import type { ProColumns } from '@ant-design/pro-components';
import { Avatar, Button, Popconfirm, Space, Tag, Typography } from 'antd';
import { adminUserStatusMeta, statusTag } from '@/constants/statusMeta';

const { Text } = Typography;

interface UserColumnsDeps {
  roles: API.AdminRole[];
  // 是否展示操作列（编辑/重置密码/办理离职）；离职页签隐藏整列，展示列保留。
  showActions?: boolean;
  canUpdateUsers: boolean;
  canResetUserPasswords: boolean;
  canTerminateUsers: boolean;
  canReadAllUserMemberships: boolean;
  currentUserId?: string;
  onEdit: (user: API.AdminUser) => void;
  onResetPassword: (user: API.AdminUser) => void;
  onTerminate: (user: API.AdminUser) => Promise<void>;
}

export function buildUserColumns({
  roles,
  showActions = true,
  canUpdateUsers,
  canResetUserPasswords,
  canTerminateUsers,
  canReadAllUserMemberships,
  currentUserId,
  onEdit,
  onResetPassword,
  onTerminate,
}: UserColumnsDeps): ProColumns<API.AdminUser>[] {
  return [
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
    ...(canReadAllUserMemberships
      ? [
          {
            title: '所属组织',
            dataIndex: 'organizations',
            width: 200,
            search: false,
            render: (_: unknown, record: API.AdminUser) => {
              const organizations = record.organizations ?? [];
              if (organizations.length === 0) {
                return (
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    -
                  </Text>
                );
              }
              const primary =
                organizations.find((item) => item.primary) ?? organizations[0];
              const label =
                organizations.length > 1
                  ? `${primary.organizationName} 等 ${organizations.length} 个组织`
                  : primary.organizationName;
              return (
                <Space size={4} style={{ minWidth: 0 }}>
                  <BankOutlined
                    style={{ color: '#1677ff', fontSize: 12, flexShrink: 0 }}
                  />
                  <Text
                    style={{
                      fontSize: 12,
                      maxWidth: 150,
                      display: 'inline-block',
                    }}
                    ellipsis={{
                      tooltip: organizations
                        .map((item) => item.organizationName)
                        .join('、'),
                    }}
                  >
                    {label}
                  </Text>
                </Space>
              );
            },
          },
        ]
      : []),
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
        // 角色名由后端按 role_codes 下标对应返回；跨组织角色不在当前
        // 角色字典内时仍能显示名称，仅在后端未返回时回退角色码。
        const names = record.roleNames ?? [];
        return (
          <Space wrap size={[4, 4]}>
            {codes.map((code, index) => {
              const matchedRole = roles.find((r) => r.code === code);
              const label = names[index] || matchedRole?.name || code;
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
      render: (_, record) =>
        statusTag(adminUserStatusMeta, record.status ?? 0, '未知'),
    },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      valueType: 'dateTime',
      width: 170,
      search: false,
    },
    ...(showActions
      ? [
          {
            title: '操作',
            valueType: 'option' as const,
            width: 230,
            fixed: 'right' as const,
            render: (_: unknown, record: API.AdminUser) => (
              <Space size={8}>
                {canUpdateUsers &&
                  record.status !== 3 &&
                  record.status !== 4 && (
                    <Button
                      type="link"
                      size="small"
                      icon={<EditOutlined />}
                      style={{ padding: 0 }}
                      onClick={() => onEdit(record)}
                    >
                      编辑
                    </Button>
                  )}
                {canResetUserPasswords && record.status === 1 && (
                  <Button
                    type="link"
                    size="small"
                    icon={<KeyOutlined />}
                    style={{
                      padding: 0,
                      color: record.hasPassword ? '#f59e0b' : '#1677ff',
                    }}
                    onClick={() => onResetPassword(record)}
                  >
                    {record.hasPassword ? '重置密码' : '设置密码'}
                  </Button>
                )}
                {canTerminateUsers &&
                  record.status === 1 &&
                  record.currentMembershipEnabled &&
                  record.id !== currentUserId && (
                    <Popconfirm
                      title={`确定为“${record.displayName || record.username}”办理离职？`}
                      description="将停用账号和全部组织权限、撤销所有在线会话；历史业务记录与钉钉绑定会保留，返聘时需重新审批角色。"
                      okText="确认离职"
                      cancelText="取消"
                      okButtonProps={{ danger: true }}
                      onConfirm={() => onTerminate(record)}
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
        ]
      : []),
  ];
}
