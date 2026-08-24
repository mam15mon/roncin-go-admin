import {
  AccountBookOutlined,
  ApartmentOutlined,
  ClockCircleOutlined,
  ContactsOutlined,
  DatabaseOutlined,
  DownOutlined,
  GlobalOutlined,
  HistoryOutlined,
  SafetyCertificateOutlined,
  ShopOutlined,
  UserOutlined,
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

  // 设置中心下拉项（直达组织、人员、角色、主数据等）
  const settingsItems = useMemo<MenuProps['items']>(() => {
    const items: MenuProps['items'] = [];

    if (access?.canManageOrganizations || access?.canAccessPlatform) {
      items.push({
        key: '/admin?tab=organizations',
        icon: <ApartmentOutlined />,
        label: '组织架构',
      });
    }

    if (access?.canManageUsers || access?.canAccessPlatform) {
      items.push({
        key: '/admin?tab=users',
        icon: <UserOutlined />,
        label: '用户人员',
      });
    }

    if (access?.canManageRoles || access?.canAccessPlatform) {
      items.push({
        key: '/admin?tab=roles',
        icon: <SafetyCertificateOutlined />,
        label: '角色权限',
      });
    }

    if (access?.canReadMasterData) {
      items.push({
        key: '/master-data',
        icon: <DatabaseOutlined />,
        label: '主数据',
      });
    }

    if (access?.canReadFeeSettings) {
      items.push({
        key: '/finance/fee-settings',
        icon: <AccountBookOutlined />,
        label: '费用设置',
      });
    }

    if (access?.canReadAudit) {
      items.push({
        key: '/admin?tab=audit',
        icon: <HistoryOutlined />,
        label: '审计日志',
      });
    }

    if (access?.canReadTasks) {
      items.push({
        key: '/admin?tab=background-tasks',
        icon: <ClockCircleOutlined />,
        label: '后台任务',
      });
    }

    return items;
  }, [
    access?.canManageOrganizations,
    access?.canManageUsers,
    access?.canManageRoles,
    access?.canReadMasterData,
    access?.canReadFeeSettings,
    access?.canReadAudit,
    access?.canReadTasks,
    access?.canAccessPlatform,
  ]);

  // 企业资源下拉项
  const enterpriseResourceItems = useMemo<MenuProps['items']>(() => {
    const items: MenuProps['items'] = [];

    if (access?.canReadPartners) {
      items.push({
        key: '/partners/customers',
        icon: <ContactsOutlined />,
        label: '客户',
      });
      items.push({
        key: '/partners/suppliers',
        icon: <ShopOutlined />,
        label: '供应商',
      });
      items.push({
        key: '/partners/foreign-agents',
        icon: <GlobalOutlined />,
        label: '国外代理',
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
    location.pathname === '/finance/fee-settings' ||
    location.pathname === '/admin' ||
    location.pathname.startsWith('/admin/');

  const isEnterpriseActive =
    location.pathname === '/partners' ||
    location.pathname.startsWith('/partners/');

  return (
    <Space size={4} className={className} align="center">
      {hasEnterprise && (
        <HeaderDropdown
          placement="bottomRight"
          menu={{
            items: enterpriseResourceItems,
            onClick: handleMenuClick,
            selectedKeys: [location.pathname],
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
            <DownOutlined className="roncin-header-menu-icon" />
          </Button>
        </HeaderDropdown>
      )}

      {hasSettings && (
        <HeaderDropdown
          placement="bottomRight"
          menu={{
            items: settingsItems,
            onClick: handleMenuClick,
            selectedKeys: [location.pathname],
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
            <DownOutlined className="roncin-header-menu-icon" />
          </Button>
        </HeaderDropdown>
      )}
    </Space>
  );
};
