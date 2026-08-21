import {
  ContactsOutlined,
  DatabaseOutlined,
  DownOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import { history, useAccess, useLocation } from '@umijs/max';
import type { MenuProps } from 'antd';
import { Button, Space } from 'antd';
import React, { useMemo } from 'react';
import HeaderDropdown from '../HeaderDropdown';

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

  // 设置中心下拉项
  const settingsItems = useMemo<MenuProps['items']>(() => {
    const items: MenuProps['items'] = [];

    if (access?.canReadMasterData) {
      items.push({
        key: '/master-data',
        icon: <DatabaseOutlined />,
        label: '主数据',
      });
    }

    if (access?.canAccessPlatform) {
      items.push({
        key: '/admin',
        icon: <SettingOutlined />,
        label: '系统管理',
      });
    }

    return items;
  }, [access?.canReadMasterData, access?.canAccessPlatform]);

  // 企业资源下拉项
  const enterpriseResourceItems = useMemo<MenuProps['items']>(() => {
    const items: MenuProps['items'] = [];

    if (access?.canReadPartners) {
      items.push({
        key: '/partners',
        icon: <ContactsOutlined />,
        label: '客户与供应商',
      });
    }

    return items;
  }, [access?.canReadPartners]);

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
    location.pathname.startsWith('/partners/');

  return (
    <Space size={4} className={className} align="center">
      {hasSettings && (
        <HeaderDropdown
          placement="bottomLeft"
          menu={{
            items: settingsItems,
            onClick: handleMenuClick,
            selectedKeys: [location.pathname],
          }}
        >
          <Button
            type="text"
            size="small"
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 4,
              height: 32,
              padding: '0 8px',
              fontSize: 13,
              fontWeight: isSettingsActive ? 600 : 500,
              color: isSettingsActive ? '#1677ff' : '#475569',
            }}
          >
            <span>设置中心</span>
            <DownOutlined
              style={{
                fontSize: 10,
                color: isSettingsActive ? '#1677ff' : '#94a3b8',
              }}
            />
          </Button>
        </HeaderDropdown>
      )}

      {hasEnterprise && (
        <HeaderDropdown
          placement="bottomLeft"
          menu={{
            items: enterpriseResourceItems,
            onClick: handleMenuClick,
            selectedKeys: [location.pathname],
          }}
        >
          <Button
            type="text"
            size="small"
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 4,
              height: 32,
              padding: '0 8px',
              fontSize: 13,
              fontWeight: isEnterpriseActive ? 600 : 500,
              color: isEnterpriseActive ? '#1677ff' : '#475569',
            }}
          >
            <span>企业资源</span>
            <DownOutlined
              style={{
                fontSize: 10,
                color: isEnterpriseActive ? '#1677ff' : '#94a3b8',
              }}
            />
          </Button>
        </HeaderDropdown>
      )}
    </Space>
  );
};
