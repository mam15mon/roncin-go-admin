import { LogoutOutlined } from '@ant-design/icons';
import { history, useModel } from '@umijs/max';
import type { MenuProps } from 'antd';
import { Spin } from 'antd';
import React, { startTransition } from 'react';
import { authServiceLogout } from '@/services/roncin/authService';
import HeaderDropdown from '../HeaderDropdown';

type AvatarDropdownProps = { children?: React.ReactNode };

export const AvatarDropdown: React.FC<AvatarDropdownProps> = ({ children }) => {
  const { initialState, setInitialState } = useModel('@@initialState');

  if (!initialState?.currentUser) return <Spin size="small" />;

  const onMenuClick: MenuProps['onClick'] = async ({ key }) => {
    if (key !== 'logout') return;

    await authServiceLogout({});
    startTransition(() => {
      setInitialState((state) => ({ ...state, currentUser: undefined }));
    });
    history.replace('/user/login');
  };

  const displayName =
    initialState.currentUser.displayName || initialState.currentUser.username;
  const username = initialState.currentUser.username;

  const items: MenuProps['items'] = [
    {
      key: 'user-info',
      disabled: true,
      label: (
        <div style={{ padding: '2px 0', color: '#0f172a' }}>
          <div style={{ fontWeight: 600, fontSize: 13 }}>{displayName}</div>
          <div style={{ fontSize: 12, color: '#64748b' }}>@{username}</div>
        </div>
      ),
    },
    { type: 'divider' },
    { key: 'logout', icon: <LogoutOutlined />, label: '退出登录' },
  ];

  return (
    <HeaderDropdown
      placement="bottomRight"
      menu={{ selectedKeys: [], onClick: onMenuClick, items }}
      arrow
    >
      {children}
    </HeaderDropdown>
  );
};
