import {
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
  canUpdateUsers: boolean;
  canResetUserPasswords: boolean;
  canTerminateUsers: boolean;
  currentUserId?: string;
  onEdit: (user: API.AdminUser) => void;
  onResetPassword: (user: API.AdminUser) => void;
  onTerminate: (user: API.AdminUser) => Promise<void>;
}

export function buildUserColumns({
  roles,
  canUpdateUsers,
  canResetUserPasswords,
  canTerminateUsers,
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
    {
      title: '操作',
      valueType: 'option',
      width: 230,
      fixed: 'right',
      render: (_, record) => (
        <Space size={8}>
          {canUpdateUsers && record.status !== 3 && record.status !== 4 && (
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
  ];
}
