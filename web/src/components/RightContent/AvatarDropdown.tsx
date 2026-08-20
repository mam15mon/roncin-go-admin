import { LogoutOutlined, SkinOutlined } from '@ant-design/icons';
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
    if (key === 'theme') {
      setInitialState((state) => ({ ...state, settingDrawerOpen: true }));
      return;
    }
    if (key !== 'logout') return;

    await authServiceLogout({});
    startTransition(() => {
      setInitialState((state) => ({ ...state, currentUser: undefined }));
    });
    history.replace('/user/login');
  };

  const items: MenuProps['items'] = [
    { key: 'theme', icon: <SkinOutlined />, label: '主题设置' },
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
