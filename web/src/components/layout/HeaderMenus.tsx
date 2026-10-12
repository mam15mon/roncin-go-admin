import { DownOutlined } from '@ant-design/icons';
import type { MenuProps } from 'antd';
import { Button, Space } from 'antd';
import React, { useMemo } from 'react';
import { useLocation } from 'react-router';
import { useAccess } from '@/app/access';
import { history } from '@/router/history';
import HeaderDropdown from '../HeaderDropdown';
import { buildWorkspaceMenus, resolveWorkspaceMenuKey } from './workspaceMenus';

export interface HeaderMenusProps {
  className?: string;
}

/**
 * 顶栏下拉菜单组件（设置中心、企业资源）
 * 与侧边菜单并存，根据 access 权限动态展示菜单项
 */
export const HeaderMenus: React.FC<HeaderMenusProps> = ({ className }) => {
  const access = useAccess();
  const location = useLocation();

  const handleMenuClick: MenuProps['onClick'] = ({ key }) => {
    if (typeof key === 'string' && key.startsWith('/')) {
      history.push(key);
    }
  };

  const menus = useMemo(() => buildWorkspaceMenus(access), [access]);
  const { settingsItems, enterpriseResourceItems } = menus;
  const selectedKey = resolveWorkspaceMenuKey(
    location.pathname,
    location.search,
    menus,
  );
  const hasSettings = Boolean(settingsItems && settingsItems.length > 0);
  const hasEnterprise = Boolean(
    enterpriseResourceItems && enterpriseResourceItems.length > 0,
  );

  if (!hasSettings && !hasEnterprise) {
    return null;
  }

  const isSettingsActive =
    location.pathname === '/master-data' ||
    location.pathname.startsWith('/master-data/') ||
    location.pathname === '/admin' ||
    location.pathname.startsWith('/admin/');

  const isEnterpriseActive =
    location.pathname === '/partners' ||
    location.pathname.startsWith('/partners/') ||
    location.pathname.startsWith('/enterprise-resources/');

  return (
    <Space
      size={4}
      className={`roncin-header-menus ${className || ''}`.trim()}
      align="center"
    >
      {hasEnterprise && (
        <HeaderDropdown
          keyboardAccessible
          placement="bottomRight"
          menu={{
            items: enterpriseResourceItems,
            onClick: handleMenuClick,
            selectedKeys: selectedKey ? [selectedKey] : [],
          }}
        >
          <Button
            type="text"
            size="small"
            className={`roncin-header-menu-btn ${
              isEnterpriseActive ? 'active' : ''
            }`}
          >
            <span>企业资源</span>
            <DownOutlined
              aria-hidden="true"
              className="roncin-header-menu-icon"
            />
          </Button>
        </HeaderDropdown>
      )}

      {hasSettings && (
        <HeaderDropdown
          keyboardAccessible
          placement="bottomRight"
          menu={{
            items: settingsItems,
            onClick: handleMenuClick,
            selectedKeys: selectedKey ? [selectedKey] : [],
          }}
        >
          <Button
            type="text"
            size="small"
            className={`roncin-header-menu-btn ${
              isSettingsActive ? 'active' : ''
            }`}
          >
            <span>设置中心</span>
            <DownOutlined
              aria-hidden="true"
              className="roncin-header-menu-icon"
            />
          </Button>
        </HeaderDropdown>
      )}
    </Space>
  );
};
