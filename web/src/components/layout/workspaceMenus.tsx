import {
  ApartmentOutlined,
  ClockCircleOutlined,
  ContactsOutlined,
  DatabaseOutlined,
  FormOutlined,
  GlobalOutlined,
  HistoryOutlined,
  NotificationOutlined,
  NumberOutlined,
  SafetyCertificateOutlined,
  SettingOutlined,
  ShopOutlined,
  UserOutlined,
} from '@ant-design/icons';
import type { MenuDataItem } from '@ant-design/pro-components';
import type { ReactNode } from 'react';
import type { AccessState } from '@/router/routeTypes';

export type WorkspaceMenuItem = { key: string; label: string; icon: ReactNode };
export type WorkspaceMenus = {
  settingsItems: WorkspaceMenuItem[];
  enterpriseResourceItems: WorkspaceMenuItem[];
};

/** 桌面快捷菜单和移动Drawer共用同一权限过滤，不改变业务详情hideInMenu。 */
export function buildWorkspaceMenus(access: AccessState): WorkspaceMenus {
  const settingsItems: WorkspaceMenuItem[] = [];

  if (access?.canPreviewNotifications) {
    settingsItems.push({
      key: '/admin/notification-test',
      icon: <NotificationOutlined />,
      label: '通知测试',
    });
  }

  if (access?.canManageOrganizations || access?.canAccessPlatform) {
    settingsItems.push({
      key: '/admin?tab=organizations',
      icon: <ApartmentOutlined />,
      label: '组织架构',
    });
  }

  if (access?.canManageUsers || access?.canAccessPlatform) {
    settingsItems.push({
      key: '/admin?tab=users',
      icon: <UserOutlined />,
      label: '用户人员',
    });
  }

  if (access?.canManageRoles || access?.canAccessPlatform) {
    settingsItems.push({
      key: '/admin?tab=roles',
      icon: <SafetyCertificateOutlined />,
      label: '角色权限',
    });
  }

  if (access?.canReadMasterDataNumberRules || access?.canAccessPlatform) {
    settingsItems.push({
      key: '/admin?tab=number-rules',
      icon: <NumberOutlined />,
      label: '单据规则',
    });
  }

  if (access?.canReadMasterData) {
    settingsItems.push({
      key: '/master-data',
      icon: <DatabaseOutlined />,
      label: '主数据',
    });
  }

  if (access?.canReadAudit) {
    settingsItems.push({
      key: '/admin?tab=audit',
      icon: <HistoryOutlined />,
      label: '审计日志',
    });
  }

  if (access?.canReadTasks) {
    settingsItems.push({
      key: '/admin?tab=background-tasks',
      icon: <ClockCircleOutlined />,
      label: '后台任务',
    });
  }

  const enterpriseResourceItems: WorkspaceMenuItem[] = [];

  if (access?.canReadPartners) {
    enterpriseResourceItems.push({
      key: '/partners/customers',
      icon: <ContactsOutlined />,
      label: '客户',
    });
    enterpriseResourceItems.push({
      key: '/partners/suppliers',
      icon: <ShopOutlined />,
      label: '供应商',
    });
    enterpriseResourceItems.push({
      key: '/partners/foreign-agents',
      icon: <GlobalOutlined />,
      label: '国外代理',
    });
  }

  if (access?.canReadEnterpriseResources) {
    enterpriseResourceItems.push({
      key: '/enterprise-resources/config',
      icon: <FormOutlined />,
      label: '配置管理',
    });
  }

  return { settingsItems, enterpriseResourceItems };
}

export function toMobileWorkspaceMenus(menus: WorkspaceMenus): MenuDataItem[] {
  return [
    {
      key: 'workspace-enterprise',
      name: '企业资源',
      icon: <ContactsOutlined />,
      items: menus.enterpriseResourceItems,
    },
    {
      key: 'workspace-settings',
      name: '设置中心',
      icon: <SettingOutlined />,
      items: menus.settingsItems,
    },
  ]
    .filter((group) => group.items.length > 0)
    .map(({ items, ...group }) => ({
      ...group,
      children: items.map((item) => ({
        key: item.key,
        path: item.key,
        name: item.label,
        icon: item.icon,
      })),
    }));
}

/** 只高亮实际query，不复制设置页按可读权限选择默认分节的规则。 */
export function resolveWorkspaceMenuKey(
  pathname: string,
  search: string,
  menus: WorkspaceMenus,
): string | undefined {
  const tab = new URLSearchParams(search).get('tab');
  if (pathname === '/admin' && !tab) return undefined;
  const target = pathname === '/admin' ? `/admin?tab=${tab}` : pathname;
  return [...menus.settingsItems, ...menus.enterpriseResourceItems].find(
    (item) => item.key === target,
  )?.key;
}
