import {
  LogoutOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons';
import { history, useModel } from '@umijs/max';
import { Avatar, Button, Divider, Spin, Tag } from 'antd';
import React, { startTransition } from 'react';
import { authServiceLogout } from '@/services/roncin/authService';
import HeaderDropdown from '../HeaderDropdown';

const ROLE_LABELS: Record<string, string> = {
  super_admin: '超级管理员',
  admin: '系统管理员',
  org_admin: '组织管理员',
  manager: '业务主管',
  operator: '调度操作员',
  sales: '销售专员',
  finance: '财务专员',
};

type AvatarDropdownProps = { children?: React.ReactNode };

export const AvatarDropdown: React.FC<AvatarDropdownProps> = () => {
  const { initialState, setInitialState } = useModel('@@initialState');

  if (!initialState?.currentUser) return <Spin size="small" />;

  const handleLogout = async () => {
    await authServiceLogout({});
    startTransition(() => {
      setInitialState((state) => ({ ...state, currentUser: undefined }));
    });
    history.replace('/user/login');
  };

  const displayName =
    initialState.currentUser.displayName ||
    initialState.currentUser.username ||
    '用户';
  const username = initialState.currentUser.username || '';
  const roleScopes = initialState.currentUser.roleScopes || [];
  const firstRoleCode = roleScopes[0]?.roleCode;
  const primaryRole =
    (firstRoleCode && ROLE_LABELS[firstRoleCode]) ||
    firstRoleCode ||
    '系统用户';
  const avatarLetter = displayName.slice(0, 1).toUpperCase();

  const dropdownContent = (
    <div
      style={{
        width: 220,
        backgroundColor: '#ffffff',
        borderRadius: 8,
        boxShadow:
          '0 6px 16px 0 rgba(0, 0, 0, 0.08), 0 3px 6px -4px rgba(0, 0, 0, 0.12)',
        border: '1px solid #f0f0f0',
        padding: '12px',
      }}
    >
      {/* 顶部个人基本信息 */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 10,
          marginBottom: 8,
        }}
      >
        <Avatar
          size={36}
          style={{
            backgroundColor: '#1677ff',
            color: '#fff',
            fontWeight: 600,
            fontSize: 15,
            flexShrink: 0,
          }}
        >
          {avatarLetter}
        </Avatar>
        <div style={{ minWidth: 0, flex: 1 }}>
          <div
            style={{
              fontWeight: 600,
              fontSize: 14,
              color: '#262626',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {displayName}
          </div>
          {username && (
            <div
              style={{
                fontSize: 12,
                color: '#8c8c8c',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}
            >
              @{username}
            </div>
          )}
        </div>
      </div>

      {/* 角色标识 */}
      <div style={{ marginBottom: 10 }}>
        <Tag
          color="blue"
          icon={<SafetyCertificateOutlined style={{ fontSize: 11 }} />}
          style={{
            fontSize: 11,
            padding: '1px 6px',
            margin: 0,
            borderRadius: 4,
          }}
        >
          {primaryRole}
        </Tag>
      </div>

      <Divider style={{ margin: '8px 0' }} />

      {/* 退出登录按钮 */}
      <Button
        type="text"
        danger
        icon={<LogoutOutlined />}
        onClick={handleLogout}
        style={{
          width: '100%',
          textAlign: 'left',
          justifyContent: 'flex-start',
          height: 32,
          padding: '0 8px',
          borderRadius: 6,
          fontSize: 13,
        }}
      >
        退出登录
      </Button>
    </div>
  );

  return (
    <HeaderDropdown
      placement="bottomRight"
      dropdownRender={() => dropdownContent}
    >
      <div
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 8,
          padding: '4px 8px',
          borderRadius: 6,
          cursor: 'pointer',
          transition: 'all 0.2s',
          height: 32,
        }}
        className="roncin-avatar-trigger"
      >
        <Avatar
          size={24}
          style={{
            backgroundColor: '#1677ff',
            color: '#ffffff',
            fontSize: 12,
            fontWeight: 600,
          }}
        >
          {avatarLetter}
        </Avatar>
        <span
          style={{
            fontSize: 13,
            fontWeight: 500,
            color: 'rgba(0, 0, 0, 0.85)',
          }}
        >
          {displayName}
        </span>
      </div>
    </HeaderDropdown>
  );
};
